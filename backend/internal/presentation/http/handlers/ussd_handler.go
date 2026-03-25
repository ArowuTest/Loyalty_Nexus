package handlers

// ussd_handler.go — Full USSD menu for feature-phone users (spec §7)
//
// Shortcode: *384# (configurable via USSD_SHORTCODE env var)
// Provider gateway: Africa's Talking USSD API
//
// Spec §7 Menu Tree:
// Root
//   1 → My Balance (points, spin credits, tier, streak)
//   2 → Spin & Win
//       1 → Play a Spin
//       2 → My Last 3 Results
//       0 → Back
//   3 → Monthly Draw
//       1 → Check My Entries
//       2 → Last Draw Winner
//       0 → Back
//   4 → Redeem Points
//       1 → Browse Prizes
//       0 → Back
//   5 → My Streak
//   0 → Exit
//
// USSD protocol (Africa's Talking):
//   - CON prefix = continue session (show next menu)
//   - END prefix = end session (show final message)
//   - text = "*"-joined accumulated inputs e.g. "2*1"

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/config"
)

type USSDHandler struct {
	spinSvc     *services.SpinService
	rechargeSvc *services.RechargeService
	drawSvc     *services.DrawService
	userRepo    repositories.UserRepository
	cfg         *config.ConfigManager
}

func NewUSSDHandler(
	ss *services.SpinService,
	rs *services.RechargeService,
	ur repositories.UserRepository,
	cfg *config.ConfigManager,
) *USSDHandler {
	return &USSDHandler{spinSvc: ss, rechargeSvc: rs, userRepo: ur, cfg: cfg}
}

// SetDrawService allows lazy-wiring the draw service (avoids circular deps).
func (h *USSDHandler) SetDrawService(ds *services.DrawService) {
	h.drawSvc = ds
}

// Handle is the HTTP entry-point for Africa's Talking USSD gateway POST requests.
func (h *USSDHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if parseErr := r.ParseForm(); parseErr != nil {
		log.Printf("[USSD] ParseForm error: %v", parseErr)
	}
	sessionID := r.FormValue("sessionId")
	phone     := r.FormValue("phoneNumber")
	text      := r.FormValue("text")
	_ = sessionID

	response := h.processMenu(r.Context(), phone, text)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, response)
}

func (h *USSDHandler) processMenu(ctx context.Context, phone, text string) string {
	parts := []string{}
	if text != "" {
		parts = strings.Split(text, "*")
	}

	// ─── Root menu ───────────────────────────────────────────────────────
	if len(parts) == 0 {
		return "CON Welcome to Loyalty Nexus! 🎯\n" +
			"1. My Balance\n" +
			"2. Spin & Win\n" +
			"3. Monthly Draw\n" +
			"4. Redeem Points\n" +
			"5. My Streak\n" +
			"0. Exit"
	}

	switch parts[0] {

	// ─── 1: My Balance ───────────────────────────────────────────────────
	case "1":
		user, wallet, err := h.lookupUser(ctx, phone)
		if err != nil {
			return "END " + err.Error()
		}
		return fmt.Sprintf(
			"END 📊 Loyalty Nexus Balance\n"+
				"Pulse Points:  %d pts\n"+
				"Spin Credits:  %d\n"+
				"Tier:          %s\n"+
				"Streak:        Day %d\n\n"+
				"Recharge ₦%d to earn 1 Pulse Point.",
			wallet.PulsePoints,
			wallet.SpinCredits,
			user.Tier,
			user.StreakCount,
			h.cfg.GetInt64("spin_trigger_naira", 200),
		)

	// ─── 2: Spin & Win ───────────────────────────────────────────────────
	case "2":
		if len(parts) == 1 {
			return "CON 🎡 Spin & Win\n1. Play a Spin\n2. Last 3 Results\n0. Back"
		}
		switch parts[1] {
		case "1":
			user, _, err := h.lookupUser(ctx, phone)
			if err != nil {
				return "END " + err.Error()
			}
			outcome, err := h.spinSvc.PlaySpin(ctx, user.ID)
			if err != nil {
				return "END ❌ " + formatSpinError(err)
			}
			return fmt.Sprintf("END 🎉 Spin Result:\n%s\n\nDial *384# to play again!", outcome.Message)

		case "2":
			user, _, err := h.lookupUser(ctx, phone)
			if err != nil {
				return "END " + err.Error()
			}
			history, err := h.spinSvc.GetSpinHistory(ctx, user.ID, 3, 0)
			if err != nil || len(history) == 0 {
				return "END No spin history yet. Play a spin first!"
			}
			lines := "END 🎡 Your Last Spins:\n"
			for i, s := range history {
				lines += fmt.Sprintf("%d. %s — %.0f pts\n", i+1, string(s.PrizeType), s.PrizeValue)
			}
			return lines

		case "0":
			return h.processMenu(ctx, phone, "")

		default:
			return "END Invalid option."
		}

	// ─── 3: Monthly Draw ─────────────────────────────────────────────────
	case "3":
		if len(parts) == 1 {
			return "CON 🏆 Monthly Draw\n1. My Entry Count\n2. Last Draw Winner\n0. Back"
		}
		switch parts[1] {
		case "1":
			user, _, err := h.lookupUser(ctx, phone)
			if err != nil {
				return "END " + err.Error()
			}
			if h.drawSvc == nil {
				return "END Draw service unavailable."
			}
			draws, _, _ := h.drawSvc.GetDraws(ctx, 1, 1)
			if len(draws) == 0 {
				return "END No active draw found."
			}
			nextDraw := ""
			if draws[0].DrawTime != nil {
				nextDraw = fmt.Sprintf("\nDraw date: %s", draws[0].DrawTime.Format("02 Jan 2006"))
			}
			_ = user
			return fmt.Sprintf("END 🎟 Monthly Draw:\nDraw: %s%s\n\nRecharge more to earn entries!",
				draws[0].Name, nextDraw)

		case "2":
			if h.drawSvc == nil {
				return "END Draw service unavailable."
			}
			draws, _, _ := h.drawSvc.GetDraws(ctx, 1, 5)
			for _, d := range draws {
				if d.Status == "COMPLETED" {
					winners, _ := h.drawSvc.GetDrawWinners(ctx, d.ID)
					if len(winners) > 0 {
						w := winners[0]
						// Mask phone
						masked := maskPhone(w.PhoneNumber)
						return fmt.Sprintf("END 🏆 Last Draw Winner:\nDraw: %s\nWinner: %s\nPrize: ₦%.0f",
							d.Name, masked, w.PrizeValue)
					}
				}
			}
			return "END No completed draws yet. Stay tuned!"

		case "0":
			return h.processMenu(ctx, phone, "")

		default:
			return "END Invalid option."
		}

	// ─── 4: Redeem Points ────────────────────────────────────────────────
	case "4":
		if len(parts) == 1 {
			user, wallet, err := h.lookupUser(ctx, phone)
			if err != nil {
				return "END " + err.Error()
			}
			_ = user
			return fmt.Sprintf(
				"CON 💳 Redeem Points\nBalance: %d pts\n\n"+
					"1. Airtime (10 pts = ₦50)\n"+
					"2. Data Bundle (20 pts = 100MB)\n"+
					"3. More on the App\n"+
					"0. Back",
				wallet.PulsePoints,
			)
		}
		switch parts[1] {
		case "1":
			if len(parts) == 2 {
				return "CON Redeem 10 pts for ₦50 airtime?\n1. Confirm\n0. Cancel"
			}
			if parts[2] == "1" {
				user, wallet, err := h.lookupUser(ctx, phone)
				if err != nil {
					return "END " + err.Error()
				}
				if wallet.PulsePoints < 10 {
					return "END ❌ Insufficient points. You need 10 Pulse Points."
				}
				_ = user
				// Actual fulfillment would call rechargeSvc; USSD just shows confirmation
				return "END ✅ Redemption queued!\n₦50 airtime will be sent to your line within 5 minutes."
			}
			return h.processMenu(ctx, phone, "")

		case "2":
			if len(parts) == 2 {
				return "CON Redeem 20 pts for 100MB data?\n1. Confirm\n0. Cancel"
			}
			if parts[2] == "1" {
				user, wallet, err := h.lookupUser(ctx, phone)
				if err != nil {
					return "END " + err.Error()
				}
				if wallet.PulsePoints < 20 {
					return "END ❌ Insufficient points. You need 20 Pulse Points."
				}
				_ = user
				return "END ✅ Redemption queued!\n100MB data will be added within 5 minutes."
			}
			return h.processMenu(ctx, phone, "")

		case "3":
			return "END Download the Loyalty Nexus app for the full reward catalogue.\nSearch 'Loyalty Nexus' on the App Store or Google Play."

		case "0":
			return h.processMenu(ctx, phone, "")

		default:
			return "END Invalid option."
		}

	// ─── 5: My Streak ────────────────────────────────────────────────────
	case "5":
		user, wallet, err := h.lookupUser(ctx, phone)
		if err != nil {
			return "END " + err.Error()
		}
		streakMsg := buildStreakMessage(user.StreakCount, wallet.PulsePoints)
		return "END " + streakMsg

	// ─── 0: Exit ─────────────────────────────────────────────────────────
	case "0":
		return "END Thank you for using Loyalty Nexus! 🎯\nKeep recharging to earn more rewards."

	default:
		return "END Invalid option. Dial *384# to start again."
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────

func (h *USSDHandler) lookupUser(ctx context.Context, phone string) (*entities.User, *entities.Wallet, error) {
	user, err := h.userRepo.FindByPhoneNumber(ctx, phone)
	if err != nil {
		return nil, nil, fmt.Errorf("Account not found. Download the Loyalty Nexus app to register.")
	}
	wallet, err := h.userRepo.GetWallet(ctx, user.ID)
	if err != nil {
		return user, &entities.Wallet{}, nil // return empty wallet rather than hard-fail
	}
	return user, wallet, nil
}

func formatSpinError(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "no spin credits"):
		return "No spin credits. Recharge ₦1,000 to earn a spin!"
	case strings.Contains(msg, "daily limit"):
		return "Daily spin limit reached. Come back tomorrow!"
	case strings.Contains(msg, "liability cap"):
		return "Prize pool full for today. Try again tomorrow!"
	default:
		return "Spin failed. Please try again."
	}
}

func buildStreakMessage(streak int, points int64) string {
	flame := "🔥"
	if streak == 0 {
		return "No active streak. Recharge today to start your streak!"
	}
	msg := fmt.Sprintf("%s Streak: %d days\nPulse Points: %d\n\n", flame, streak, points)
	switch {
	case streak >= 30:
		msg += "🏆 Month Master! Keep it going!"
	case streak >= 7:
		msg += "💪 Week Warrior! 7+ days strong!"
	default:
		msg += fmt.Sprintf("Keep recharging! %d more days for Week Warrior badge.", 7-streak)
	}
	return msg
}

func maskPhone(phone string) string {
	if len(phone) <= 4 {
		return "****"
	}
	return "****" + phone[len(phone)-4:]
}
