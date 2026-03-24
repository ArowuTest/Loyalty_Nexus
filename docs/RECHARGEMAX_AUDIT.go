package services

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	crand "crypto/rand"
	"encoding/binary"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"rechargemax/internal/domain/entities"
	"rechargemax/internal/domain/repositories"
)

// DrawService handles draw management and CSV export/import
type DrawService struct {
	db               *gorm.DB
	drawRepo         repositories.DrawRepository
	rechargeRepo     repositories.RechargeRepository
	subscriptionRepo repositories.SubscriptionRepository
	wheelSpinRepo    repositories.SpinResultRepository
}

// DrawEntryExport represents a draw entry for CSV export
type DrawEntryExport struct {
	MSISDN string
	Points int64
}

// WinnerImport represents a winner from CSV import
type WinnerImport struct {
	MSISDN   string
	Position int
	Prize    string
	Amount   int64
}

// NewDrawService creates a new draw service
func NewDrawService(
	db *gorm.DB,
	drawRepo repositories.DrawRepository,
	rechargeRepo repositories.RechargeRepository,
	subscriptionRepo repositories.SubscriptionRepository,
	wheelSpinRepo repositories.SpinResultRepository,
) *DrawService {
	return &DrawService{
		db:               db,
		drawRepo:         drawRepo,
		rechargeRepo:     rechargeRepo,
		subscriptionRepo: subscriptionRepo,
		wheelSpinRepo:    wheelSpinRepo,
	}
}

// generateDrawCode generates a unique draw code in the format DRAW-YYYYMMDD-XXXX
func generateDrawCode() string {
	var b [8]byte
	crand.Read(b[:]) //nolint:errcheck — Read never fails on Linux
	n := int(binary.BigEndian.Uint64(b[:]) % 9000)
	return fmt.Sprintf("DRAW-%s-%04d", time.Now().Format("20060102"), n+1000)
}

// CreateDraw creates a new draw record
func (s *DrawService) CreateDraw(ctx context.Context, name, description string, drawDate time.Time, prizePool int64) (*entities.Draw, error) {
	descPtr := &description
	drawTimePtr := &drawDate
	draw := &entities.Draw{
		ID:          uuid.New(),
		DrawCode:    generateDrawCode(),
		Name:        name,
		Type:        "DAILY",
		Description: descPtr,
		StartTime:   drawDate.Add(-24 * time.Hour), // Start 24h before draw
		EndTime:     drawDate,
		DrawTime:    drawTimePtr,
		Status:      "UPCOMING",
		PrizePool:   float64(prizePool),
	}

	err := s.drawRepo.Create(ctx, draw)
	if err != nil {
		return nil, fmt.Errorf("failed to create draw: %w", err)
	}

	return draw, nil
}

// CreateDrawWithTemplate creates a new draw with a prize template
func (s *DrawService) CreateDrawWithTemplate(
	ctx context.Context,
	name, description string,
	drawDate time.Time,
	drawTypeID, prizeTemplateID uuid.UUID,
) (*entities.Draw, error) {
	// Get prize template to calculate total prize pool
	var totalPrizePool float64
	var categories []entities.PrizeCategory
	
	err := s.db.Where("prize_template_id = ?", prizeTemplateID).Find(&categories).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get prize categories: %w", err)
	}
	
	for _, cat := range categories {
		totalPrizePool += cat.PrizeAmount * float64(cat.WinnerCount)
	}
	
	descPtr := &description
	drawTimePtr := &drawDate
	draw := &entities.Draw{
		ID:              uuid.New(),
		DrawCode:        generateDrawCode(),
		Name:            name,
		Type:            "DAILY", // Will be updated based on draw type
		Description:     descPtr,
		StartTime:       drawDate.Add(-24 * time.Hour),
		EndTime:         drawDate,
		DrawTime:        drawTimePtr,
		Status:          "UPCOMING",
		PrizePool:       totalPrizePool,
		DrawTypeID:      &drawTypeID,
		PrizeTemplateID: &prizeTemplateID,
	}
	
	err = s.drawRepo.Create(ctx, draw)
	if err != nil {
		return nil, fmt.Errorf("failed to create draw: %w", err)
	}
	
	return draw, nil
}

// ExportDrawEntries aggregates draw entries from the draw_entries table for
// the given date range and writes a CSV to outputPath.
// Each row = one MSISDN + its total entries count across the period.
func (s *DrawService) ExportDrawEntries(ctx context.Context, startDate, endDate time.Time, outputPath string) (string, error) {
	// Aggregate entries count from draw_entries joined to draws (by date)
	type row struct {
		MSISDN       string
		TotalEntries int64
	}
	var rows []row

	err := s.db.WithContext(ctx).
		Table("draw_entries de").
		Select("de.msisdn, COALESCE(SUM(de.entries_count), 0) AS total_entries").
		Joins("JOIN draws d ON d.id = de.draw_id").
		Where("d.created_at BETWEEN ? AND ?", startDate, endDate).
		Group("de.msisdn").
		Order("total_entries DESC").
		Scan(&rows).Error
	if err != nil {
		return "", fmt.Errorf("failed to aggregate draw entries: %w", err)
	}

	// Create CSV file
	file, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{"MSISDN", "TotalEntries"}); err != nil {
		return "", fmt.Errorf("failed to write CSV header: %w", err)
	}
	for _, r := range rows {
		if err := writer.Write([]string{r.MSISDN, strconv.FormatInt(r.TotalEntries, 10)}); err != nil {
			return "", fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return outputPath, nil
}

func (s *DrawService) ImportWinners(ctx context.Context, drawID uuid.UUID, csvPath string) ([]*WinnerImport, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Validate header
	expectedHeader := []string{"MSISDN", "Position", "Prize", "Amount"}
	if len(header) != len(expectedHeader) {
		return nil, fmt.Errorf("invalid CSV header format. Expected: %v", expectedHeader)
	}

	var winners []*WinnerImport

	// Read winners
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV row: %w", err)
		}

		if len(record) != 4 {
			continue // Skip invalid rows
		}

		position, err := strconv.Atoi(record[1])
		if err != nil {
			continue // Skip invalid position
		}

		amount, err := strconv.ParseInt(record[3], 10, 64)
		if err != nil {
			continue // Skip invalid amount
		}

		winners = append(winners, &WinnerImport{
			MSISDN:   record[0],
			Position: position,
			Prize:    record[2],
			Amount:   amount,
		})
	}

	return winners, nil
}

// GetDrawByID retrieves a draw by ID
func (s *DrawService) GetDrawByID(ctx context.Context, drawID uuid.UUID) (*entities.Draw, error) {
	return s.drawRepo.FindByID(ctx, drawID)
}

// GetDraws retrieves all draws with pagination
func (s *DrawService) GetDraws(ctx context.Context, page, limit int) ([]*entities.Draw, int64, error) {
	// Calculate offset from page number
	offset := (page - 1) * limit
	
	draws, err := s.drawRepo.FindAll(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get draws: %w", err)
	}

	total, err := s.drawRepo.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count draws: %w", err)
	}

	return draws, total, nil
}

// UpdateDrawStatus updates the status of a draw
func (s *DrawService) UpdateDrawStatus(ctx context.Context, drawID uuid.UUID, status string) error {
	draw, err := s.drawRepo.FindByID(ctx, drawID)
	if err != nil {
		return fmt.Errorf("draw not found: %w", err)
	}

	draw.Status = status
	if status == "completed" {
		draw.CompletedAt = timePtr(time.Now())
	}

	return s.drawRepo.Update(ctx, draw)
}

// GetActiveDraw gets the currently active draw
func (s *DrawService) GetActiveDraw(ctx context.Context) (*entities.Draw, error) {
	draws, err := s.drawRepo.FindByStatus(ctx, "ACTIVE", 1, 0)
	if err != nil {
		return nil, err
	}
	if len(draws) == 0 {
		return nil, fmt.Errorf("no active draw found")
	}
	return draws[0], nil
}

// GetUpcomingDraws gets upcoming draws
func (s *DrawService) GetUpcomingDraws(ctx context.Context, limit int) ([]*entities.Draw, error) {
	return s.drawRepo.FindUpcoming(ctx, limit)
}

// GetCompletedDraws gets completed draws
func (s *DrawService) GetCompletedDraws(ctx context.Context, page, limit int) ([]*entities.Draw, int64, error) {
	draws, err := s.drawRepo.FindByStatus(ctx, "COMPLETED", 100, 0)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get completed draws: %w", err)
	}

	// For pagination, we'd need a proper method in the repository
	// This is a simplified version
	total := int64(len(draws))

	return draws, total, nil
}

// GetActiveDraws returns all active draws
func (s *DrawService) GetActiveDraws(ctx context.Context) ([]*entities.Draw, error) {
	draws, err := s.drawRepo.FindByStatus(ctx, "ACTIVE", 100, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get active draws: %w", err)
	}
	return draws, nil
}

// GetUserEntries returns user's entries for a specific draw
func (s *DrawService) GetUserEntries(ctx context.Context, drawID uuid.UUID, msisdn string) ([]DrawEntryResponse, error) {
	// Get draw to verify it exists and get date range
	draw, err := s.GetDrawByID(ctx, drawID)
	if err != nil {
		return nil, err
	}

	// Get user's recharges during draw period
	// This is a simplified implementation
	var entries []DrawEntryResponse

	// Implement actual entry retrieval logic
	// In production, this would:
	// 1. Query recharges for this user during draw period
	// 2. Calculate points from each recharge (₦200 = 1 point)
	// 3. Query subscriptions for this user during draw period
	// 4. Calculate subscription points (₦20/day = 1 point)
	// 5. Query wheel spins for bonus points
	// 6. Create entry records for each point
	//
	// Example implementation:
	// user, err := s.userRepo.FindByMSISDN(ctx, msisdn)
	// if err != nil {
	//     return nil, fmt.Errorf("user not found: %w", err)
	// }
	// 
	// // Get recharges during draw period
	// recharges, _ := s.rechargeRepo.FindByUserIDAndDateRange(ctx, user.ID, draw.StartDate, draw.EndDate)
	// for _, r := range recharges {
	//     if r.Status == "completed" {
	//         points := r.Amount / 20000 // ₦200 = 1 point
	//         for i := int64(0); i < points; i++ {
	//             entries = append(entries, DrawEntryResponse{
	//                 DrawID:      draw.ID,
	//                 MSISDN:      msisdn,
	//                 EntryNumber: fmt.Sprintf("%s-%d", r.ID.String(), i),
	//                 Source:      "recharge",
	//                 Amount:      r.Amount,
	//                 CreatedAt:   r.CreatedAt,
	//             })
	//         }
	//     }
	// }
	
	// Query draw_entries table for this user/draw combination
	type entryRow struct {
		ID          string    `gorm:"column:id"`
		DrawID      string    `gorm:"column:draw_id"`
		MSISDN      string    `gorm:"column:msisdn"`
		EntrySource string    `gorm:"column:entry_source"`
		Amount      int64     `gorm:"column:amount"`
		CreatedAt   time.Time `gorm:"column:created_at"`
	}
	var rows []entryRow
	s.db.WithContext(ctx).Raw(`
		SELECT id, draw_id, msisdn, entry_source, amount, created_at
		FROM draw_entries
		WHERE draw_id = ? AND msisdn = ?
		ORDER BY created_at ASC
	`, draw.ID, msisdn).Scan(&rows)

	userID := uuid.Nil
	if user, err2 := s.rechargeRepo.FindByReference(ctx, ""); err2 == nil && user != nil {
		// user lookup not needed here; MSISDN is enough
		_ = user
	}

	for _, r := range rows {
		id, _ := uuid.Parse(r.ID)
		drawUUID, _ := uuid.Parse(r.DrawID)
		entries = append(entries, DrawEntryResponse{
			ID:        id,
			DrawID:    drawUUID,
			UserID:    userID,
			MSISDN:    r.MSISDN,
			Amount:    r.Amount,
			EntryDate: r.CreatedAt,
		})
	}

	return entries, nil
}

// DrawEntryResponse represents a draw entry
type DrawEntryResponse struct {
	ID        uuid.UUID `json:"id"`
	DrawID    uuid.UUID `json:"draw_id"`
	UserID    uuid.UUID `json:"user_id"`
	MSISDN    string    `json:"msisdn"`
	Amount    int64     `json:"amount"` // Amount in kobo
	EntryDate time.Time `json:"entry_date"`
}

// GetDrawWinners returns winners for a specific draw
func (s *DrawService) GetDrawWinners(ctx context.Context, drawID uuid.UUID) ([]DrawWinnerResponse, error) {
	// Get draw to verify it exists
	_, err := s.GetDrawByID(ctx, drawID)
	if err != nil {
		return nil, err
	}

	// Get winners from winner repository
	// In production, this would:
	// 1. Query winners table for this draw_id
	// 2. Join with users table to get user details
	// 3. Return winner information with prize details
	//
	// Example implementation:
	// winners, err := s.winnerRepo.FindByDrawID(ctx, drawID)
	// if err != nil {
	//     return nil, fmt.Errorf("failed to get winners: %w", err)
	// }
	// 
	// var response []DrawWinnerResponse
	// for _, winner := range winners {
	//     user, _ := s.userRepo.FindByMSISDN(ctx, winner.MSISDN)
	//     response = append(response, DrawWinnerResponse{
	//         ID:          winner.ID,
	//         DrawID:      winner.DrawID,
	//         MSISDN:      winner.MSISDN,
	//         UserName:    user.FullName,
	//         PrizeName:   winner.PrizeName,
	//         PrizeValue:  winner.PrizeValue,
	//         ClaimStatus: winner.ClaimStatus,
	//         WonAt:       winner.CreatedAt,
	//     })
	// }
	// return response, nil
	
	// Query winners table for this draw
	type winnerRow struct {
		ID               string     `gorm:"column:id"`
		DrawID           string     `gorm:"column:draw_id"`
		MSISDN           string     `gorm:"column:msisdn"`
		PrizeType        string     `gorm:"column:prize_type"`
		PrizeDescription string     `gorm:"column:prize_description"`
		PrizeAmount      *int64     `gorm:"column:prize_amount"`
		ClaimStatus      string     `gorm:"column:claim_status"`
		CreatedAt        time.Time  `gorm:"column:created_at"`
	}
	var rows []winnerRow
	s.db.WithContext(ctx).Raw(`
		SELECT id, draw_id, msisdn, prize_type, prize_description, prize_amount, claim_status, created_at
		FROM winners
		WHERE draw_id = ?
		ORDER BY position ASC
	`, drawID).Scan(&rows)

	var response []DrawWinnerResponse
	for _, r := range rows {
		id, _ := uuid.Parse(r.ID)
		dID, _ := uuid.Parse(r.DrawID)
		prizeVal := int64(0)
		if r.PrizeAmount != nil {
			prizeVal = *r.PrizeAmount
		}
		response = append(response, DrawWinnerResponse{
			ID:         id,
			DrawID:     dID,
			MSISDN:     r.MSISDN,
			PrizeType:  r.PrizeType,
			PrizeValue: float64(prizeVal),
			Status:     r.ClaimStatus,
			WonAt:      r.CreatedAt,
		})
	}
	return response, nil
}

// DrawWinnerResponse represents a draw winner
type DrawWinnerResponse struct {
	ID         uuid.UUID `json:"id"`
	DrawID     uuid.UUID `json:"draw_id"`
	UserID     uuid.UUID `json:"user_id"`
	MSISDN     string    `json:"msisdn"`
	FullName   string    `json:"full_name"`
	PrizeType  string    `json:"prize_type"`
	PrizeValue float64   `json:"prize_value"`
	Status     string    `json:"status"`
	WonAt      time.Time `json:"won_at"`
}



// UpdateDraw updates draw details (admin operation)
func (s *DrawService) UpdateDraw(ctx context.Context, drawID string, updates map[string]interface{}) (*entities.Draw, error) {
	// Parse UUID
	did, err := uuid.Parse(drawID)
	if err != nil {
		return nil, fmt.Errorf("invalid draw ID format: %w", err)
	}
	
	// Get existing draw
	draw, err := s.drawRepo.FindByID(ctx, did)
	if err != nil {
		return nil, fmt.Errorf("draw not found: %w", err)
	}
	
	// Apply updates
	if name, ok := updates["name"].(string); ok {
		draw.Name = name
	}
	
	if description, ok := updates["description"].(string); ok {
		draw.Description = &description
	}
	
	if drawDate, ok := updates["draw_date"].(string); ok {
		parsedDate, err := time.Parse(time.RFC3339, drawDate)
		if err == nil {
			draw.DrawTime = &parsedDate
		}
	}
	
	if status, ok := updates["status"].(string); ok {
		validStatuses := []string{"PENDING", "ACTIVE", "COMPLETED", "CANCELLED", "UPCOMING"}
		isValid := false
		for _, s := range validStatuses {
			if status == s {
				isValid = true
				break
			}
		}
		if isValid {
			draw.Status = status
		}
	}
	
	if prizePool, ok := updates["prize_pool"].(float64); ok {
		draw.PrizePool = prizePool
	}
	
	// Update winners count and runner-ups count
	if winnersCount, ok := updates["winners_count"].(float64); ok {
		draw.WinnersCount = int(winnersCount)
	}
	
	if runnerUpsCount, ok := updates["runner_ups_count"].(float64); ok {
		draw.RunnerUpsCount = int(runnerUpsCount)
	}
	
	// Save updated draw - use UpdateStatus for status-only updates to avoid draw_code unique constraint issues
	if len(updates) == 1 {
		if status, ok := updates["status"].(string); ok {
			if err := s.drawRepo.UpdateStatus(ctx, did, status); err != nil {
				return nil, fmt.Errorf("failed to update draw status: %w", err)
			}
			draw.Status = status
			return draw, nil
		}
	}
	// For other updates, use targeted column updates to avoid overwriting draw_code
	updateMap := map[string]interface{}{}
	if _, ok := updates["name"]; ok { updateMap["name"] = draw.Name }
	if _, ok := updates["description"]; ok { updateMap["description"] = draw.Description }
	if _, ok := updates["draw_date"]; ok { updateMap["draw_time"] = draw.DrawTime }
	if _, ok := updates["status"]; ok { updateMap["status"] = draw.Status }
	if _, ok := updates["prize_pool"]; ok { updateMap["prize_pool"] = draw.PrizePool }
	if _, ok := updates["winners_count"]; ok { updateMap["winners_count"] = draw.WinnersCount }
	if _, ok := updates["runner_ups_count"]; ok { updateMap["runner_ups_count"] = draw.RunnerUpsCount }
	if len(updateMap) > 0 {
		if err := s.db.Model(draw).Updates(updateMap).Error; err != nil {
			return nil, fmt.Errorf("failed to update draw: %w", err)
		}
	}
	
	return draw, nil
}

// ExecuteDraw executes a draw (triggers winner selection)
func (s *DrawService) ExecuteDraw(ctx context.Context, drawID string) error {
	// Parse UUID
	did, err := uuid.Parse(drawID)
	if err != nil {
		return fmt.Errorf("invalid draw ID format: %w", err)
	}
	
	// Get draw
	draw, err := s.drawRepo.FindByID(ctx, did)
	if err != nil {
		return fmt.Errorf("draw not found: %w", err)
	}
	
	// Validate draw status
	if draw.Status == "COMPLETED" {
		return fmt.Errorf("draw has already been executed")
	}
	
	if draw.Status == "CANCELLED" {
		return fmt.Errorf("draw has been cancelled")
	}
	
	// Check if draw has entries
	if draw.TotalEntries == 0 {
		return fmt.Errorf("no entries found for this draw")
	}
	
	// Update draw status to completed
	draw.Status = "COMPLETED"
	now := time.Now()
	draw.DrawTime = &now
	draw.CompletedAt = &now
	
	if err := s.drawRepo.Update(ctx, draw); err != nil {
		return fmt.Errorf("failed to update draw status: %w", err)
	}
	
	// Prize Tier System: Category-aware winner selection
	// 1. Load prize categories from template
	// 2. Get all unique MSISDNs from draw entries
	// 3. Select winners for each category (no duplicates across categories)
	// 4. Store winners with category information
	
	// Load prize categories
	if draw.PrizeTemplateID == nil {
		return fmt.Errorf("draw does not have a prize template assigned")
	}
	
	var prizeCategories []entities.PrizeCategory
	if err := s.db.Where("prize_template_id = ?", *draw.PrizeTemplateID).
		Order("display_order ASC").
		Find(&prizeCategories).Error; err != nil {
		return fmt.Errorf("failed to load prize categories: %w", err)
	}
	
	if len(prizeCategories) == 0 {
		return fmt.Errorf("no prize categories found for this template")
	}
	
	// Get all unique MSISDNs from draw entries
	var uniqueMSISDNs []string
	if err := s.db.Table("draw_entries").
		Where("draw_id = ?", did).
		Distinct("msisdn").
		Pluck("msisdn", &uniqueMSISDNs).Error; err != nil {
		return fmt.Errorf("failed to get unique MSISDNs: %w", err)
	}
	
	if len(uniqueMSISDNs) == 0 {
		return fmt.Errorf("no unique MSISDNs found in draw entries")
	}
	
	// Track selected MSISDNs to prevent duplicates across categories
	selectedMSISDNs := make(map[string]bool)
	var allWinners []entities.DrawWinners
	position := 1
	
	// Select winners for each prize category
	for _, category := range prizeCategories {
		// Filter out already selected MSISDNs
		availableMSISDNs := make([]string, 0)
		for _, msisdn := range uniqueMSISDNs {
			if !selectedMSISDNs[msisdn] {
				availableMSISDNs = append(availableMSISDNs, msisdn)
			}
		}
		
		// Check if we have enough MSISDNs
		totalNeeded := category.WinnerCount + category.RunnerUpCount
		if len(availableMSISDNs) < totalNeeded {
			return fmt.Errorf("insufficient unique MSISDNs for category %s: need %d, have %d",
				category.CategoryName, totalNeeded, len(availableMSISDNs))
		}
		
		// Crypto-random Fisher-Yates shuffle (SEC-009: crypto/rand, not math/rand)
		shuffledMSISDNs := make([]string, len(availableMSISDNs))
		copy(shuffledMSISDNs, availableMSISDNs)
		cryptoShuffle(shuffledMSISDNs)
		
		// Select winners for this category
		for i := 0; i < category.WinnerCount; i++ {
			msisdn := shuffledMSISDNs[i]
			selectedMSISDNs[msisdn] = true
			
				categoryID := category.ID
				categoryName := category.CategoryName
				winner := entities.DrawWinners{
					ID:              uuid.New(),
					DrawID:          did,
					MSISDN:          msisdn,
					Position:        position,
					PrizeAmount:     int64(category.PrizeAmount),
					IsRunnerUp:      false,
					PrizeCategoryID: &categoryID,
					CategoryName:    &categoryName,
					CreatedAt:       &now,
				}
				allWinners = append(allWinners, winner)
				position++
			}
			
			// Select runner-ups for this category
			for i := category.WinnerCount; i < totalNeeded; i++ {
				msisdn := shuffledMSISDNs[i]
				selectedMSISDNs[msisdn] = true
				
				categoryID := category.ID
				categoryName := category.CategoryName
				runnerUp := entities.DrawWinners{
					ID:              uuid.New(),
					DrawID:          did,
					MSISDN:          msisdn,
					Position:        position,
					PrizeAmount:     int64(category.PrizeAmount),
				IsRunnerUp:      true,
				PrizeCategoryID: &categoryID,
				CategoryName:    &categoryName,
				CreatedAt:       &now,
			}
			allWinners = append(allWinners, runnerUp)
			position++
		}
	}
	
	// Save all winners to database
	if len(allWinners) > 0 {
		if err := s.db.Create(&allWinners).Error; err != nil {
			return fmt.Errorf("failed to save winners: %w", err)
		}
	}
	
	// Update draw statistics
	draw.TotalWinners = len(allWinners)
	if err := s.drawRepo.Update(ctx, draw); err != nil {
		return fmt.Errorf("failed to update draw statistics: %w", err)
	}
	
	return nil
}

// ProcessCSVEntries processes a CSV file containing MSISDN and Points
// Format: MSISDN,Points
// Example: 08012345678,5
func (s *DrawService) ProcessCSVEntries(ctx context.Context, drawID uuid.UUID, csvReader io.Reader) (int, error) {
	reader := csv.NewReader(csvReader)
	reader.FieldsPerRecord = 2
	reader.TrimLeadingSpace = true
	
	entriesCreated := 0
	lineNumber := 0
	
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return entriesCreated, fmt.Errorf("error reading CSV at line %d: %w", lineNumber, err)
		}
		
		lineNumber++
		
		// Skip header row if present
		if lineNumber == 1 && (record[0] == "MSISDN" || record[0] == "msisdn") {
			continue
		}
		
		msisdn := record[0]
		pointsStr := record[1]
		
		// Validate MSISDN format (Nigerian: 080/081/070/090/091 + 8 digits)
		if !isValidNigerianMSISDN(msisdn) {
			return entriesCreated, fmt.Errorf("invalid MSISDN format at line %d: %s", lineNumber, msisdn)
		}
		
		// Parse points
		points, err := strconv.Atoi(pointsStr)
		if err != nil || points <= 0 {
			return entriesCreated, fmt.Errorf("invalid points value at line %d: %s", lineNumber, pointsStr)
		}
		
		// Create N entries for this MSISDN based on points
		for i := 0; i < points; i++ {
			now := time.Now()
			entry := &entities.DrawEntries{
				ID:        uuid.New(),
				DrawID:    drawID,
				MSISDN:    msisdn,
				CreatedAt: &now,
			}
			
			if err := s.drawRepo.CreateEntry(ctx, entry); err != nil {
				return entriesCreated, fmt.Errorf("failed to create entry for %s: %w", msisdn, err)
			}
			
			entriesCreated++
		}
	}
	
	return entriesCreated, nil
}

// isValidNigerianMSISDN validates Nigerian phone number format
func isValidNigerianMSISDN(msisdn string) bool {
	// Remove any spaces or dashes
	msisdn = strings.ReplaceAll(msisdn, " ", "")
	msisdn = strings.ReplaceAll(msisdn, "-", "")
	
	// Must be 11 digits starting with 0, or 13 digits starting with 234
	if len(msisdn) == 11 && msisdn[0] == '0' {
		// Check if starts with valid prefix
		validPrefixes := []string{"0803", "0806", "0810", "0813", "0814", "0816", "0903", "0906", "0913", "0916", "0805", "0807", "0811", "0815", "0905", "0915", "0802", "0808", "0812", "0902", "0904", "0907", "0912", "0701", "0708", "0809", "0817", "0818", "0909", "0908"}
		for _, vp := range validPrefixes {
			if strings.HasPrefix(msisdn, vp) {
				return true
			}
		}
	} else if len(msisdn) == 13 && strings.HasPrefix(msisdn, "234") {
		// Convert to 0-format and validate
		return isValidNigerianMSISDN("0" + msisdn[3:])
	}
	
	return false
}



// cryptoShuffle performs a Fisher-Yates shuffle using crypto/rand to ensure
// unpredictable draw winner selection (SEC-009).
func cryptoShuffle(s []string) {
	for i := len(s) - 1; i > 0; i-- {
		var b [8]byte
		crand.Read(b[:]) //nolint:errcheck
		j := int(binary.BigEndian.Uint64(b[:]) % uint64(i+1))
		s[i], s[j] = s[j], s[i]
	}
}
package services

import (
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"rechargemax/internal/domain/entities"
	"rechargemax/internal/domain/repositories"
)

// PointsService handles user points management and tracking
type PointsService struct {
	userRepo         repositories.UserRepository
	rechargeRepo     repositories.RechargeRepository
	ussdRepo         repositories.USSDRechargeRepository
	subscriptionRepo repositories.SubscriptionRepository
	spinRepo         repositories.SpinRepository
	adjustmentRepo   repositories.PointsAdjustmentRepository
	notificationService *NotificationService
}

// NewPointsService creates a new points service
func NewPointsService(
	userRepo repositories.UserRepository,
	rechargeRepo repositories.RechargeRepository,
	ussdRepo repositories.USSDRechargeRepository,
	subscriptionRepo repositories.SubscriptionRepository,
	spinRepo repositories.SpinRepository,
	adjustmentRepo repositories.PointsAdjustmentRepository,
	notificationService *NotificationService,
) *PointsService {
	return &PointsService{
		userRepo:         userRepo,
		rechargeRepo:     rechargeRepo,
		ussdRepo:         ussdRepo,
		subscriptionRepo: subscriptionRepo,
		spinRepo:         spinRepo,
		adjustmentRepo:   adjustmentRepo,
		notificationService: notificationService,
	}
}

// UserPointsSummary represents a user's points summary
type UserPointsSummary struct {
	UserID           uuid.UUID `json:"user_id"`
	MSISDN           string    `json:"msisdn"`
	Email            string    `json:"email"`
	FullName         string    `json:"full_name"`
	TotalPoints      int       `json:"total_points"`
	AvailablePoints  int       `json:"available_points"`
	LockedPoints     int       `json:"locked_points"`
	LifetimePoints   int       `json:"lifetime_points"`
	LastEarnedAt     *time.Time `json:"last_earned_at"`
	PointsBySource   map[string]int `json:"points_by_source"`
}

// PointsHistoryEntry represents a single points transaction
type PointsHistoryEntry struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	MSISDN      string     `json:"msisdn"`
	Points      int        `json:"points"`
	Source      string     `json:"source"`
	Description string     `json:"description"`
	ReferenceID *uuid.UUID `json:"reference_id"`
	Status      string     `json:"status"`
	CreatedBy   *uuid.UUID `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
}

// PointsStatistics represents overall points statistics
type PointsStatistics struct {
	TotalUsers           int            `json:"total_users"`
	TotalPointsIssued    int            `json:"total_points_issued"`
	TotalPointsAvailable int            `json:"total_points_available"`
	TotalPointsLocked    int            `json:"total_points_locked"`
	PointsBySource       map[string]int `json:"points_by_source"`
	TopUsers             []UserPointsSummary `json:"top_users"`
}

// GetUsersWithPoints retrieves all users with their points summary
func (s *PointsService) GetUsersWithPoints(ctx context.Context, searchQuery string, dateFrom, dateTo *time.Time) ([]*UserPointsSummary, error) {
	users, err := s.userRepo.FindAll(ctx, 10000, 0) // Get all users
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve users: %w", err)
	}

	var summaries []*UserPointsSummary
	for _, user := range users {
		// Apply search filter
		if searchQuery != "" {
			lowerQuery := strings.ToLower(searchQuery)
			if !strings.Contains(strings.ToLower(user.MSISDN), lowerQuery) &&
				!strings.Contains(strings.ToLower(user.Email), lowerQuery) &&
				!strings.Contains(strings.ToLower(user.FullName), lowerQuery) {
				continue
			}
		}

		summary, err := s.getUserPointsSummary(ctx, user.ID, dateFrom, dateTo)
		if err != nil {
			continue // Skip users with errors
		}

		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// getUserPointsSummary calculates points summary for a single user
func (s *PointsService) getUserPointsSummary(ctx context.Context, userID uuid.UUID, dateFrom, dateTo *time.Time) (*UserPointsSummary, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	summary := &UserPointsSummary{
		UserID:         user.ID,
		MSISDN:         user.MSISDN,
		Email:          user.Email,
		FullName:       user.FullName,
		TotalPoints:    user.TotalPoints,
		LifetimePoints: user.TotalPoints,
		PointsBySource: make(map[string]int),
	}

	// Calculate points by source
	pointsBySource, err := s.calculatePointsBySource(ctx, userID, dateFrom, dateTo)
	if err == nil {
		summary.PointsBySource = pointsBySource
	}

	// Get last earned date
	lastEarned, err := s.getLastPointsEarnedDate(ctx, userID)
	if err == nil && lastEarned != nil {
		summary.LastEarnedAt = lastEarned
	}

	// For now, assume all points are available (locked points would be in active draws)
	summary.AvailablePoints = user.TotalPoints
	summary.LockedPoints = 0

	return summary, nil
}

// calculatePointsBySource calculates points distribution by source
func (s *PointsService) calculatePointsBySource(ctx context.Context, userID uuid.UUID, dateFrom, dateTo *time.Time) (map[string]int, error) {
	pointsBySource := make(map[string]int)

	// Platform recharges
	recharges, err := s.rechargeRepo.FindByUserID(ctx, userID, 10000, 0)
	if err == nil {
		for _, r := range recharges {
			if dateFrom != nil && r.CreatedAt.Before(*dateFrom) {
				continue
			}
			if dateTo != nil && r.CreatedAt.After(*dateTo) {
				continue
			}
			if r.Status == "SUCCESS" {
				pointsBySource["platform_recharge"] += int(r.Amount / 20000) // ₦200 = 1 point
			}
		}
	}

	// USSD recharges
	user, err := s.userRepo.FindByID(ctx, userID)
	if err == nil {
		var startDate, endDate time.Time
		if dateFrom != nil {
			startDate = *dateFrom
		}
		if dateTo != nil {
			endDate = *dateTo
		}
		ussdRecharges, err := s.ussdRepo.FindByMSISDN(ctx, user.MSISDN, startDate, endDate)
		if err == nil {
			for _, ur := range ussdRecharges {
			if dateFrom != nil && ur.ReceivedAt.Before(*dateFrom) {
				continue
			}
			if dateTo != nil && ur.ReceivedAt.After(*dateTo) {
				continue
			}
			pointsBySource["ussd_recharge"] += ur.PointsEarned
			}
		}
	}

	// Wheel spins
	spins, err := s.spinRepo.FindByUserID(ctx, userID, 10000, 0)
	if err == nil {
		for _, spin := range spins {
			if dateFrom != nil && spin.CreatedAt.Before(*dateFrom) {
				continue
			}
			if dateTo != nil && spin.CreatedAt.After(*dateTo) {
				continue
			}
			if spin.PrizeType == "points" {
				pointsBySource["wheel_spin"] += int(spin.PrizeValue)
			}
		}
	}

	// Admin adjustments
	adjustments, err := s.adjustmentRepo.FindByUserID(ctx, userID)
	if err == nil {
		for _, adj := range adjustments {
			if dateFrom != nil && adj.CreatedAt.Before(*dateFrom) {
				continue
			}
			if dateTo != nil && adj.CreatedAt.After(*dateTo) {
				continue
			}
			if adj.Points > 0 {
				pointsBySource["admin_added"] += adj.Points
			} else {
				pointsBySource["admin_deducted"] += -adj.Points
			}
		}
	}

	return pointsBySource, nil
}

// getLastPointsEarnedDate gets the last date points were earned
func (s *PointsService) getLastPointsEarnedDate(ctx context.Context, userID uuid.UUID) (*time.Time, error) {
	var lastDate *time.Time

	// Check latest recharge
	recharges, err := s.rechargeRepo.FindByUserID(ctx, userID, 10000, 0)
	if err == nil {
		for _, r := range recharges {
			if r.Status == "SUCCESS" {
				if lastDate == nil || r.CreatedAt.After(*lastDate) {
					lastDate = &r.CreatedAt
				}
			}
		}
	}

	// Check latest USSD recharge
	user2, err := s.userRepo.FindByID(ctx, userID)
	if err == nil {
		ussdRecharges, err := s.ussdRepo.FindByMSISDN(ctx, user2.MSISDN, time.Time{}, time.Time{})
		if err == nil {
		for _, ur := range ussdRecharges {
			if lastDate == nil || ur.ReceivedAt.After(*lastDate) {
				lastDate = &ur.ReceivedAt
			}
		}
		}
	}

	// Check latest spin
	spins, err := s.spinRepo.FindByUserID(ctx, userID, 10000, 0)
	if err == nil {
		for _, spin := range spins {
			if spin.PrizeType == "points" {
				if lastDate == nil || spin.CreatedAt.After(*lastDate) {
					lastDate = &spin.CreatedAt
				}
			}
		}
	}

	return lastDate, nil
}

// GetPointsHistory retrieves points transaction history
func (s *PointsService) GetPointsHistory(ctx context.Context, userID *uuid.UUID, source string, dateFrom, dateTo *time.Time) ([]*PointsHistoryEntry, error) {
	var history []*PointsHistoryEntry

	// If userID is specified, get history for that user
	if userID != nil {
		userHistory, err := s.getUserPointsHistory(ctx, *userID, source, dateFrom, dateTo)
		if err != nil {
			return nil, err
		}
		history = append(history, userHistory...)
	} else {
		// Get history for all users
		users, err := s.userRepo.FindAll(ctx, 10000, 0) // Get all users
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve users: %w", err)
		}

		for _, user := range users {
			userHistory, err := s.getUserPointsHistory(ctx, user.ID, source, dateFrom, dateTo)
			if err != nil {
				continue
			}
			history = append(history, userHistory...)
		}
	}

	return history, nil
}

// getUserPointsHistory gets points history for a specific user
func (s *PointsService) getUserPointsHistory(ctx context.Context, userID uuid.UUID, source string, dateFrom, dateTo *time.Time) ([]*PointsHistoryEntry, error) {
	var history []*PointsHistoryEntry

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Platform recharges
	if source == "" || source == "platform_recharge" {
		recharges, err := s.rechargeRepo.FindByUserID(ctx, userID, 10000, 0)
		if err == nil {
			for _, r := range recharges {
				if dateFrom != nil && r.CreatedAt.Before(*dateFrom) {
					continue
				}
				if dateTo != nil && r.CreatedAt.After(*dateTo) {
					continue
				}
				if r.Status == "SUCCESS" {
					points := int(r.Amount / 20000)
					history = append(history, &PointsHistoryEntry{
						ID:          r.ID,
						UserID:      userID,
						MSISDN:      user.MSISDN,
						Points:      points,
						Source:      "platform_recharge",
						Description: fmt.Sprintf("Recharge of ₦%.2f", float64(r.Amount)/100),
						ReferenceID: &r.ID,
						Status:      "completed",
						CreatedAt:   r.CreatedAt,
					})
				}
			}
		}
	}

	// USSD recharges
	if source == "" || source == "ussd_recharge" {
		user3, err := s.userRepo.FindByID(ctx, userID)
		if err == nil {
			ussdRecharges, err := s.ussdRepo.FindByMSISDN(ctx, user3.MSISDN, time.Time{}, time.Time{})
			if err == nil {
				for _, ur := range ussdRecharges {
				if dateFrom != nil && ur.ReceivedAt.Before(*dateFrom) {
					continue
				}
				if dateTo != nil && ur.ReceivedAt.After(*dateTo) {
					continue
				}
				history = append(history, &PointsHistoryEntry{
					ID:          ur.ID,
					UserID:      userID,
					MSISDN:      user.MSISDN,
					Points:      ur.PointsEarned,
					Source:      "ussd_recharge",
					Description: fmt.Sprintf("USSD recharge of ₦%.2f on %s", float64(ur.Amount)/100, ur.Network),
					ReferenceID: &ur.ID,
					Status:      "completed",
					CreatedAt:   ur.ReceivedAt,
				})
				}
			}
		}
	}

	// Wheel spins
	if source == "" || source == "wheel_spin" {
		spins, err := s.spinRepo.FindByUserID(ctx, userID, 10000, 0)
		if err == nil {
			for _, spin := range spins {
				if dateFrom != nil && spin.CreatedAt.Before(*dateFrom) {
					continue
				}
				if dateTo != nil && spin.CreatedAt.After(*dateTo) {
					continue
				}
				if spin.PrizeType == "points" {
					history = append(history, &PointsHistoryEntry{
						ID:          spin.ID,
						UserID:      userID,
						MSISDN:      user.MSISDN,
						Points:      int(spin.PrizeValue),
						Source:      "wheel_spin",
						Description: fmt.Sprintf("Won %d points from wheel spin", spin.PrizeValue),
						ReferenceID: &spin.ID,
						Status:      "completed",
						CreatedAt:   spin.CreatedAt,
					})
				}
			}
		}
	}

	// Admin adjustments
	if source == "" || source == "admin_adjustment" {
		adjustments, err := s.adjustmentRepo.FindByUserID(ctx, userID)
		if err == nil {
			for _, adj := range adjustments {
				if dateFrom != nil && adj.CreatedAt.Before(*dateFrom) {
					continue
				}
				if dateTo != nil && adj.CreatedAt.After(*dateTo) {
					continue
				}
				history = append(history, &PointsHistoryEntry{
					ID:          adj.ID,
					UserID:      userID,
					MSISDN:      user.MSISDN,
					Points:      adj.Points,
					Source:      "admin_adjustment",
					Description: fmt.Sprintf("%s - %s", adj.Reason, adj.Description),
					ReferenceID: &adj.ID,
					Status:      "completed",
					CreatedBy:   &adj.AdminID,
					CreatedAt:   adj.CreatedAt,
				})
			}
		}
	}

	return history, nil
}

// AdjustUserPoints manually adjusts user points (admin function)
func (s *PointsService) AdjustUserPoints(ctx context.Context, userID uuid.UUID, points int, reason, description string, adminID uuid.UUID) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Update user points
	newTotal := user.TotalPoints + points
	if newTotal < 0 {
		return fmt.Errorf("adjustment would result in negative points balance")
	}

	user.TotalPoints = newTotal
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user points: %w", err)
	}

	// Create adjustment record
	adjustment := &entities.PointsAdjustment{
		ID:          uuid.New(),
		UserID:      userID,
		Points:      points,
		Reason:      reason,
		Description: description,
		AdminID:     adminID,
		CreatedBy:   adminID, // mirrors AdminID — satisfies created_by NOT NULL constraint
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.adjustmentRepo.Create(ctx, adjustment); err != nil {
		return fmt.Errorf("failed to create adjustment record: %w", err)
	}

	// Notify user of points adjustment
	if s.notificationService != nil {
		adjustmentType := "added"
		absPoints := points
		if points < 0 {
			adjustmentType = "deducted"
			absPoints = -points
		}
		msg := fmt.Sprintf("%d loyalty points have been %s to your RechargeMax account. Reason: %s", absPoints, adjustmentType, reason)
		go s.notificationService.SendSMS(ctx, user.MSISDN, msg)
	}

	return nil
}

// GetPointsStatistics retrieves overall points statistics
func (s *PointsService) GetPointsStatistics(ctx context.Context, dateFrom, dateTo *time.Time) (*PointsStatistics, error) {
	stats := &PointsStatistics{
		PointsBySource: make(map[string]int),
	}

	users, err := s.userRepo.FindAll(ctx, 10000, 0) // Get all users
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve users: %w", err)
	}

	stats.TotalUsers = len(users)

	// Calculate statistics
	var userSummaries []UserPointsSummary
	for _, user := range users {
		stats.TotalPointsIssued += user.TotalPoints
		stats.TotalPointsAvailable += user.TotalPoints // Simplified

		// Get points by source for this user
		pointsBySource, err := s.calculatePointsBySource(ctx, user.ID, dateFrom, dateTo)
		if err == nil {
			for source, points := range pointsBySource {
				stats.PointsBySource[source] += points
			}
		}

		// Collect user summary for top users calculation
		summary, err := s.getUserPointsSummary(ctx, user.ID, dateFrom, dateTo)
		if err == nil {
			userSummaries = append(userSummaries, *summary)
		}
	}

	// Get top 10 users by points using efficient sorting
	stats.TopUsers = s.getTopUsersByPoints(userSummaries, 10)

	return stats, nil
}

// getTopUsersByPoints returns top N users sorted by total points
func (s *PointsService) getTopUsersByPoints(users []UserPointsSummary, topN int) []UserPointsSummary {
	// Quick sort implementation for better performance
	if len(users) <= 1 {
		return users
	}

	// Sort users by total points (descending)
	s.quickSortUsers(users, 0, len(users)-1)

	// Return top N
	if len(users) > topN {
		return users[:topN]
	}
	return users
}

// quickSortUsers sorts users by total points in descending order
func (s *PointsService) quickSortUsers(users []UserPointsSummary, low, high int) {
	if low < high {
		pivot := s.partitionUsers(users, low, high)
		s.quickSortUsers(users, low, pivot-1)
		s.quickSortUsers(users, pivot+1, high)
	}
}

// partitionUsers partitions the users array for quicksort
func (s *PointsService) partitionUsers(users []UserPointsSummary, low, high int) int {
	pivot := users[high].TotalPoints
	i := low - 1

	for j := low; j < high; j++ {
		if users[j].TotalPoints > pivot { // Descending order
			i++
			users[i], users[j] = users[j], users[i]
		}
	}

	users[i+1], users[high] = users[high], users[i+1]
	return i + 1
}

// ExportUsersWithPointsToCSV exports users with points to CSV format
func (s *PointsService) ExportUsersWithPointsToCSV(ctx context.Context, searchQuery string, dateFrom, dateTo *time.Time) (string, error) {
	users, err := s.GetUsersWithPoints(ctx, searchQuery, dateFrom, dateTo)
	if err != nil {
		return "", err
	}

	var csvBuilder strings.Builder
	writer := csv.NewWriter(&csvBuilder)

	// Write header
	header := []string{"MSISDN", "Email", "Full Name", "Total Points", "Available Points", "Locked Points", "Lifetime Points", "Last Earned", "Platform Recharge", "USSD Recharge", "Wheel Spin", "Admin Added", "Admin Deducted"}
	writer.Write(header)

	// Write data
	for _, user := range users {
		lastEarned := ""
		if user.LastEarnedAt != nil {
			lastEarned = user.LastEarnedAt.Format("2006-01-02 15:04:05")
		}

		row := []string{
			user.MSISDN,
			user.Email,
			user.FullName,
			strconv.Itoa(user.TotalPoints),
			strconv.Itoa(user.AvailablePoints),
			strconv.Itoa(user.LockedPoints),
			strconv.Itoa(user.LifetimePoints),
			lastEarned,
			strconv.Itoa(user.PointsBySource["platform_recharge"]),
			strconv.Itoa(user.PointsBySource["ussd_recharge"]),
			strconv.Itoa(user.PointsBySource["wheel_spin"]),
			strconv.Itoa(user.PointsBySource["admin_added"]),
			strconv.Itoa(user.PointsBySource["admin_deducted"]),
		}
		writer.Write(row)
	}

	writer.Flush()
	return csvBuilder.String(), nil
}

// ExportPointsHistoryToCSV exports points history to CSV format
func (s *PointsService) ExportPointsHistoryToCSV(ctx context.Context, userID *uuid.UUID, source string, dateFrom, dateTo *time.Time) (string, error) {
	history, err := s.GetPointsHistory(ctx, userID, source, dateFrom, dateTo)
	if err != nil {
		return "", err
	}

	var csvBuilder strings.Builder
	writer := csv.NewWriter(&csvBuilder)

	// Write header
	header := []string{"Date", "MSISDN", "Points", "Source", "Description", "Status", "Created By"}
	writer.Write(header)

	// Write data
	for _, entry := range history {
		createdBy := ""
		if entry.CreatedBy != nil {
			createdBy = entry.CreatedBy.String()
		}

		row := []string{
			entry.CreatedAt.Format("2006-01-02 15:04:05"),
			entry.MSISDN,
			strconv.Itoa(entry.Points),
			entry.Source,
			entry.Description,
			entry.Status,
			createdBy,
		}
		writer.Write(row)
	}

	writer.Flush()
	return csvBuilder.String(), nil
}
package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"gorm.io/datatypes"
	"rechargemax/internal/domain/entities"
	"rechargemax/internal/domain/repositories"
)

// HLRService handles network detection via HLR lookup
type HLRService struct {
	networkCacheRepo repositories.NetworkCacheRepository
	termiiAPIKey     string
	cacheTTLDays     int
	httpClient       *http.Client
}

// NewHLRService creates a new HLR service instance
func NewHLRService(
	networkCacheRepo repositories.NetworkCacheRepository,
	termiiAPIKey string,
) *HLRService {
	return &HLRService{
		networkCacheRepo: networkCacheRepo,
		termiiAPIKey:     termiiAPIKey,
		cacheTTLDays:     60, // 60-day cache TTL
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// NetworkDetectionResult contains the result of network detection
type NetworkDetectionResult struct {
	MSISDN       string
	Network      string
	Source       string // 'hlr_api', 'cache', 'user_selection', 'prefix_fallback'
	Confidence   string // 'high', 'medium', 'low'
	CachedUntil  *time.Time
	ErrorMessage string
}

// DetectNetwork detects the network for a given MSISDN.
//
// Business rule (per product spec):
//  1. Try HLR API lookup (most accurate — handles ported numbers)
//  2. If HLR unavailable/fails, check cache — but ONLY accept entries whose
//     LookupSource is "hlr_api" or "user_selection" (NOT "prefix_fallback")
//  3. If no trusted cache, accept user's explicit network selection
//  4. No prefix fallback — ported numbers make prefix unreliable.
//     Return an error and ask the caller to prompt the user for their network.
func (s *HLRService) DetectNetwork(ctx context.Context, msisdn string, userSelectedNetwork *string) (*NetworkDetectionResult, error) {
	// ── Step 1: HLR API lookup (primary source) ──────────────────────────────
	hlrResult, hlrErr := s.lookupViaHLR(ctx, msisdn)
	if hlrErr == nil && hlrResult != nil {
		return hlrResult, nil
	}

	// ── Step 2: Trusted cache (hlr_api or user_selection sourced only) ───────
	cachedResult, cacheErr := s.getTrustedCachedNetwork(ctx, msisdn)
	if cacheErr == nil && cachedResult != nil {
		return cachedResult, nil
	}

	// ── Step 3: Explicit user selection (fallback when HLR unavailable) ──────
	if userSelectedNetwork != nil && *userSelectedNetwork != "" {
		return s.saveUserSelection(ctx, msisdn, *userSelectedNetwork)
	}

	// ── Step 4: Cannot determine network — no prefix fallback ────────────────
	// Return structured error so the caller can prompt the user to select their network.
	return nil, fmt.Errorf("network detection failed (HLR: %v; cache: %v) — user network selection required", hlrErr, cacheErr)
}

// getCachedNetwork retrieves network from cache if valid
func (s *HLRService) getCachedNetwork(ctx context.Context, msisdn string) (*NetworkDetectionResult, error) {
	// Normalize phone to international format for cache lookup
	normalizedMSISDN := normalizeToInternational(msisdn)
	cache, err := s.networkCacheRepo.FindValidCache(ctx, normalizedMSISDN)
	if err != nil {
		return nil, err
	}

	return &NetworkDetectionResult{
		MSISDN:      msisdn,
		Network:     cache.Network,
		Source:      "cache",
		Confidence:  s.getConfidenceLevel(cache.LookupSource),
		CachedUntil: &cache.CacheExpires,
	}, nil
}

// getTrustedCachedNetwork retrieves network from cache ONLY if the entry was
// sourced from hlr_api or user_selection. prefix_fallback entries are rejected
// because ported numbers make prefix detection unreliable.
func (s *HLRService) getTrustedCachedNetwork(ctx context.Context, msisdn string) (*NetworkDetectionResult, error) {
	normalizedMSISDN := normalizeToInternational(msisdn)
	cache, err := s.networkCacheRepo.FindValidCache(ctx, normalizedMSISDN)
	if err != nil {
		return nil, err
	}

	// Only trust hlr_api and user_selection sourced entries
	if cache.LookupSource == "prefix_fallback" {
		return nil, fmt.Errorf("cache entry is prefix_fallback sourced — not trusted for ported number detection")
	}

	return &NetworkDetectionResult{
		MSISDN:      msisdn,
		Network:     cache.Network,
		Source:      "cache",
		Confidence:  s.getConfidenceLevel(cache.LookupSource),
		CachedUntil: &cache.CacheExpires,
	}, nil
}

// lookupViaHLR performs HLR lookup via Termii API
func (s *HLRService) lookupViaHLR(ctx context.Context, msisdn string) (*NetworkDetectionResult, error) {
	if s.termiiAPIKey == "" {
		return nil, errors.New("Termii API key not configured")
	}

	// Use a short 3-second deadline for HLR lookup to avoid blocking the recharge flow.
	// If Termii is slow/unreachable, we fall back to prefix detection immediately.
	hlrCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// Termii HLR Lookup API endpoint
	url := fmt.Sprintf("https://api.ng.termii.com/api/check/dnd?api_key=%s&phone_number=%s", s.termiiAPIKey, msisdn)

	req, err := http.NewRequestWithContext(hlrCtx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HLR request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HLR API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HLR API returned status %d: %s", resp.StatusCode, string(body))
	}

	var hlrResponse TermiiHLRResponse
	if err := json.NewDecoder(resp.Body).Decode(&hlrResponse); err != nil {
		return nil, fmt.Errorf("failed to decode HLR response: %w", err)
	}

	// Map Termii network name to our standard format
	network := s.normalizeNetworkName(hlrResponse.Network)
	if network == "" {
		return nil, errors.New("invalid network returned from HLR API")
	}

	// Save to cache
	return s.saveHLRResult(ctx, msisdn, network, "termii", hlrResponse)
}

// saveHLRResult saves HLR lookup result to cache
func (s *HLRService) saveHLRResult(ctx context.Context, msisdn, network, provider string, response interface{}) (*NetworkDetectionResult, error) {
	// Normalize phone to international format (234...) for database storage
	normalizedMSISDN := normalizeToInternational(msisdn)
	
	responseJSON, _ := json.Marshal(response)
	
	now := time.Now()
	cacheExpires := now.AddDate(0, 0, s.cacheTTLDays)

	cache := &entities.NetworkCache{
		MSISDN:       normalizedMSISDN,
		Network:      network,
		LastVerified: now,
		CacheExpires: cacheExpires,
		LookupSource: "hlr_api",
		HLRProvider:  &provider,
		HLRResponse:  datatypes.JSON(responseJSON),
		IsValid:      true,
	}

	// Try to find existing cache entry
	existing, err := s.networkCacheRepo.FindByMSISDN(ctx, normalizedMSISDN)
	if err == nil && existing != nil {
		// Update existing
		cache.ID = existing.ID
		err = s.networkCacheRepo.Update(ctx, cache)
	} else {
		// Create new
		err = s.networkCacheRepo.Create(ctx, cache)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to save HLR result to cache: %w", err)
	}

	return &NetworkDetectionResult{
		MSISDN:      msisdn,
		Network:     network,
		Source:      "hlr_api",
		Confidence:  "high",
		CachedUntil: &cacheExpires,
	}, nil
}

// saveUserSelection saves user-selected network to cache
func (s *HLRService) saveUserSelection(ctx context.Context, msisdn, network string) (*NetworkDetectionResult, error) {
	// Normalize phone to international format (234...) for database storage
	normalizedMSISDN := normalizeToInternational(msisdn)
	
	now := time.Now()
	cacheExpires := now.AddDate(0, 0, s.cacheTTLDays)

	cache := &entities.NetworkCache{
		MSISDN:       normalizedMSISDN,
		Network:      network,
		LastVerified: now,
		CacheExpires: cacheExpires,
		LookupSource: "user_selection",
		IsValid:      true,
	}

	existing, err := s.networkCacheRepo.FindByMSISDN(ctx, normalizedMSISDN)
	if err == nil && existing != nil {
		cache.ID = existing.ID
		err = s.networkCacheRepo.Update(ctx, cache)
	} else {
		err = s.networkCacheRepo.Create(ctx, cache)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to save user selection: %w", err)
	}

	return &NetworkDetectionResult{
		MSISDN:      msisdn,
		Network:     network,
		Source:      "user_selection",
		Confidence:  "medium",
		CachedUntil: &cacheExpires,
	}, nil
}

// savePrefixDetection saves prefix-based detection to cache
func (s *HLRService) savePrefixDetection(ctx context.Context, msisdn, network string) (*NetworkDetectionResult, error) {
	// Normalize phone to international format (234...) for database storage
	normalizedMSISDN := normalizeToInternational(msisdn)
	
	now := time.Now()
	cacheExpires := now.AddDate(0, 0, 7) // Only 7 days for prefix-based (less reliable)

	cache := &entities.NetworkCache{
		MSISDN:       normalizedMSISDN,
		Network:      network,
		LastVerified: now,
		CacheExpires: cacheExpires,
		LookupSource: "prefix_fallback",
		IsValid:      true,
	}

	existing, err := s.networkCacheRepo.FindByMSISDN(ctx, normalizedMSISDN)
	if err == nil && existing != nil {
		cache.ID = existing.ID
		err = s.networkCacheRepo.Update(ctx, cache)
	} else {
		err = s.networkCacheRepo.Create(ctx, cache)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to save prefix detection: %w", err)
	}

	return &NetworkDetectionResult{
		MSISDN:      msisdn,
		Network:     network,
		Source:      "prefix_fallback",
		Confidence:  "low",
		CachedUntil: &cacheExpires,
	}, nil
}

// InvalidateCache invalidates cached network for an MSISDN (called when recharge fails)
func (s *HLRService) InvalidateCache(ctx context.Context, msisdn, reason string) error {
	// Normalize phone to international format for cache lookup
	normalizedMSISDN := normalizeToInternational(msisdn)
	return s.networkCacheRepo.Invalidate(ctx, normalizedMSISDN, reason)
}

// detectByPrefix performs prefix-based network detection (fallback)
func (s *HLRService) detectByPrefix(msisdn string) *NetworkDetectionResult {
	if len(msisdn) < 4 {
		return nil
	}

	prefix := msisdn[:4]
	network := ""

	// MTN prefixes
	mtnPrefixes := []string{"0803", "0806", "0703", "0706", "0813", "0816", "0810", "0814", "0903", "0906", "0913", "0916"}
	for _, p := range mtnPrefixes {
		if prefix == p {
			network = "MTN"
			break
		}
	}

	// Airtel prefixes
	if network == "" {
		airtelPrefixes := []string{"0802", "0808", "0708", "0812", "0701", "0902", "0907", "0901", "0904", "0912"}
		for _, p := range airtelPrefixes {
			if prefix == p {
				network = "Airtel"
				break
			}
		}
	}

	// Glo prefixes
	if network == "" {
		gloPrefixes := []string{"0805", "0807", "0705", "0815", "0811", "0905", "0915"}
		for _, p := range gloPrefixes {
			if prefix == p {
				network = "Glo"
				break
			}
		}
	}

	// 9mobile prefixes
	if network == "" {
		nineMobilePrefixes := []string{"0809", "0817", "0818", "0909", "0908"}
		for _, p := range nineMobilePrefixes {
			if prefix == p {
				network = "9mobile"
				break
			}
		}
	}

	if network == "" {
		return nil
	}

	return &NetworkDetectionResult{
		MSISDN:     msisdn,
		Network:    network,
		Source:     "prefix_fallback",
		Confidence: "low",
	}
}

// normalizeNetworkName converts various network name formats to standard format
func (s *HLRService) normalizeNetworkName(name string) string {
	switch name {
	case "MTN", "mtn", "MTN Nigeria":
		return "MTN"
	case "Airtel", "airtel", "Airtel Nigeria":
		return "Airtel"
	case "Glo", "glo", "Globacom":
		return "Glo"
	case "9mobile", "9Mobile", "Etisalat":
		return "9mobile"
	default:
		return ""
	}
}

// getConfidenceLevel returns confidence level based on lookup source
func (s *HLRService) getConfidenceLevel(source string) string {
	switch source {
	case "hlr_api":
		return "high"
	case "user_selection":
		return "medium"
	case "prefix_fallback":
		return "low"
	default:
		return "unknown"
	}
}

// TermiiHLRResponse represents the response from Termii HLR API
type TermiiHLRResponse struct {
	Number      string `json:"number"`
	Status      string `json:"status"`
	Network     string `json:"network"`
	NetworkCode string `json:"network_code"`
}

// normalizeToInternational converts phone number to international format (234...)
// Accepts: 08031234567 or 2348031234567
// Returns: 2348031234567
func normalizeToInternational(phone string) string {
	// Remove all non-digit characters
	digitsOnly := ""
	for _, char := range phone {
		if char >= '0' && char <= '9' {
			digitsOnly += string(char)
		}
	}
	
	// If starts with 0 (local format), replace with 234
	if len(digitsOnly) == 11 && digitsOnly[0] == '0' {
		return "234" + digitsOnly[1:]
	}
	
	// If already in international format, return as-is
	if len(digitsOnly) == 13 && digitsOnly[:3] == "234" {
		return digitsOnly
	}
	
	// Fallback: return as-is (will fail validation)
	return digitsOnly
}
package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// fraudConfig holds tunable thresholds.
// These can be made configurable via platform_settings in a future iteration.
const (
	maxAmountKobo         int64 = 50_000_000 // ₦500,000 hard ceiling per transaction
	maxTxPerHour                = 15          // velocity: max transactions in 1 hour
	maxFailedRecharge1h         = 5           // too many failures in 1 hour → suspicious
	maxDailyRechargeKobo  int64 = 200_000_000 // ₦2,000,000 daily cap per MSISDN
)

// FraudDetectionService performs lightweight, database-backed fraud checks.
// It is intentionally non-blocking: a DB error is logged but does not block
// the transaction (fail-open strategy to preserve uptime).
type FraudDetectionService struct {
	db *gorm.DB
}

// NewFraudDetectionService creates a FraudDetectionService.
// Pass a *gorm.DB to enable velocity/blacklist checks; nil disables DB checks.
func NewFraudDetectionService(db ...*gorm.DB) *FraudDetectionService {
	svc := &FraudDetectionService{}
	if len(db) > 0 {
		svc.db = db[0]
	}
	return svc
}

// CheckTransaction checks if a single transaction is potentially fraudulent.
// Returns (isFraud bool, reason string, error).
func (s *FraudDetectionService) CheckTransaction(ctx context.Context, msisdn string, amount int64) (bool, string, error) {
	// 1. Hard amount ceiling
	if amount > maxAmountKobo {
		return true, fmt.Sprintf("amount ₦%.2f exceeds maximum allowed ₦%.2f",
			float64(amount)/100, float64(maxAmountKobo)/100), nil
	}

	if s.db == nil {
		return false, "", nil
	}

	// 2. MSISDN blacklist check
	var blacklistCount int64
	if err := s.db.WithContext(ctx).
		Table("msisdn_blacklist").
		Where("msisdn = ? AND is_active = true", msisdn).
		Count(&blacklistCount).Error; err != nil {
		log.Printf("[fraud] blacklist check error: %v", err)
	} else if blacklistCount > 0 {
		return true, "MSISDN is blacklisted", nil
	}

	// 3. Transaction velocity (hourly)
	windowStart := time.Now().Add(-1 * time.Hour)
	var txCount int64
	if err := s.db.WithContext(ctx).
		Table("transactions").
		Where("msisdn = ? AND created_at >= ?", msisdn, windowStart).
		Count(&txCount).Error; err != nil {
		log.Printf("[fraud] velocity check error: %v", err)
	} else if txCount >= maxTxPerHour {
		return true, fmt.Sprintf("transaction velocity exceeded: %d transactions in 1 hour", txCount), nil
	}

	// 4. Daily cumulative amount cap
	dayStart := time.Now().Truncate(24 * time.Hour)
	var dailyTotal struct{ Total int64 }
	if err := s.db.WithContext(ctx).
		Table("transactions").
		Select("COALESCE(SUM(amount), 0) AS total").
		Where("msisdn = ? AND status = 'SUCCESS' AND created_at >= ?", msisdn, dayStart).
		Scan(&dailyTotal).Error; err != nil {
		log.Printf("[fraud] daily cap check error: %v", err)
	} else if dailyTotal.Total+amount > maxDailyRechargeKobo {
		return true, fmt.Sprintf("daily limit exceeded: ₦%.2f cumulative", float64(dailyTotal.Total)/100), nil
	}

	return false, "", nil
}

// CheckRecharge checks if a recharge is potentially fraudulent.
// Returns (isFraud bool, reason string, error).
func (s *FraudDetectionService) CheckRecharge(ctx context.Context, msisdn string, amount int64) (bool, string, error) {
	// Delegate to CheckTransaction — recharge is a subtype of transaction
	isFraud, reason, err := s.CheckTransaction(ctx, msisdn, amount)
	if err != nil || isFraud {
		return isFraud, reason, err
	}

	if s.db == nil {
		return false, "", nil
	}

	// Extra check: too many failed recharges in the last hour (credential stuffing / card testing)
	windowStart := time.Now().Add(-1 * time.Hour)
	var failedCount int64
	if err := s.db.WithContext(ctx).
		Table("transactions").
		Where("msisdn = ? AND status = 'FAILED' AND created_at >= ?", msisdn, windowStart).
		Count(&failedCount).Error; err != nil {
		log.Printf("[fraud] failed-recharge check error: %v", err)
	} else if failedCount >= maxFailedRecharge1h {
		return true, fmt.Sprintf("too many failed recharges: %d in the last hour", failedCount), nil
	}

	return false, "", nil
}
package services

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ─────────────────────────────────────────────────────────────────────────────
// Request / response DTOs
// ─────────────────────────────────────────────────────────────────────────────

// CommissionFilter constrains which transactions are included in a report.
type CommissionFilter struct {
	StartDate string // YYYY-MM-DD
	EndDate   string // YYYY-MM-DD
	Network   string // optional: filter by network name
	Provider  string // optional: filter by provider name
}

// CommissionReport is the full reconciliation payload.
type CommissionReport struct {
	Summary            CommissionSummary            `json:"summary"`
	ByNetwork          []CommissionByNetwork         `json:"by_network"`
	ByProvider         []CommissionByProvider        `json:"by_provider"`
	ByDate             []CommissionByDate             `json:"by_date"`
	RecentTransactions []CommissionTransaction        `json:"recent_transactions"`
}

// CommissionSummary aggregates across the entire filtered period.
type CommissionSummary struct {
	TotalTransactions   int64   `json:"total_transactions"`
	TotalRechargeAmount int64   `json:"total_recharge_amount"`
	TotalCommission     int64   `json:"total_commission"`
	AverageCommission   float64 `json:"average_commission"`
	CommissionRate      float64 `json:"commission_rate"`
}

// CommissionByNetwork breaks down commission per network operator.
type CommissionByNetwork struct {
	Network           string  `json:"network"`
	TransactionCount  int64   `json:"transaction_count"`
	TotalAmount       int64   `json:"total_amount"`
	TotalCommission   int64   `json:"total_commission"`
	AverageCommission float64 `json:"average_commission"`
	CommissionRate    float64 `json:"commission_rate"`
}

// CommissionByProvider breaks down commission per recharge provider.
type CommissionByProvider struct {
	Provider          string  `json:"provider"`
	TransactionCount  int64   `json:"transaction_count"`
	TotalAmount       int64   `json:"total_amount"`
	TotalCommission   int64   `json:"total_commission"`
	AverageCommission float64 `json:"average_commission"`
	CommissionRate    float64 `json:"commission_rate"`
}

// CommissionByDate aggregates per calendar day.
type CommissionByDate struct {
	Date             string `json:"date"`
	TransactionCount int64  `json:"transaction_count"`
	TotalAmount      int64  `json:"total_amount"`
	TotalCommission  int64  `json:"total_commission"`
}

// CommissionTransaction is a single masked transaction line item.
type CommissionTransaction struct {
	ID             string    `json:"id"`
	MSISDN         string    `json:"msisdn"` // masked: 0801****234
	Network        string    `json:"network"`
	Provider       string    `json:"provider"`
	Amount         int64     `json:"amount"`
	Commission     int64     `json:"commission"`
	CommissionRate float64   `json:"commission_rate"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

// ─────────────────────────────────────────────────────────────────────────────
// CommissionService
// ─────────────────────────────────────────────────────────────────────────────

// CommissionService runs commission-reconciliation queries.
type CommissionService struct {
	db *gorm.DB
}

// NewCommissionService constructs a CommissionService.
func NewCommissionService(db *gorm.DB) *CommissionService {
	return &CommissionService{db: db}
}

// GetReconciliation builds the full commission report for the given filter.
func (s *CommissionService) GetReconciliation(ctx context.Context, f CommissionFilter) (*CommissionReport, error) {
	start, end, err := parseDateRange(f.StartDate, f.EndDate)
	if err != nil {
		return nil, err
	}
	db := s.db.WithContext(ctx)
	report := &CommissionReport{}

	// ── Summary ───────────────────────────────────────────────────────────────
	type summaryRow struct {
		TotalTransactions   int64
		TotalRechargeAmount int64
		TotalCommission     int64
	}
	var sum summaryRow
	q := db.Table("transactions").
		Select("COUNT(*) AS total_transactions, COALESCE(SUM(amount),0) AS total_recharge_amount, COALESCE(SUM(commission_amount),0) AS total_commission").
		Where("created_at BETWEEN ? AND ? AND status = 'SUCCESS'", start, end)
	if f.Network != "" {
		q = q.Where("network = ?", strings.ToUpper(f.Network))
	}
	if f.Provider != "" {
		q = q.Where("provider = ?", f.Provider)
	}
	q.Scan(&sum)

	report.Summary = CommissionSummary{
		TotalTransactions:   sum.TotalTransactions,
		TotalRechargeAmount: sum.TotalRechargeAmount,
		TotalCommission:     sum.TotalCommission,
	}
	if sum.TotalTransactions > 0 {
		report.Summary.AverageCommission = float64(sum.TotalCommission) / float64(sum.TotalTransactions)
	}
	if sum.TotalRechargeAmount > 0 {
		report.Summary.CommissionRate = float64(sum.TotalCommission) / float64(sum.TotalRechargeAmount) * 100
	}

	// ── By Network ────────────────────────────────────────────────────────────
	type netRow struct {
		Network          string
		TransactionCount int64
		TotalAmount      int64
		TotalCommission  int64
	}
	var netRows []netRow
	db.Table("transactions").
		Select("network, COUNT(*) AS transaction_count, COALESCE(SUM(amount),0) AS total_amount, COALESCE(SUM(commission_amount),0) AS total_commission").
		Where("created_at BETWEEN ? AND ? AND status = 'SUCCESS'", start, end).
		Group("network").
		Scan(&netRows)
	for _, r := range netRows {
		avg, rate := commRates(r.TotalCommission, r.TransactionCount, r.TotalAmount)
		report.ByNetwork = append(report.ByNetwork, CommissionByNetwork{
			Network: r.Network, TransactionCount: r.TransactionCount,
			TotalAmount: r.TotalAmount, TotalCommission: r.TotalCommission,
			AverageCommission: avg, CommissionRate: rate,
		})
	}

	// ── By Provider ───────────────────────────────────────────────────────────
	type provRow struct {
		Provider         string
		TransactionCount int64
		TotalAmount      int64
		TotalCommission  int64
	}
	var provRows []provRow
	db.Table("transactions").
		Select("provider, COUNT(*) AS transaction_count, COALESCE(SUM(amount),0) AS total_amount, COALESCE(SUM(commission_amount),0) AS total_commission").
		Where("created_at BETWEEN ? AND ? AND status = 'SUCCESS'", start, end).
		Group("provider").
		Scan(&provRows)
	for _, r := range provRows {
		avg, rate := commRates(r.TotalCommission, r.TransactionCount, r.TotalAmount)
		report.ByProvider = append(report.ByProvider, CommissionByProvider{
			Provider: r.Provider, TransactionCount: r.TransactionCount,
			TotalAmount: r.TotalAmount, TotalCommission: r.TotalCommission,
			AverageCommission: avg, CommissionRate: rate,
		})
	}

	// ── By Date ───────────────────────────────────────────────────────────────
	type dateRow struct {
		Day              string
		TransactionCount int64
		TotalAmount      int64
		TotalCommission  int64
	}
	var dateRows []dateRow
	db.Table("transactions").
		Select("DATE(created_at) AS day, COUNT(*) AS transaction_count, COALESCE(SUM(amount),0) AS total_amount, COALESCE(SUM(commission_amount),0) AS total_commission").
		Where("created_at BETWEEN ? AND ? AND status = 'SUCCESS'", start, end).
		Group("DATE(created_at)").
		Order("day ASC").
		Scan(&dateRows)
	for _, r := range dateRows {
		report.ByDate = append(report.ByDate, CommissionByDate{
			Date: r.Day, TransactionCount: r.TransactionCount,
			TotalAmount: r.TotalAmount, TotalCommission: r.TotalCommission,
		})
	}

	// ── Recent Transactions ───────────────────────────────────────────────────
	type txnRow struct {
		ID            string
		MSISDN        string
		Network       string
		Provider      string
		Amount        int64
		CommissionAmt int64
		Status        string
		CreatedAt     time.Time
	}
	var txns []txnRow
	db.Table("transactions").
		Select("id, msisdn, network, provider, amount, commission_amount AS commission_amt, status, created_at").
		Where("created_at BETWEEN ? AND ? AND status = 'SUCCESS'", start, end).
		Order("created_at DESC").
		Limit(20).
		Scan(&txns)
	for _, t := range txns {
		_, rate := commRates(t.CommissionAmt, 1, t.Amount)
		msisdn := t.MSISDN
		if len(msisdn) > 7 {
			msisdn = msisdn[:4] + "****" + msisdn[len(msisdn)-3:]
		}
		report.RecentTransactions = append(report.RecentTransactions, CommissionTransaction{
			ID: t.ID, MSISDN: msisdn, Network: t.Network, Provider: t.Provider,
			Amount: t.Amount, Commission: t.CommissionAmt, CommissionRate: rate,
			Status: t.Status, CreatedAt: t.CreatedAt,
		})
	}

	return report, nil
}

// ExportCSV returns a CSV byte slice for the given filter.
func (s *CommissionService) ExportCSV(ctx context.Context, f CommissionFilter) ([]byte, error) {
	start, end, err := parseDateRange(f.StartDate, f.EndDate)
	if err != nil {
		return nil, err
	}

	type txnRow struct {
		CreatedAt     time.Time
		ID            string
		MSISDN        string
		Network       string
		Provider      string
		Amount        int64
		CommissionAmt int64
		Status        string
	}
	var txns []txnRow
	s.db.WithContext(ctx).Table("transactions").
		Select("created_at, id, msisdn, network, provider, amount, commission_amount AS commission_amt, status").
		Where("created_at BETWEEN ? AND ? AND status = 'SUCCESS'", start, end).
		Order("created_at ASC").
		Scan(&txns)

	var sb strings.Builder
	sb.WriteString("Date,Transaction ID,Phone Number,Network,Provider,Amount (₦),Commission (₦),Commission Rate (%),Status\n")
	for _, t := range txns {
		commRate := float64(0)
		if t.Amount > 0 {
			commRate = float64(t.CommissionAmt) / float64(t.Amount) * 100
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%.2f,%.2f,%.2f,%s\n",
			t.CreatedAt.Format("2006-01-02"),
			t.ID, t.MSISDN, t.Network, t.Provider,
			float64(t.Amount)/100,
			float64(t.CommissionAmt)/100,
			commRate, t.Status,
		))
	}
	return []byte(sb.String()), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func parseDateRange(startStr, endStr string) (time.Time, time.Time, error) {
	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid start_date: use YYYY-MM-DD")
	}
	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid end_date: use YYYY-MM-DD")
	}
	end = end.Add(24*time.Hour - time.Second)
	return start, end, nil
}

func commRates(commission, txCount, amount int64) (avg, rate float64) {
	if txCount > 0 {
		avg = math.Round(float64(commission)/float64(txCount)*100) / 100
	}
	if amount > 0 {
		rate = math.Round(float64(commission)/float64(amount)*10000) / 100
	}
	return
}
