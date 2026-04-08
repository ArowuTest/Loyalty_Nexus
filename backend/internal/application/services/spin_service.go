package services

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"time"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/config"
	"loyalty-nexus/internal/pkg/safe"
	"loyalty-nexus/internal/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SpinService struct {
	userRepo    repositories.UserRepository
	txRepo      repositories.TransactionRepository
	prizeRepo   repositories.PrizeRepository
	fulfillSvc  *PrizeFulfillmentService
	notifySvc   *NotificationService
	cfg         *config.ConfigManager
	db          *gorm.DB
}

func NewSpinService(
	ur repositories.UserRepository,
	tr repositories.TransactionRepository,
	pr repositories.PrizeRepository,
	fs *PrizeFulfillmentService,
	ns *NotificationService,
	cfg *config.ConfigManager,
	db *gorm.DB,
) *SpinService {
	return &SpinService{
		userRepo:   ur,
		txRepo:     tr,
		prizeRepo:  pr,
		fulfillSvc: fs,
		notifySvc:  ns,
		cfg:        cfg,
		db:         db,
	}
}

type SpinOutcome struct {
	SpinResult  *entities.SpinResult `json:"spin_result"`
	PrizeLabel  string               `json:"prize_label"`
	SlotIndex   int                  `json:"slot_index"`
	Message     string               `json:"message"`
	NeedsMoMo   bool                 `json:"needs_momo_setup"` // Prompt user to link MoMo
}

// PlaySpin executes a single spin:
// 0. Verify user has ≥1 spin credit (REQ-3.1) — checked first to give clear error
// 1. Validate daily limit (REQ-3.6)
// 2. Check daily liability cap (REQ-3.5) — force low value if hit
// 3. Select prize via CSPRNG (REQ-3.2)
// 4. Atomically deduct credit + write spin result + ledger entry
// 5. Dispatch fulfillment in background goroutine
func (s *SpinService) PlaySpin(ctx context.Context, userID uuid.UUID) (*SpinOutcome, error) {
	// --- Step 0: Wallet check first — gives clear error before any daily-limit math ---
	wallet, err := s.userRepo.GetWalletForUpdate(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("wallet not found")
	}
	if wallet.SpinCredits < 1 {
		return nil, fmt.Errorf("no spin credits available — recharge ₦1,000 or more to earn a free spin")
	}

	// --- Step 1: Daily spin limit based on tier ---
	todayMidnight := time.Now().UTC().Truncate(24 * time.Hour).Unix()
	todayAmountKobo, err := s.txRepo.SumAmountByUserSince(ctx, userID, todayMidnight)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate daily recharge: %w", err)
	}

	tierCalc := utils.NewSpinTierCalculatorDB(s.db)
	dailyCap := 0
	if tier, err := tierCalc.GetSpinTierFromDB(todayAmountKobo); err == nil && tier.SpinsPerDay > 0 {
		dailyCap = tier.SpinsPerDay
	}

	dailySpins, err := s.prizeRepo.CountUserSpinsToday(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("spin count check failed: %w", err)
	}
	if dailySpins >= dailyCap {
		return nil, fmt.Errorf("daily spin limit reached (%d/%d) — recharge more today to unlock additional spins", dailySpins, dailyCap)
	}

	// --- Step 2: Daily liability cap ---
	capNaira := s.cfg.GetInt64("daily_prize_liability_cap_naira", 500000)
	capKobo := capNaira * 100
	currentLiability, _ := s.txRepo.DailyLiabilityTotal(ctx)
	forceLowValue := currentLiability >= capKobo

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// --- Step 4: Select prize via CSPRNG (REQ-3.2) ---
	prize, slotIdx, err := s.selectPrize(ctx, forceLowValue, todayAmountKobo)
	if err != nil {
		return nil, fmt.Errorf("prize selection failed: %w", err)
	}

	// --- Step 5: Atomic DB transaction ---
	var spinResult *entities.SpinResult
	err = s.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		// Deduct 1 spin credit (use dbTx directly to avoid nested transaction on SQLite)
		if err := dbTx.Table("wallets").Where("user_id = ?", wallet.UserID).
			UpdateColumn("spin_credits", gorm.Expr("spin_credits - 1")).Error; err != nil {
			return fmt.Errorf("credit deduction failed: %w", err)
		}
		wallet.SpinCredits--

		// If this is a no-win slot, skip DB write entirely (RechargeMax pattern)
		if prize.IsNoWin {
			// Deduct spin credit but create no spin_results row
			spinResult = &entities.SpinResult{
				ID:                uuid.New(),
				UserID:            userID,
				PrizeType:         entities.PrizeTryAgain,
				PrizeValue:        0,
				SlotIndex:         slotIdx,
				FulfillmentStatus: entities.FulfillNA,
				ClaimStatus:       entities.ClaimClaimed,
				CreatedAt:         time.Now(),
			}
			// Still write a minimal spin_results row so CountUserSpinsToday works
			if err := s.prizeRepo.CreateSpinResultTx(ctx, dbTx, spinResult); err != nil {
				return fmt.Errorf("spin result write failed: %w", err)
			}
			return nil
		}

		// Determine initial fulfillment status and claim status
		fulfillStatus := entities.FulfillPending
		claimStatus := entities.ClaimPending

		switch prize.PrizeType {
		case entities.PrizeTryAgain:
			fulfillStatus = entities.FulfillNA
			claimStatus = entities.ClaimClaimed // No claim needed
		case entities.PrizePulsePoints:
			// Points are auto-credited immediately, no claim needed
			claimStatus = entities.ClaimClaimed
		case entities.PrizeAirtime, entities.PrizeDataBundle:
			// Airtime and Data require user to click "Claim" before fulfillment
			fulfillStatus = entities.FulfillPendingClaim
		case entities.PrizeMoMoCash:
			if !user.MoMoVerified {
				fulfillStatus = entities.FulfillPendingMoMo
			} else {
				// Even if verified, they still need to claim it via the dashboard
				fulfillStatus = entities.FulfillPendingClaim
			}
		}

		spinResult = &entities.SpinResult{
			ID:                uuid.New(),
			UserID:            userID,
			PrizeType:         prize.PrizeType,
			PrizeValue:        prize.BaseValue,
			SlotIndex:         slotIdx,
			FulfillmentStatus: fulfillStatus,
			ClaimStatus:       claimStatus,
			ExpiresAt:         time.Now().Add(30 * 24 * time.Hour),
			CreatedAt:         time.Now(),
		}
		if err := s.prizeRepo.CreateSpinResultTx(ctx, dbTx, spinResult); err != nil {
			return fmt.Errorf("spin result write failed: %w", err)
		}

		// Ledger entry: spin play
		spinMeta, _ := json.Marshal(map[string]interface{}{
			"prize_type":  prize.PrizeType,
			"prize_value": prize.BaseValue,
			"spin_id":     spinResult.ID.String(),
		})
		spinTx := &entities.Transaction{
			ID:          uuid.New(),
			UserID:      userID,
			PhoneNumber: user.PhoneNumber,
			Type:        entities.TxTypeSpinPlay,
			SpinDelta:   -1,
			Reference:   "spin_" + spinResult.ID.String(),
			Metadata:    spinMeta,
			CreatedAt:   time.Now(),
		}
		if err := s.txRepo.SaveTx(ctx, dbTx, spinTx); err != nil {
			return err
		}

			// For Pulse Points prizes, award immediately via ledger
			if prize.PrizeType == entities.PrizePulsePoints {
				pts := int64(prize.BaseValue)
				if err := dbTx.Table("wallets").Where("user_id = ?", wallet.UserID).Updates(map[string]interface{}{
					"pulse_points":    gorm.Expr("pulse_points + ?", pts),
					"lifetime_points": gorm.Expr("lifetime_points + ?", pts),
				}).Error; err != nil {
					return err
				}
				wallet.PulsePoints += pts
				wallet.LifetimePoints += pts
			ptsTx := &entities.Transaction{
				ID:           uuid.New(),
				UserID:       userID,
				PhoneNumber:  user.PhoneNumber,
				Type:         entities.TxTypePrizeAward,
				PointsDelta:  pts,
				BalanceAfter: wallet.PulsePoints,
				Reference:    "prize_pts_" + spinResult.ID.String(),
				CreatedAt:    time.Now(),
			}
			if err := s.txRepo.SaveTx(ctx, dbTx, ptsTx); err != nil {
				return err
			}
			if err := s.prizeRepo.UpdateSpinFulfillment(ctx, spinResult.ID, entities.FulfillCompleted, "", ""); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

		// --- Step 6: Background fulfillment for physical prizes ---
		if spinResult.FulfillmentStatus == entities.FulfillPending {
			safe.Go(func() {
				if err := s.fulfillSvc.Fulfill(context.Background(), spinResult); err != nil {
					log.Printf("[SPIN] Fulfillment failed for %s: %v", spinResult.ID, err)
				}
			})
		}

	// Build outcome response
	outcome := &SpinOutcome{
		SpinResult: spinResult,
		PrizeLabel: s.buildPrizeLabel(prize),
		SlotIndex:  slotIdx,
		NeedsMoMo:  spinResult.FulfillmentStatus == entities.FulfillPendingMoMo,
	}
	outcome.Message = s.buildWinMessage(outcome, user.PhoneNumber)

	return outcome, nil
}

// GetWheelConfig returns the assembled spin wheel for the frontend.
// Prizes are ordered by sort_order; each slot uses the admin-configured color_scheme.
func (s *SpinService) GetWheelConfig(ctx context.Context) (*entities.SpinWheelPayload, error) {
	prizes, err := s.prizeRepo.ListActivePrizesSorted(ctx)
	if err != nil {
		return nil, err
	}

	fallbackColors := []string{"#FF6B35", "#FFD700", "#00B4D8", "#06D6A0", "#EF476F", "#118AB2", "#073B4C", "#FFB703"}
	slots := make([]entities.SpinSlot, len(prizes))
	for i, p := range prizes {
		color := p.ColorScheme
		if color == "" {
			color = fallbackColors[i%len(fallbackColors)]
		}
		slots[i] = entities.SpinSlot{
			Index:     i,
			PrizeType: p.PrizeType,
			Label:     s.buildPrizeLabel(&p),
			Color:     color,
			IconName:  p.IconName,
			IsNoWin:   p.IsNoWin,
			NoWinMsg:  p.NoWinMessage,
		}
	}
	return &entities.SpinWheelPayload{
		Slots:           slots,
		RequiredCredits: 1,
	}, nil
}

// selectPrize uses CSPRNG weighted random selection.
func (s *SpinService) selectPrize(ctx context.Context, forceLowValue bool, todayAmountKobo int64) (*entities.PrizePoolEntry, int, error) {
	var prizes []entities.PrizePoolEntry
	var err error

	if forceLowValue {
		prizes, err = s.prizeRepo.ListActivePrizesMaxValue(ctx, 5000) // Max ₦50 when cap hit
	} else {
		prizes, err = s.prizeRepo.ListActivePrizes(ctx)
	}
	if err != nil || len(prizes) == 0 {
		return nil, 0, fmt.Errorf("no active prizes available")
	}

	// Check daily inventory caps and minimum recharge
	eligible := make([]entities.PrizePoolEntry, 0, len(prizes))
	for _, p := range prizes {
		if p.MinimumRecharge > 0 && todayAmountKobo < p.MinimumRecharge {
			continue // User hasn't recharged enough today for this prize
		}
		if p.DailyInventoryCap != nil {
			used, _ := s.prizeRepo.GetDailyInventoryUsed(ctx, p.ID)
			if used >= *p.DailyInventoryCap {
				continue // Inventory exhausted for this prize today
			}
		}
		eligible = append(eligible, p)
	}
	if len(eligible) == 0 {
		eligible = prizes // Fallback: no inventory caps remain, use all
	}

	// Weighted CSPRNG selection — weights are NUMERIC(5,2) summing to 100.00
	// Scale to integer precision (multiply by 100 → range 0–10000) for rand.Int
	totalWeightF := 0.0
	for _, p := range eligible {
		totalWeightF += p.ProbWeight
	}
	if totalWeightF == 0 {
		return nil, 0, fmt.Errorf("all prizes have zero weight")
	}
	totalWeightInt := int64(totalWeightF * 100)
	roll, _ := rand.Int(rand.Reader, big.NewInt(totalWeightInt))
	cursor := int64(0)
	for i, p := range eligible {
		cursor += int64(p.ProbWeight * 100)
		if roll.Int64() < cursor {
			return &eligible[i], i, nil
		}
	}
	return &eligible[0], 0, nil
}

func (s *SpinService) buildPrizeLabel(p *entities.PrizePoolEntry) string {
	switch p.PrizeType {
	case entities.PrizeTryAgain:
		return "Try Again"
	case entities.PrizePulsePoints:
		return fmt.Sprintf("+%.0f Points", p.BaseValue)
	case entities.PrizeAirtime:
		return fmt.Sprintf("₦%.0f Airtime", p.BaseValue)
	case entities.PrizeDataBundle:
		return fmt.Sprintf("%.0fMB Data", p.BaseValue)
	case entities.PrizeMoMoCash:
		return fmt.Sprintf("₦%.0f MoMo Cash", p.BaseValue)
	default:
		return p.Name
	}
}

func (s *SpinService) buildWinMessage(o *SpinOutcome, phone string) string {
	if o.SpinResult.PrizeType == entities.PrizeTryAgain {
		return "Better luck next time! Keep recharging to earn more spins."
	}
	if o.NeedsMoMo {
		return fmt.Sprintf("You won %s! Link your MTN MoMo number to claim your cash prize.", o.PrizeLabel)
	}
	return fmt.Sprintf("Congratulations! You won %s!", o.PrizeLabel)
}

// ─── Admin Prize CRUD ─────────────────────────────────────────────────────

// GetAllPrizes returns prizes from the prize_pool table.
// When includeInactive is true, inactive prizes are included (admin view).
func (s *SpinService) GetAllPrizes(ctx context.Context, includeInactive ...bool) ([]entities.PrizePoolEntry, error) {
	var prizes []entities.PrizePoolEntry
	q := s.db.WithContext(ctx).Table("prize_pool")
	if len(includeInactive) == 0 || !includeInactive[0] {
		q = q.Where("is_active = ?", true)
	}
	err := q.Order("sort_order ASC, win_probability_weight DESC").Find(&prizes).Error
	return prizes, err
}

// PrizeProbabilitySummary is returned by GetPrizeProbabilitySummary.
type PrizeProbabilitySummary struct {
	TotalWeight     float64                `json:"total_weight"`      // sum of all active weights (max 100.00)
	RemainingBudget float64                `json:"remaining_budget"` // 100.00 - TotalWeight
	PercentUsed     float64                `json:"percent_used"`     // same as TotalWeight (already a percentage)
	Prizes          []PrizeProbabilityItem `json:"prizes"`
}

// PrizeProbabilityItem is one row in the probability summary.
type PrizeProbabilityItem struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	PrizeType   string  `json:"prize_type"`
	Weight      float64 `json:"weight"`   // NUMERIC(5,2) — directly the percentage (e.g. 25.00 = 25%)
	Percent     float64 `json:"percent"` // same as Weight for backward compat
	IsActive    bool    `json:"is_active"`
	IsNoWin     bool    `json:"is_no_win"`
	ColorScheme string  `json:"color_scheme"`
	SortOrder   int     `json:"sort_order"`
}

// GetPrizeProbabilitySummary returns the probability budget breakdown for the admin wheel editor.
func (s *SpinService) GetPrizeProbabilitySummary(ctx context.Context) (*PrizeProbabilitySummary, error) {
	prizes, err := s.GetAllPrizes(ctx, true) // include inactive
	if err != nil {
		return nil, err
	}
	totalWeight := 0.0
	items := make([]PrizeProbabilityItem, 0, len(prizes))
	for _, p := range prizes {
		if p.IsActive {
			totalWeight += p.ProbWeight
		}
		items = append(items, PrizeProbabilityItem{
			ID:          p.ID.String(),
			Name:        p.Name,
			PrizeType:   string(p.PrizeType),
			Weight:      p.ProbWeight,
			Percent:     p.ProbWeight, // weight IS the percent (e.g. 25.00 = 25%)
			IsActive:    p.IsActive,
			IsNoWin:     p.IsNoWin,
			ColorScheme: p.ColorScheme,
			SortOrder:   p.SortOrder,
		})
	}
	return &PrizeProbabilitySummary{
		TotalWeight:     totalWeight,
		RemainingBudget: 100.00 - totalWeight,
		PercentUsed:     totalWeight, // already a percentage
		Prizes:          items,
	}, nil
}

// ReorderPrizes updates the sort_order of prizes in bulk.
// orderedIDs is a slice of prize UUIDs in the desired display order (index 0 = first on wheel).
func (s *SpinService) ReorderPrizes(ctx context.Context, orderedIDs []uuid.UUID) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, id := range orderedIDs {
			if err := tx.Table("prize_pool").Where("id = ?", id).Update("sort_order", i).Error; err != nil {
				return fmt.Errorf("reorder prize %s: %w", id, err)
			}
		}
		return nil
	})
}

// GetPrize returns a single prize by ID.
func (s *SpinService) GetPrize(ctx context.Context, prizeID uuid.UUID) (*entities.PrizePoolEntry, error) {
	var p entities.PrizePoolEntry
	if err := s.db.WithContext(ctx).Table("prize_pool").Where("id = ?", prizeID).First(&p).Error; err != nil {
		return nil, fmt.Errorf("prize not found: %w", err)
	}
	return &p, nil
}

// CreatePrize creates a new prize slot (admin).
// Validates that total ProbWeight of all active prizes ≤ 100.00 (representing 100%).
func (s *SpinService) CreatePrize(ctx context.Context, data map[string]interface{}) (*entities.PrizePoolEntry, error) {
	name, _ := data["name"].(string)
	if name == "" {
		return nil, fmt.Errorf("prize name is required")
	}
	prizeTypeStr, _ := data["prize_type"].(string)
	if prizeTypeStr == "" {
		return nil, fmt.Errorf("prize_type is required")
	}
	baseValue, _ := data["base_value"].(float64)
	probWeight := 0.0
	if pw, ok := data["win_probability_weight"].(float64); ok {
		probWeight = pw
	}
	isActive := true
	if ia, ok := data["is_active"].(bool); ok {
		isActive = ia
	}

	// Validate total weight doesn't exceed 100.00 (= 100%)
	if s.db != nil {
		var currentTotal float64
		s.db.WithContext(ctx).Table("prize_pool").
			Where("is_active = true").
			Select("COALESCE(SUM(win_probability_weight), 0)").
			Scan(&currentTotal)
		if currentTotal+probWeight > 100.00 {
			return nil, fmt.Errorf("adding this prize (%.2f%%) would exceed 100%% total (current: %.2f%%/100%%)", probWeight, currentTotal)
		}
	}

	prize := entities.PrizePoolEntry{
		ID:         uuid.New(),
		Name:       name,
		PrizeType:  entities.PrizeType(prizeTypeStr),
		BaseValue:  baseValue,
		IsActive:   isActive,
		ProbWeight: probWeight,
	}
	if cap, ok := data["daily_inventory_cap"].(float64); ok {
		capInt := int(cap)
		prize.DailyInventoryCap = &capInt
	}
	if isNoWin, ok := data["is_no_win"].(bool); ok {
		prize.IsNoWin = isNoWin
	}
	if noWinMsg, ok := data["no_win_message"].(string); ok {
		prize.NoWinMessage = noWinMsg
	}
	if color, ok := data["color_scheme"].(string); ok {
		prize.ColorScheme = color
	}
	if sortOrder, ok := data["sort_order"].(float64); ok {
		prize.SortOrder = int(sortOrder)
	}
	if minRecharge, ok := data["minimum_recharge"].(float64); ok {
		minRechargeInt := int64(minRecharge)
		prize.MinimumRecharge = minRechargeInt
	}
	if iconName, ok := data["icon_name"].(string); ok {
		prize.IconName = iconName
	}
	if terms, ok := data["terms_and_conditions"].(string); ok {
		prize.TermsAndConditions = terms
	}
	if prizeCode, ok := data["prize_code"].(string); ok {
		prize.PrizeCode = prizeCode
	}
	if variationCode, ok := data["variation_code"].(string); ok {
		prize.VariationCode = variationCode
	}

	if err := s.db.WithContext(ctx).Table("prize_pool").Create(&prize).Error; err != nil {
		return nil, fmt.Errorf("create prize: %w", err)
	}
	return &prize, nil
}

// UpdatePrize updates an existing prize slot (admin).
func (s *SpinService) UpdatePrize(ctx context.Context, prizeID uuid.UUID, data map[string]interface{}) (*entities.PrizePoolEntry, error) {
	prize, err := s.GetPrize(ctx, prizeID)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{}
	if v, ok := data["name"].(string); ok && v != "" {
		updates["name"] = v
		prize.Name = v
	}
	if v, ok := data["prize_type"].(string); ok && v != "" {
		updates["prize_type"] = v
	}
	if v, ok := data["base_value"].(float64); ok {
		updates["base_value"] = v
		prize.BaseValue = v
	}
	if v, ok := data["win_probability_weight"].(float64); ok {
		newWeight := v
		// Validate weight cap (exclude current prize from count)
		if s.db != nil {
			var otherTotal float64
			s.db.WithContext(ctx).Table("prize_pool").
				Where("is_active = true AND id != ?", prizeID).
				Select("COALESCE(SUM(win_probability_weight), 0)").
				Scan(&otherTotal)
			if otherTotal+newWeight > 100.00 {
				return nil, fmt.Errorf("updating to %.2f%% would exceed 100%% (others: %.2f%%/100%%)", newWeight, otherTotal)
			}
		}
		updates["win_probability_weight"] = newWeight
		prize.ProbWeight = newWeight
	}
	if v, ok := data["is_active"].(bool); ok {
		updates["is_active"] = v
		prize.IsActive = v
	}
	if v, ok := data["daily_inventory_cap"].(float64); ok {
		capInt := int(v)
		updates["daily_inventory_cap"] = capInt
		prize.DailyInventoryCap = &capInt
	}
	if v, ok := data["is_no_win"].(bool); ok {
		updates["is_no_win"] = v
		prize.IsNoWin = v
	}
	if v, ok := data["no_win_message"].(string); ok {
		updates["no_win_message"] = v
		prize.NoWinMessage = v
	}
	if v, ok := data["color_scheme"].(string); ok {
		updates["color_scheme"] = v
		prize.ColorScheme = v
	}
	if v, ok := data["sort_order"].(float64); ok {
		sortInt := int(v)
		updates["sort_order"] = sortInt
		prize.SortOrder = sortInt
	}
	if v, ok := data["minimum_recharge"].(float64); ok {
		updates["minimum_recharge"] = int64(v)
		prize.MinimumRecharge = int64(v)
	}
	if v, ok := data["icon_name"].(string); ok {
		updates["icon_name"] = v
		prize.IconName = v
	}
	if v, ok := data["terms_and_conditions"].(string); ok {
		updates["terms_and_conditions"] = v
		prize.TermsAndConditions = v
	}
	if v, ok := data["prize_code"].(string); ok {
		updates["prize_code"] = v
		prize.PrizeCode = v
	}
	if v, ok := data["variation_code"].(string); ok {
		updates["variation_code"] = v
		prize.VariationCode = v
	}

	if len(updates) == 0 {
		return prize, nil
	}
	if err := s.db.WithContext(ctx).Table("prize_pool").Where("id = ?", prizeID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("update prize: %w", err)
	}
	return prize, nil
}

// DeletePrize soft-deletes a prize (sets is_active = false).
func (s *SpinService) DeletePrize(ctx context.Context, prizeID uuid.UUID) error {
	return s.db.WithContext(ctx).
		Table("prize_pool").
		Where("id = ?", prizeID).
		Update("is_active", false).Error
}

// ─── Eligibility ─────────────────────────────────────────────────────────

// SpinEligibility communicates whether a user can spin and why/why not.
// It also carries tier progress data so the frontend DailySpinProgress
// component can render the current tier, today's recharge total, and
// a progress bar toward the next tier — all from a single endpoint.
type SpinEligibility struct {
	Eligible       bool   `json:"eligible"`
	AvailableSpins int    `json:"available_spins"`
	SpinsUsedToday int    `json:"spins_used_today"`
	MaxSpinsToday  int    `json:"max_spins_today"`
	SpinCredits    int    `json:"spin_credits"`
	Message        string `json:"message"`
	// Tier progress fields — used by DailySpinProgress component
	CurrentTierName  string  `json:"current_tier_name"`
	TodayAmountNaira float64 `json:"today_amount_naira"`
	ProgressPercent  float64 `json:"progress_percent"`
	// Nudge: shown when ineligible due to no credits or daily cap reached
	TriggerNaira      int64  `json:"trigger_naira,omitempty"`
	NextTierName      string `json:"next_tier_name,omitempty"`
	NextTierMinAmount int64  `json:"next_tier_min_amount,omitempty"`
	AmountToNextTier  int64  `json:"amount_to_next_tier,omitempty"`
	NextTierSpins     int    `json:"next_tier_spins,omitempty"`
}

// CheckEligibility checks whether a user is eligible to spin.
// It also returns tier progress data (current tier name, today's recharge total,
// progress percent) so the frontend DailySpinProgress component can render
// without a separate API call.
func (s *SpinService) CheckEligibility(ctx context.Context, userID uuid.UUID) (*SpinEligibility, error) {
	wallet, err := s.userRepo.GetWallet(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("wallet not found: %w", err)
	}

	// 1. Calculate today's cumulative recharge amount
	todayMidnight := time.Now().UTC().Truncate(24 * time.Hour).Unix()
	todayAmountKobo, err := s.txRepo.SumAmountByUserSince(ctx, userID, todayMidnight)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate daily recharge: %w", err)
	}
	todayAmountNaira := float64(todayAmountKobo) / 100.0

	// 2. Determine daily spin cap and current tier from cumulative amount
	tierCalc := utils.NewSpinTierCalculatorDB(s.db)
	dailyCap := 0
	currentTierName := ""
	progressPercent := 0.0

	allTiers, _ := tierCalc.GetAllTiersFromDB()
	currentTierIdx := -1
	if currentTier, err := tierCalc.GetSpinTierFromDB(todayAmountKobo); err == nil && currentTier.SpinsPerDay > 0 {
		dailyCap = currentTier.SpinsPerDay
		currentTierName = currentTier.TierDisplayName
		// Find index of current tier in allTiers for progress calculation
		for i, t := range allTiers {
			if t.TierDisplayName == currentTier.TierDisplayName {
				currentTierIdx = i
				break
			}
		}
	}

	// Calculate progress percent toward next tier
	if currentTierIdx >= 0 && currentTierIdx+1 < len(allTiers) {
		nxt := allTiers[currentTierIdx+1]
		cur := allTiers[currentTierIdx]
		tierRange := float64(nxt.MinDailyAmount - cur.MinDailyAmount)
		if tierRange > 0 {
			progressPercent = float64(todayAmountKobo-cur.MinDailyAmount) / tierRange * 100
			if progressPercent > 100 {
				progressPercent = 100
			}
		}
	} else if currentTierIdx < 0 && len(allTiers) > 0 {
		// Below minimum tier — progress toward Bronze
		nxt := allTiers[0]
		if nxt.MinDailyAmount > 0 {
			progressPercent = float64(todayAmountKobo) / float64(nxt.MinDailyAmount) * 100
			if progressPercent > 100 {
				progressPercent = 100
			}
		}
	}

	// 3. Count all spins played today
	used, err := s.prizeRepo.CountUserSpinsToday(ctx, userID)
	if err != nil {
		return nil, err
	}

	// If user has no spin credits, they can't spin regardless of tier
	if wallet.SpinCredits < 1 {
		trigger := s.cfg.GetInt64("spin_trigger_naira", 1000)
		resp := &SpinEligibility{
			Eligible:         false,
			SpinCredits:      wallet.SpinCredits,
			Message:          fmt.Sprintf("No spin credits. Recharge ₦%d to earn one!", trigger),
			TriggerNaira:     trigger,
			CurrentTierName:  currentTierName,
			TodayAmountNaira: todayAmountNaira,
			ProgressPercent:  progressPercent,
		}
		// Nudge toward next tier
		for _, t := range allTiers {
			if t.MinDailyAmount > todayAmountKobo {
				resp.NextTierName = t.TierDisplayName
				resp.NextTierMinAmount = t.MinDailyAmount
				resp.AmountToNextTier = t.MinDailyAmount - todayAmountKobo
				resp.NextTierSpins = t.SpinsPerDay
				break
			}
		}
		return resp, nil
	}

	// If user has reached their daily cap based on their tier
	if used >= dailyCap {
		resp := &SpinEligibility{
			Eligible:         false,
			SpinsUsedToday:   used,
			MaxSpinsToday:    dailyCap,
			SpinCredits:      wallet.SpinCredits,
			Message:          fmt.Sprintf("Daily spin limit reached (%d/%d). Recharge more today to unlock additional spins!", used, dailyCap),
			CurrentTierName:  currentTierName,
			TodayAmountNaira: todayAmountNaira,
			ProgressPercent:  progressPercent,
		}

		// Build upgrade nudge
		for _, t := range allTiers {
			if t.MinDailyAmount > todayAmountKobo {
				resp.NextTierName = t.TierDisplayName
				resp.NextTierMinAmount = t.MinDailyAmount
				resp.AmountToNextTier = t.MinDailyAmount - todayAmountKobo
				resp.NextTierSpins = t.SpinsPerDay
				break
			}
		}
		return resp, nil
	}

	available := dailyCap - used
	// User cannot spin more times than they have credits
	if available > wallet.SpinCredits {
		available = wallet.SpinCredits
	}

	return &SpinEligibility{
		Eligible:         true,
		AvailableSpins:   available,
		SpinsUsedToday:   used,
		MaxSpinsToday:    dailyCap,
		SpinCredits:      wallet.SpinCredits,
		Message:          fmt.Sprintf("You have %d spin(s) available today!", available),
		CurrentTierName:  currentTierName,
		TodayAmountNaira: todayAmountNaira,
		ProgressPercent:  progressPercent,
	}, nil
}

// ─── MoMo Hold Flow (Spec §8.2 — new, not in RechargeMax) ───────────────

// ConfirmMoMoPrize is called once the user links their MoMo number after winning.
// Transitions spin_result from pending_momo_setup → processing → dispatches fulfillment.
func (s *SpinService) ConfirmMoMoPrize(ctx context.Context, userID uuid.UUID, spinResultID uuid.UUID) error {
	// Fetch spin result
	spinResult, err := s.prizeRepo.FindSpinResult(ctx, spinResultID)
	if err != nil {
		return fmt.Errorf("spin result not found: %w", err)
	}
	if spinResult.UserID != userID {
		return fmt.Errorf("spin result does not belong to this user")
	}
	if spinResult.FulfillmentStatus != entities.FulfillPendingMoMo {
		return fmt.Errorf("spin result is not awaiting MoMo setup (status: %s)", spinResult.FulfillmentStatus)
	}

	// Verify user now has MoMo linked
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	if !user.MoMoVerified || user.MoMoNumber == "" {
		return fmt.Errorf("please verify your MoMo number first")
	}

	// Transition to processing
	if err := s.prizeRepo.UpdateSpinFulfillment(ctx, spinResultID, entities.FulfillProcessing, "", ""); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Dispatch fulfillment
	safe.Go(func() {
		if err := s.fulfillSvc.Fulfill(context.Background(), spinResult); err != nil {
			log.Printf("[SPIN] MoMo fulfillment failed for %s: %v", spinResultID, err)
		}
	})
	return nil
}

// GetSpinHistory returns a user's spin history.
func (s *SpinService) GetSpinHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entities.SpinResult, error) {
	var results []entities.SpinResult
	err := s.db.WithContext(ctx).
		Table("spin_results").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&results).Error
	return results, err
}

// GetStats returns platform-wide spin statistics.
func (s *SpinService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	var totalSpins, spinsToday int64
	var pendingFulfillments int64
	s.db.Table("spin_results").Count(&totalSpins)
	s.db.Table("spin_results").Where("created_at >= ?", time.Now().Truncate(24*time.Hour)).Count(&spinsToday)
	s.db.Table("spin_results").
		Where("fulfillment_status IN ('pending','processing','pending_momo_setup')").
		Count(&pendingFulfillments)
	return map[string]interface{}{
		"total_spins":          totalSpins,
		"spins_today":          spinsToday,
		"pending_fulfillments": pendingFulfillments,
	}, nil
}

// RollbackSpin cancels a spin that was initiated via USSD but never completed
// because the USSD session timed out (REQ-6.5).
// It marks the spin result as "failed" and restores the user's spin credit.
func (s *SpinService) RollbackSpin(ctx context.Context, spinResultID uuid.UUID) error {
return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
var result entities.SpinResult
if err := tx.Table("spin_results").
Where("id = ? AND fulfillment_status = ?", spinResultID, entities.FulfillPending).
First(&result).Error; err != nil {
// Already fulfilled or does not exist — nothing to roll back.
return nil
}

// Mark the spin result as failed with a clear reason.
if err := tx.Table("spin_results").
Where("id = ?", spinResultID).
Updates(map[string]interface{}{
"fulfillment_status": entities.FulfillFailed,
"error_message":      "USSD session timed out — spin rolled back",
}).Error; err != nil {
return fmt.Errorf("rollback spin result: %w", err)
}

// Restore the spin credit to the user's wallet.
if err := tx.Table("wallets").
Where("user_id = ?", result.UserID).
UpdateColumn("spin_credits", gorm.Expr("spin_credits + 1")).Error; err != nil {
return fmt.Errorf("restore spin credit: %w", err)
}

return nil
})
}

// ─── Spin Tiers (Admin) ──────────────────────────────────────────────────

func (s *SpinService) GetAllSpinTiers(ctx context.Context) ([]entities.SpinTier, error) {
	var tiers []entities.SpinTier
	err := s.db.WithContext(ctx).Order("sort_order ASC").Find(&tiers).Error
	return tiers, err
}

func (s *SpinService) CreateSpinTier(ctx context.Context, data map[string]interface{}) (*entities.SpinTier, error) {
	tier := entities.SpinTier{
		ID:       uuid.New(),
		IsActive: true,
	}

	if v, ok := data["tier_name"].(string); ok { tier.TierName = v }
	if v, ok := data["tier_display_name"].(string); ok { tier.TierDisplayName = v }
	if v, ok := data["min_daily_amount"].(float64); ok { tier.MinDailyAmount = int64(v) }
	if v, ok := data["max_daily_amount"].(float64); ok { tier.MaxDailyAmount = int64(v) }
	if v, ok := data["spins_per_day"].(float64); ok { tier.SpinsPerDay = int(v) }
	if v, ok := data["tier_color"].(string); ok { tier.TierColor = v }
	if v, ok := data["tier_icon"].(string); ok { tier.TierIcon = v }
	if v, ok := data["tier_badge"].(string); ok { tier.TierBadge = v }
	if v, ok := data["description"].(string); ok { tier.Description = v }
	if v, ok := data["sort_order"].(float64); ok { tier.SortOrder = int(v) }
	if v, ok := data["is_active"].(bool); ok { tier.IsActive = v }

	if err := s.db.WithContext(ctx).Create(&tier).Error; err != nil {
		return nil, err
	}
	return &tier, nil
}

func (s *SpinService) UpdateSpinTier(ctx context.Context, id uuid.UUID, data map[string]interface{}) (*entities.SpinTier, error) {
	var tier entities.SpinTier
	if err := s.db.WithContext(ctx).First(&tier, "id = ?", id).Error; err != nil {
		return nil, err
	}

	updates := map[string]interface{}{}
	if v, ok := data["tier_name"].(string); ok { updates["tier_name"] = v; tier.TierName = v }
	if v, ok := data["tier_display_name"].(string); ok { updates["tier_display_name"] = v; tier.TierDisplayName = v }
	if v, ok := data["min_daily_amount"].(float64); ok { updates["min_daily_amount"] = int64(v); tier.MinDailyAmount = int64(v) }
	if v, ok := data["max_daily_amount"].(float64); ok { updates["max_daily_amount"] = int64(v); tier.MaxDailyAmount = int64(v) }
	if v, ok := data["spins_per_day"].(float64); ok { updates["spins_per_day"] = int(v); tier.SpinsPerDay = int(v) }
	if v, ok := data["tier_color"].(string); ok { updates["tier_color"] = v; tier.TierColor = v }
	if v, ok := data["tier_icon"].(string); ok { updates["tier_icon"] = v; tier.TierIcon = v }
	if v, ok := data["tier_badge"].(string); ok { updates["tier_badge"] = v; tier.TierBadge = v }
	if v, ok := data["description"].(string); ok { updates["description"] = v; tier.Description = v }
	if v, ok := data["sort_order"].(float64); ok { updates["sort_order"] = int(v); tier.SortOrder = int(v) }
	if v, ok := data["is_active"].(bool); ok { updates["is_active"] = v; tier.IsActive = v }

	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(&tier).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	return &tier, nil
}

func (s *SpinService) DeleteSpinTier(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Model(&entities.SpinTier{}).Where("id = ?", id).Update("is_active", false).Error
}
