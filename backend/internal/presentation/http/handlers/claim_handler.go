package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/presentation/http/middleware"
)

type ClaimHandler struct {
	claimSvc *services.ClaimService
}

func NewClaimHandler(claimSvc *services.ClaimService) *ClaimHandler {
	return &ClaimHandler{claimSvc: claimSvc}
}

func (h *ClaimHandler) GetMyWins(w http.ResponseWriter, r *http.Request) {
	uidStr, ok := r.Context().Value(middleware.ContextUserID).(string)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID, err := uuid.Parse(uidStr)
	if err != nil {
		jsonError(w, "invalid user id", http.StatusUnauthorized)
		return
	}

	wins, err := h.claimSvc.GetMyWins(r.Context(), userID)
	if err != nil {
		jsonError(w, "failed to get wins: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Enrich each result with a human-readable prize_label so the frontend
	// can display the prize name without computing it from type + value.
	type enrichedWin struct {
		ID                interface{} `json:"id"`
		PrizeType         interface{} `json:"prize_type"`
		PrizeValue        interface{} `json:"prize_value"`
		PrizeLabel        string      `json:"prize_label"`
		FulfillmentStatus interface{} `json:"fulfillment_status"`
		FulfillmentRef    interface{} `json:"fulfillment_ref,omitempty"`
		ClaimStatus       interface{} `json:"claim_status"`
		CreatedAt         interface{} `json:"created_at"`
		ExpiresAt         interface{} `json:"expires_at"`
		MoMoClaimNumber   interface{} `json:"momo_claim_number,omitempty"`
	}
	out := make([]enrichedWin, 0, len(wins))
	for _, w := range wins {
		label := prizeLabelFor(string(w.PrizeType), w.PrizeValue)
		out = append(out, enrichedWin{
			ID:                w.ID,
			PrizeType:         w.PrizeType,
			PrizeValue:        w.PrizeValue,
			PrizeLabel:        label,
			FulfillmentStatus: w.FulfillmentStatus,
			FulfillmentRef:    w.FulfillmentRef,
			ClaimStatus:       w.ClaimStatus,
			CreatedAt:         w.CreatedAt,
			ExpiresAt:         w.ExpiresAt,
			MoMoClaimNumber:   w.MoMoClaimNumber,
		})
	}
	jsonOK(w, out)
}

// prizeLabelFor returns a human-readable label for a prize.
// value is in KOBO for monetary prizes; pulse_points are whole units.
func prizeLabelFor(prizeType string, valueKobo float64) string {
	naira := valueKobo / 100.0
	switch prizeType {
	case "airtime":
		return fmt.Sprintf("₦%.0f Airtime", naira)
	case "data_bundle":
		// For data we show the Naira value since MB info is in the prize name.
		return fmt.Sprintf("₦%.0f Data Bundle", naira)
	case "pulse_points":
		return fmt.Sprintf("+%.0f Pulse Points", valueKobo) // points are not kobo
	case "momo_cash":
		return fmt.Sprintf("₦%.0f MoMo Cash", naira)
	default:
		return "Prize"
	}
}

func (h *ClaimHandler) ClaimPrize(w http.ResponseWriter, r *http.Request) {
	uidStr, ok := r.Context().Value(middleware.ContextUserID).(string)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID, err := uuid.Parse(uidStr)
	if err != nil {
		jsonError(w, "invalid user id", http.StatusUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	claimID, err := uuid.Parse(idStr)
	if err != nil {
		jsonError(w, "invalid claim id", http.StatusBadRequest)
		return
	}

	var req services.ClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.claimSvc.ClaimPrize(r.Context(), userID, claimID, req)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonOK(w, result)
}

func (h *ClaimHandler) CheckMoMoAccount(w http.ResponseWriter, r *http.Request) {
	phone := r.URL.Query().Get("phone")
	if phone == "" {
		jsonError(w, "phone parameter is required", http.StatusBadRequest)
		return
	}

	hasAccount, name, err := h.claimSvc.CheckMoMoAccount(r.Context(), phone)
	if err != nil {
		jsonError(w, "failed to check account: "+err.Error(), http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]interface{}{
		"has_momo_account": hasAccount,
		"account_name":     name,
	})
}
