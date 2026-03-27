package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/external"
)

type AdminClaimService struct {
	prizeRepo repositories.PrizeRepository
	momo      external.MoMoPayer
}

func NewAdminClaimService(prizeRepo repositories.PrizeRepository, momo external.MoMoPayer) *AdminClaimService {
	return &AdminClaimService{
		prizeRepo: prizeRepo,
		momo:      momo,
	}
}

func (s *AdminClaimService) ListClaims(ctx context.Context, status string, limit, offset int) ([]entities.SpinResult, int64, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.prizeRepo.ListAdminClaims(ctx, status, limit, offset)
}

func (s *AdminClaimService) GetClaimDetails(ctx context.Context, claimID uuid.UUID) (*entities.SpinResult, error) {
	return s.prizeRepo.FindSpinResult(ctx, claimID)
}

type ApproveClaimRequest struct {
	AdminNotes       string `json:"admin_notes"`
	PaymentReference string `json:"payment_reference"`
}

func (s *AdminClaimService) ApproveClaim(ctx context.Context, claimID, adminID uuid.UUID, req ApproveClaimRequest) (*entities.SpinResult, error) {
	claim, err := s.prizeRepo.FindSpinResult(ctx, claimID)
	if err != nil {
		return nil, fmt.Errorf("claim not found: %w", err)
	}

	if claim.ClaimStatus != entities.ClaimPendingAdmin {
		return nil, fmt.Errorf("claim is not pending admin review (current status: %s)", claim.ClaimStatus)
	}

	// If it's a MoMo cash prize, we have a MoMo number, and MoMo client is wired — disburse automatically.
	if claim.PrizeType == entities.PrizeMoMoCash && claim.MoMoClaimNumber != "" && s.momo != nil {
		ref := "LN_CLAIM_" + claim.ID.String()
		momoRef, err := s.momo.Disburse(ctx, claim.MoMoClaimNumber, int64(claim.PrizeValue), ref)
		if err != nil {
			return nil, fmt.Errorf("momo disbursement failed: %w", err)
		}
		if req.PaymentReference == "" {
			req.PaymentReference = momoRef
		}
		_ = s.prizeRepo.UpdateSpinFulfillment(ctx, claimID, entities.FulfillCompleted, momoRef, "")
	} else if claim.PrizeType == entities.PrizeMoMoCash {
		// Manual bank transfer case
		_ = s.prizeRepo.UpdateSpinFulfillment(ctx, claimID, entities.FulfillCompleted, req.PaymentReference, "")
	}

	err = s.prizeRepo.UpdateAdminClaimReview(ctx, claimID, entities.ClaimApproved, adminID, req.AdminNotes, "", req.PaymentReference)
	if err != nil {
		return nil, err
	}

	return s.prizeRepo.FindSpinResult(ctx, claimID)
}

type RejectClaimRequest struct {
	RejectionReason string `json:"rejection_reason"`
	AdminNotes      string `json:"admin_notes"`
}

func (s *AdminClaimService) RejectClaim(ctx context.Context, claimID, adminID uuid.UUID, req RejectClaimRequest) (*entities.SpinResult, error) {
	claim, err := s.prizeRepo.FindSpinResult(ctx, claimID)
	if err != nil {
		return nil, fmt.Errorf("claim not found: %w", err)
	}

	if claim.ClaimStatus != entities.ClaimPendingAdmin {
		return nil, fmt.Errorf("claim is not pending admin review (current status: %s)", claim.ClaimStatus)
	}

	if req.RejectionReason == "" {
		return nil, fmt.Errorf("rejection reason is required")
	}

	err = s.prizeRepo.UpdateAdminClaimReview(ctx, claimID, entities.ClaimRejected, adminID, req.AdminNotes, req.RejectionReason, "")
	if err != nil {
		return nil, err
	}

	_ = s.prizeRepo.UpdateSpinFulfillment(ctx, claimID, entities.FulfillFailed, "", req.RejectionReason)

	return s.prizeRepo.FindSpinResult(ctx, claimID)
}

// GetPendingClaims returns all claims in PENDING_ADMIN_REVIEW status.
func (s *AdminClaimService) GetPendingClaims(ctx context.Context) ([]entities.SpinResult, error) {
	results, _, err := s.prizeRepo.ListAdminClaims(ctx, string(entities.ClaimPendingAdmin), 1000, 0)
	return results, err
}

// ClaimStatistics holds aggregated claim counts and amounts.
type ClaimStatistics struct {
	TotalClaims    int64   `json:"total_claims"`
	PendingClaims  int64   `json:"pending_claims"`
	ApprovedClaims int64   `json:"approved_claims"`
	RejectedClaims int64   `json:"rejected_claims"`
	ClaimedClaims  int64   `json:"claimed_claims"`
	ExpiredClaims  int64   `json:"expired_claims"`
	TotalValueNGN  float64 `json:"total_value_ngn"`
	ApprovedValueNGN float64 `json:"approved_value_ngn"`
	PendingValueNGN  float64 `json:"pending_value_ngn"`
}

// GetStatistics returns aggregated claim statistics.
func (s *AdminClaimService) GetStatistics(ctx context.Context) (*ClaimStatistics, error) {
	type countRow struct {
		Status string
		Count  int64
		Total  float64
	}
	var rows []countRow
	err := s.prizeRepo.AggregateClaimStats(ctx, &rows)
	if err != nil {
		return nil, err
	}

	stats := &ClaimStatistics{}
	for _, r := range rows {
		switch entities.ClaimStatus(r.Status) {
		case entities.ClaimPending:
			stats.PendingClaims = r.Count
			stats.PendingValueNGN = r.Total / 100 // kobo → naira
		case entities.ClaimPendingAdmin:
			stats.PendingClaims += r.Count
			stats.PendingValueNGN += r.Total / 100
		case entities.ClaimApproved:
			stats.ApprovedClaims = r.Count
			stats.ApprovedValueNGN = r.Total / 100
		case entities.ClaimRejected:
			stats.RejectedClaims = r.Count
		case entities.ClaimClaimed:
			stats.ClaimedClaims = r.Count
		case entities.ClaimExpired:
			stats.ExpiredClaims = r.Count
		}
		stats.TotalClaims += r.Count
		stats.TotalValueNGN += r.Total / 100
	}
	return stats, nil
}

// ExportCSV returns all claims as a CSV-formatted string.
func (s *AdminClaimService) ExportCSV(ctx context.Context, status string) (string, error) {
	claims, _, err := s.prizeRepo.ListAdminClaims(ctx, status, 100000, 0)
	if err != nil {
		return "", err
	}

	header := "id,user_id,prize_type,prize_value_ngn,claim_status,momo_claim_number,bank_account_number,bank_account_name,bank_name,reviewed_by,reviewed_at,admin_notes,rejection_reason,payment_reference,expires_at,created_at\n"
	rows := header
	for _, c := range claims {
		reviewedBy := ""
		if c.ReviewedBy != nil {
			reviewedBy = c.ReviewedBy.String()
		}
		reviewedAt := ""
		if c.ReviewedAt != nil {
			reviewedAt = c.ReviewedAt.Format("2006-01-02T15:04:05Z")
		}
		rows += fmt.Sprintf("%s,%s,%s,%.2f,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s\n",
			c.ID.String(),
			c.UserID.String(),
			string(c.PrizeType),
			c.PrizeValue/100, // kobo → naira
			string(c.ClaimStatus),
			c.MoMoClaimNumber,
			c.BankAccountNumber,
			c.BankAccountName,
			c.BankName,
			reviewedBy,
			reviewedAt,
			c.AdminNotes,
			c.RejectionReason,
			c.PaymentReference,
			c.ExpiresAt.Format("2006-01-02T15:04:05Z"),
			c.CreatedAt.Format("2006-01-02T15:04:05Z"),
		)
	}
	return rows, nil
}
