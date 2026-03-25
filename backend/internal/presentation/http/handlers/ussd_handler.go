package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/config"
)

// USSDHandler processes USSD requests from Africa's Talking gateway.
// Format: text is a *-separated string of accumulated menu choices.
// e.g., "1" = first input, "1*2" = first menu chose 1, second menu chose 2
type USSDHandler struct {
	spinSvc    *services.SpinService
	rechargeSvc *services.RechargeService
	userRepo   repositories.UserRepository
	cfg        *config.ConfigManager
}

func NewUSSDHandler(ss *services.SpinService, rs *services.RechargeService, ur repositories.UserRepository, cfg *config.ConfigManager) *USSDHandler {
	return &USSDHandler{spinSvc: ss, rechargeSvc: rs, userRepo: ur, cfg: cfg}
}

func (h *USSDHandler) Handle(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	sessionID := r.FormValue("sessionId")
	phone     := r.FormValue("phoneNumber")
	text      := r.FormValue("text")
	_ = sessionID

	response := h.processMenu(r, phone, text)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, response)
}

func (h *USSDHandler) processMenu(r *http.Request, phone, text string) string {
	parts := strings.Split(text, "*")
	level := len(parts)
	if text == "" {
		level = 0
	}

	// Root menu
	if level == 0 {
		return fmt.Sprintf("CON Welcome to Loyalty Nexus!\n1. Check Balance\n2. Spin & Win\n3. My Points\n4. Nexus Studio\n5. Exit")
	}

	switch parts[0] {
	case "1": // Check Balance
		user, err := h.userRepo.FindByPhoneNumber(r.Context(), phone)
		if err != nil {
			return "END Account not found. Dial *789# to register."
		}
		wallet, err := h.userRepo.GetWallet(r.Context(), user.ID)
		if err != nil {
			return "END Unable to fetch balance."
		}
		return fmt.Sprintf("END Your Loyalty Nexus Balance:\nPulse Points: %d\nSpin Credits: %d\nStreak: Day %d",
			wallet.PulsePoints, wallet.SpinCredits, user.StreakCount)

	case "2": // Spin & Win
		if level == 1 {
			return "CON Spin & Win:\n1. Play a Spin\n2. My Spin History\n0. Back"
		}
		if parts[1] == "1" {
			user, err := h.userRepo.FindByPhoneNumber(r.Context(), phone)
			if err != nil {
				return "END Account not found."
			}
			outcome, err := h.spinSvc.PlaySpin(r.Context(), user.ID)
			if err != nil {
				return "END " + err.Error()
			}
			return "END " + outcome.Message
		}
		return "END Coming soon."

	case "3": // My Points
		user, err := h.userRepo.FindByPhoneNumber(r.Context(), phone)
		if err != nil {
			return "END Account not found."
		}
		wallet, _ := h.userRepo.GetWallet(r.Context(), user.ID)
		return fmt.Sprintf("END Pulse Points: %d\nTier: %s\nRecharge ₦%d to earn more.",
			wallet.PulsePoints, user.Tier,
			h.cfg.GetInt64("spin_trigger_naira", 1000))

	case "4": // Nexus Studio menu (lightweight)
		return "END Open the Loyalty Nexus app to access your full AI Studio experience."

	case "5":
		return "END Thank you for using Loyalty Nexus. Keep recharging to earn more rewards!"

	default:
		return "END Invalid option. Dial again."
	}
}
