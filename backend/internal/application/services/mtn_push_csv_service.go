package services

// mtn_push_csv_service.go
//
// Fallback pipeline for when the MTN push webhook API is unavailable.
// An admin uploads a CSV file containing:
//
//   msisdn, date, time, amount[, recharge_type]
//
// Each row is processed through the same ProcessMTNPush pipeline as a live
// webhook event.  The service:
//   1. Parses and validates every row
//   2. Generates a deterministic synthetic transaction_ref for idempotency
//      (SHA-256 of "CSV:<msisdn>:<date>T<time>:<amount>")
//   3. Calls ProcessMTNPush for each valid row
//   4. Writes a full audit trail to mtn_push_csv_uploads + mtn_push_csv_rows
//
// All configurable thresholds (min amount, spin rate, pulse rate) come from
// the same network_configs table used by the live webhook — no hardcoding.

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ─── DB models ────────────────────────────────────────────────────────────────

// csvUpload mirrors the mtn_push_csv_uploads table.
type csvUpload struct {
	ID            uuid.UUID  `gorm:"column:id;primaryKey"`
	UploadedBy    string     `gorm:"column:uploaded_by"`
	Filename      string     `gorm:"column:filename"`
	UploadedAt    time.Time  `gorm:"column:uploaded_at;autoCreateTime"`
	TotalRows     int        `gorm:"column:total_rows"`
	ProcessedRows int        `gorm:"column:processed_rows"`
	SkippedRows   int        `gorm:"column:skipped_rows"`
	FailedRows    int        `gorm:"column:failed_rows"`
	Status        string     `gorm:"column:status"`
	Note          string     `gorm:"column:note"`
	CompletedAt   *time.Time `gorm:"column:completed_at"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (csvUpload) TableName() string { return "mtn_push_csv_uploads" }

// csvRow mirrors the mtn_push_csv_rows table.
type csvRow struct {
	ID             uuid.UUID  `gorm:"column:id;primaryKey"`
	UploadID       uuid.UUID  `gorm:"column:upload_id"`
	RowNumber      int        `gorm:"column:row_number"`
	RawMSISDN      string     `gorm:"column:raw_msisdn"`
	RawDate        string     `gorm:"column:raw_date"`
	RawTime        string     `gorm:"column:raw_time"`
	RawAmount      string     `gorm:"column:raw_amount"`
	RechargeType   string     `gorm:"column:recharge_type"`
	MSISDN         string     `gorm:"column:msisdn"`
	RechargeAt     *time.Time `gorm:"column:recharge_at"`
	AmountNaira    float64    `gorm:"column:amount_naira"`
	Status         string     `gorm:"column:status"`
	SkipReason     string     `gorm:"column:skip_reason"`
	ErrorMsg       string     `gorm:"column:error_msg"`
	TransactionRef string     `gorm:"column:transaction_ref"`
	SpinCredits    int        `gorm:"column:spin_credits"`
	PulsePoints    int64      `gorm:"column:pulse_points"`
	DrawEntries    int        `gorm:"column:draw_entries"`
	ProcessedAt    *time.Time `gorm:"column:processed_at"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime"`
}

func (csvRow) TableName() string { return "mtn_push_csv_rows" }

// ─── Request / response types ─────────────────────────────────────────────────

// CSVUploadRequest carries the parsed upload metadata from the HTTP handler.
type CSVUploadRequest struct {
	// UploadedBy is the admin's user ID or email (extracted from JWT).
	UploadedBy string
	// Filename is the original filename (for display in the audit log).
	Filename string
	// Reader is the CSV file body.
	Reader io.Reader
	// Note is an optional admin note attached to the batch.
	Note string
}

// CSVUploadResult is returned to the HTTP handler after processing.
type CSVUploadResult struct {
	UploadID      uuid.UUID `json:"upload_id"`
	TotalRows     int       `json:"total_rows"`
	ProcessedRows int       `json:"processed_rows"`
	SkippedRows   int       `json:"skipped_rows"`
	FailedRows    int       `json:"failed_rows"`
	Status        string    `json:"status"`
}

// CSVUploadSummary is returned by GetCSVUpload for the admin UI.
type CSVUploadSummary struct {
	ID            uuid.UUID  `json:"id"`
	UploadedBy    string     `json:"uploaded_by"`
	Filename      string     `json:"filename"`
	UploadedAt    time.Time  `json:"uploaded_at"`
	TotalRows     int        `json:"total_rows"`
	ProcessedRows int        `json:"processed_rows"`
	SkippedRows   int        `json:"skipped_rows"`
	FailedRows    int        `json:"failed_rows"`
	Status        string     `json:"status"`
	Note          string     `json:"note,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

// CSVRowDetail is a single row result returned by GetCSVUploadRows.
type CSVRowDetail struct {
	RowNumber      int        `json:"row_number"`
	RawMSISDN      string     `json:"raw_msisdn"`
	RawDate        string     `json:"raw_date"`
	RawTime        string     `json:"raw_time"`
	RawAmount      string     `json:"raw_amount"`
	RechargeType   string     `json:"recharge_type"`
	Status         string     `json:"status"`
	SkipReason     string     `json:"skip_reason,omitempty"`
	ErrorMsg       string     `json:"error_msg,omitempty"`
	TransactionRef string     `json:"transaction_ref,omitempty"`
	SpinCredits    int        `json:"spin_credits,omitempty"`
	PulsePoints    int64      `json:"pulse_points,omitempty"`
	DrawEntries    int        `json:"draw_entries,omitempty"`
	ProcessedAt    *time.Time `json:"processed_at,omitempty"`
}

// ─── Service ──────────────────────────────────────────────────────────────────

// MTNPushCSVService processes admin CSV bulk uploads.
type MTNPushCSVService struct {
	db         *gorm.DB
	mtnPushSvc *MTNPushService
}

// NewMTNPushCSVService constructs the service.
func NewMTNPushCSVService(db *gorm.DB, mtnPushSvc *MTNPushService) *MTNPushCSVService {
	return &MTNPushCSVService{db: db, mtnPushSvc: mtnPushSvc}
}

// ProcessCSVUpload parses the CSV, processes every valid row through the MTN
// push pipeline, and writes a full audit trail.
//
// CSV format (header row required):
//
//	msisdn,date,time,amount[,recharge_type]
//
// date format: YYYY-MM-DD
// time format: HH:MM or HH:MM:SS  (WAT assumed)
// amount:      naira value (e.g. 1000 or 1000.00)
// recharge_type: optional; defaults to AIRTIME
//
// The function always returns a result — partial failures are recorded in the
// audit log and do not cause the whole batch to fail.
func (s *MTNPushCSVService) ProcessCSVUpload(ctx context.Context, req CSVUploadRequest) (*CSVUploadResult, error) {
	// ── 1. Create the batch header row ────────────────────────────────────────
	upload := &csvUpload{
		ID:         uuid.New(),
		UploadedBy: req.UploadedBy,
		Filename:   req.Filename,
		Status:     "PROCESSING",
		Note:       req.Note,
	}
	if err := s.db.WithContext(ctx).Create(upload).Error; err != nil {
		return nil, fmt.Errorf("create upload record: %w", err)
	}

	// ── 2. Parse CSV ──────────────────────────────────────────────────────────
	reader := csv.NewReader(req.Reader)
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // allow variable columns (recharge_type optional)

	// Read header row
	header, err := reader.Read()
	if err != nil {
		s.markUploadFailed(ctx, upload, "failed to read CSV header: "+err.Error())
		return nil, fmt.Errorf("read CSV header: %w", err)
	}
	colIdx, parseErr := parseCSVHeader(header)
	if parseErr != nil {
		s.markUploadFailed(ctx, upload, parseErr.Error())
		return nil, parseErr
	}

	// ── 3. Process each data row ──────────────────────────────────────────────
	var (
		totalRows     int
		processedRows int
		skippedRows   int
		failedRows    int
	)

	rowNum := 1 // 1-based (header = 0)
	for {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		rowNum++
		totalRows++

		row := &csvRow{
			ID:       uuid.New(),
			UploadID: upload.ID,
			RowNumber: rowNum,
		}

		if readErr != nil {
			row.Status = "FAILED"
			row.ErrorMsg = "CSV parse error: " + readErr.Error()
			failedRows++
			s.saveRow(ctx, row)
			continue
		}

		// Extract raw fields
		row.RawMSISDN = safeCol(record, colIdx.msisdn)
		row.RawDate   = safeCol(record, colIdx.date)
		row.RawTime   = safeCol(record, colIdx.time)
		row.RawAmount = safeCol(record, colIdx.amount)
		if colIdx.rechargeType >= 0 {
			row.RechargeType = strings.ToUpper(strings.TrimSpace(safeCol(record, colIdx.rechargeType)))
		}
		if row.RechargeType == "" {
			row.RechargeType = "AIRTIME"
		}

		// Parse and validate
		phone := normalisePhone(row.RawMSISDN)
		if phone == "" {
			row.Status = "FAILED"
			row.ErrorMsg = fmt.Sprintf("invalid MSISDN %q", row.RawMSISDN)
			failedRows++
			s.saveRow(ctx, row)
			continue
		}
		row.MSISDN = phone

		rechargeAt, timeErr := parseCSVDateTime(row.RawDate, row.RawTime)
		if timeErr != nil {
			row.Status = "FAILED"
			row.ErrorMsg = fmt.Sprintf("invalid date/time %q %q: %v", row.RawDate, row.RawTime, timeErr)
			failedRows++
			s.saveRow(ctx, row)
			continue
		}
		row.RechargeAt = &rechargeAt

		amount, amtErr := strconv.ParseFloat(strings.TrimSpace(row.RawAmount), 64)
		if amtErr != nil || amount <= 0 {
			row.Status = "FAILED"
			row.ErrorMsg = fmt.Sprintf("invalid amount %q", row.RawAmount)
			failedRows++
			s.saveRow(ctx, row)
			continue
		}
		row.AmountNaira = amount

		// Build a deterministic transaction ref for idempotency.
		// Format: "CSV:<msisdn>:<date>T<time>:<amount_kobo>"
		// Using SHA-256 prefix keeps it short and collision-resistant.
		refRaw := fmt.Sprintf("CSV:%s:%sT%s:%.0f",
			phone, row.RawDate, row.RawTime, amount*100)
		h := sha256.Sum256([]byte(refRaw))
		txRef := fmt.Sprintf("CSV-%x", h[:8]) // 16 hex chars
		row.TransactionRef = txRef

		// ── 4. Call the MTN push pipeline ─────────────────────────────────────
		payload := MTNPushPayload{
			TransactionRef: txRef,
			MSISDN:         phone,
			RechargeType:   row.RechargeType,
			Amount:         amount,
			Timestamp:      rechargeAt.Format(time.RFC3339),
		}

		result, procErr := s.mtnPushSvc.ProcessMTNPush(ctx, payload)
		now := time.Now()
		row.ProcessedAt = &now

		if procErr != nil {
			// Check if it's a duplicate (idempotency hit) — not a hard failure.
			if strings.Contains(procErr.Error(), "duplicate") ||
				(result != nil && result.IsDuplicate) {
				row.Status = "SKIPPED"
				row.SkipReason = "duplicate"
				skippedRows++
			} else {
				row.Status = "FAILED"
				row.ErrorMsg = procErr.Error()
				failedRows++
				log.Printf("[CSV-UPLOAD] row %d (%s ₦%.2f) failed: %v",
					rowNum, phone, amount, procErr)
			}
		} else {
			if result.IsDuplicate {
				row.Status = "SKIPPED"
				row.SkipReason = "duplicate"
				skippedRows++
			} else {
				row.Status = "OK"
				row.SpinCredits = result.SpinCredits
				row.PulsePoints = result.PulsePoints
				row.DrawEntries = result.DrawEntries
				processedRows++
			}
		}

		s.saveRow(ctx, row)
	}

	// ── 5. Update batch header ────────────────────────────────────────────────
	now := time.Now()
	finalStatus := "DONE"
	if failedRows > 0 && processedRows == 0 {
		finalStatus = "FAILED"
	} else if failedRows > 0 {
		finalStatus = "PARTIAL"
	}

	s.db.WithContext(ctx).Model(upload).Updates(map[string]interface{}{
		"total_rows":     totalRows,
		"processed_rows": processedRows,
		"skipped_rows":   skippedRows,
		"failed_rows":    failedRows,
		"status":         finalStatus,
		"completed_at":   now,
	})

	log.Printf("[CSV-UPLOAD] %s complete: %d/%d processed, %d skipped, %d failed",
		upload.ID, processedRows, totalRows, skippedRows, failedRows)

	return &CSVUploadResult{
		UploadID:      upload.ID,
		TotalRows:     totalRows,
		ProcessedRows: processedRows,
		SkippedRows:   skippedRows,
		FailedRows:    failedRows,
		Status:        finalStatus,
	}, nil
}

// GetUpload returns the summary for a single upload batch.
func (s *MTNPushCSVService) GetUpload(ctx context.Context, uploadID uuid.UUID) (*CSVUploadSummary, error) {
	var u csvUpload
	if err := s.db.WithContext(ctx).Where("id = ?", uploadID).First(&u).Error; err != nil {
		return nil, err
	}
	return &CSVUploadSummary{
		ID:            u.ID,
		UploadedBy:    u.UploadedBy,
		Filename:      u.Filename,
		UploadedAt:    u.UploadedAt,
		TotalRows:     u.TotalRows,
		ProcessedRows: u.ProcessedRows,
		SkippedRows:   u.SkippedRows,
		FailedRows:    u.FailedRows,
		Status:        u.Status,
		Note:          u.Note,
		CompletedAt:   u.CompletedAt,
	}, nil
}

// ListUploads returns recent upload batches (newest first).
func (s *MTNPushCSVService) ListUploads(ctx context.Context, limit, offset int) ([]CSVUploadSummary, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var uploads []csvUpload
	var total int64
	if err := s.db.WithContext(ctx).Model(&csvUpload{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := s.db.WithContext(ctx).
		Order("uploaded_at DESC").
		Limit(limit).Offset(offset).
		Find(&uploads).Error; err != nil {
		return nil, 0, err
	}
	out := make([]CSVUploadSummary, len(uploads))
	for i, u := range uploads {
		out[i] = CSVUploadSummary{
			ID:            u.ID,
			UploadedBy:    u.UploadedBy,
			Filename:      u.Filename,
			UploadedAt:    u.UploadedAt,
			TotalRows:     u.TotalRows,
			ProcessedRows: u.ProcessedRows,
			SkippedRows:   u.SkippedRows,
			FailedRows:    u.FailedRows,
			Status:        u.Status,
			Note:          u.Note,
			CompletedAt:   u.CompletedAt,
		}
	}
	return out, total, nil
}

// GetUploadRows returns the per-row detail for a batch (paginated).
func (s *MTNPushCSVService) GetUploadRows(ctx context.Context, uploadID uuid.UUID, limit, offset int) ([]CSVRowDetail, int64, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var rows []csvRow
	var total int64
	if err := s.db.WithContext(ctx).Model(&csvRow{}).
		Where("upload_id = ?", uploadID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := s.db.WithContext(ctx).
		Where("upload_id = ?", uploadID).
		Order("row_number ASC").
		Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]CSVRowDetail, len(rows))
	for i, r := range rows {
		out[i] = CSVRowDetail{
			RowNumber:      r.RowNumber,
			RawMSISDN:      r.RawMSISDN,
			RawDate:        r.RawDate,
			RawTime:        r.RawTime,
			RawAmount:      r.RawAmount,
			RechargeType:   r.RechargeType,
			Status:         r.Status,
			SkipReason:     r.SkipReason,
			ErrorMsg:       r.ErrorMsg,
			TransactionRef: r.TransactionRef,
			SpinCredits:    r.SpinCredits,
			PulsePoints:    r.PulsePoints,
			DrawEntries:    r.DrawEntries,
			ProcessedAt:    r.ProcessedAt,
		}
	}
	return out, total, nil
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

func (s *MTNPushCSVService) saveRow(ctx context.Context, row *csvRow) {
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		log.Printf("[CSV-UPLOAD] failed to save row %d: %v", row.RowNumber, err)
	}
}

func (s *MTNPushCSVService) markUploadFailed(ctx context.Context, upload *csvUpload, reason string) {
	s.db.WithContext(ctx).Model(upload).Updates(map[string]interface{}{
		"status":       "FAILED",
		"note":         reason,
		"completed_at": time.Now(),
	})
}

// csvColIndex holds the column positions parsed from the header row.
type csvColIndex struct {
	msisdn       int
	date         int
	time         int
	amount       int
	rechargeType int // -1 if absent
}

// parseCSVHeader maps header names to column indices.
// Accepted header names (case-insensitive):
//
//	msisdn / phone / phone_number
//	date
//	time
//	amount / recharge_amount
//	recharge_type / type (optional)
func parseCSVHeader(header []string) (csvColIndex, error) {
	idx := csvColIndex{msisdn: -1, date: -1, time: -1, amount: -1, rechargeType: -1}
	for i, h := range header {
		switch strings.ToLower(strings.TrimSpace(h)) {
		case "msisdn", "phone", "phone_number":
			idx.msisdn = i
		case "date":
			idx.date = i
		case "time":
			idx.time = i
		case "amount", "recharge_amount":
			idx.amount = i
		case "recharge_type", "type":
			idx.rechargeType = i
		}
	}
	var missing []string
	if idx.msisdn < 0  { missing = append(missing, "msisdn") }
	if idx.date < 0    { missing = append(missing, "date") }
	if idx.time < 0    { missing = append(missing, "time") }
	if idx.amount < 0  { missing = append(missing, "amount") }
	if len(missing) > 0 {
		return idx, fmt.Errorf("CSV missing required columns: %s", strings.Join(missing, ", "))
	}
	return idx, nil
}

// parseCSVDateTime parses "YYYY-MM-DD" + "HH:MM" or "HH:MM:SS" into a time.Time
// in WAT (UTC+1).
func parseCSVDateTime(dateStr, timeStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)
	timeStr = strings.TrimSpace(timeStr)

	// Normalise time to HH:MM:SS
	switch len(timeStr) {
	case 5: // HH:MM
		timeStr += ":00"
	case 8: // HH:MM:SS — already correct
	default:
		return time.Time{}, fmt.Errorf("time %q must be HH:MM or HH:MM:SS", timeStr)
	}

	combined := dateStr + "T" + timeStr
	t, err := time.ParseInLocation("2006-01-02T15:04:05", combined, WAT)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse %q: %w", combined, err)
	}
	return t, nil
}

// safeCol returns record[i] if i is within bounds, otherwise "".
func safeCol(record []string, i int) string {
	if i < 0 || i >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[i])
}
