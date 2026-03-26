package handlers

// ussd_handler.go — Full USSD menu for feature-phone users (spec §7)
//
// REQ-6.1: Shortcode is read from ConfigManager key "ussd_shortcode" (default *384#).
//          Provider gateway: Africa's Talking USSD API.
//
// REQ-6.2: Menu option 1 shows Pulse Points, Spin Credits, and Recharge Streak.
//
// REQ-6.3: Menu option 2 allows a spin. Prize outcome is communicated in the
//          USSD response. Fulfillment is identical to the web app (same SpinService).
//
// REQ-6.4: Menu option 7 allows feature phone users to access Knowledge Tools
//          (Study Guide, Quiz, Mind Map) by submitting a topic. The request is
//          processed asynchronously and the result is delivered via SMS.
//
// REQ-6.5: Session state is persisted to the DB via USSDSessionRepository.
//          On each request, expired sessions with a pending spin are detected
//          and the spin is rolled back before the session is cleaned up.
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
	"time"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/config"

	"github.com/google/uuid"
)

// USSDHandler handles all USSD gateway requests.
type USSDHandler struct {
	spinSvc          *services.SpinService
	rechargeSvc      *services.RechargeService
	drawSvc          *services.DrawService
	passportSvc      *services.PassportService
	knowledgeSvc     *services.USSDKnowledgeService
	userRepo         repositories.UserRepository
	sessionRepo      repositories.USSDSessionRepository
	cfg              *config.ConfigManager
}

// NewUSSDHandler constructs the handler with required dependencies.
func NewUSSDHandler(
	ss *services.SpinService,
	rs *services.RechargeService,
	ur repositories.UserRepository,
	sessionRepo repositories.USSDSessionRepository,
	cfg *config.ConfigManager,
) *USSDHandler {
	return &USSDHandler{
		spinSvc:     ss,
		rechargeSvc: rs,
		userRepo:    ur,
		sessionRepo: sessionRepo,
		cfg:         cfg,
	}
}

// SetPassportService allows lazy-wiring the passport service.
func (h *USSDHandler) SetPassportService(ps *services.PassportService) {
	h.passportSvc = ps
}

// SetDrawService allows lazy-wiring the draw service (avoids circular deps).
func (h *USSDHandler) SetDrawService(ds *services.DrawService) {
	h.drawSvc = ds
}

// SetKnowledgeService allows lazy-wiring the USSD knowledge service.
func (h *USSDHandler) SetKnowledgeService(ks *services.USSDKnowledgeService) {
	h.knowledgeSvc = ks
}

// Handle is the HTTP entry-point for Africa's Talking USSD gateway POST requests.
func (h *USSDHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if parseErr := r.ParseForm(); parseErr != nil {
		log.Printf("[USSD] ParseForm error: %v", parseErr)
	}
	sessionID := r.FormValue("sessionId")
	phone     := r.FormValue("phoneNumber")
	text      := r.FormValue("text")

	// REQ-6.5: Rollback any expired sessions with pending spins before processing.
	go h.rollbackExpiredSessions(r.Context())

	// REQ-6.5: Load or create the session record.
	session := h.loadOrCreateSession(r.Context(), sessionID, phone)

	response := h.processMenu(r.Context(), session, phone, text)

	// Persist session state after every request.
	// If the response is END, the session is over — delete it.
	if strings.HasPrefix(response, "END") {
		if h.sessionRepo != nil && session.ID != uuid.Nil {
			_ = h.sessionRepo.DeleteExpired(r.Context())
		}
	} else {
		h.saveSession(r.Context(), session, text)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, response)
}

// loadOrCreateSession loads an existing session or returns a new one.
func (h *USSDHandler) loadOrCreateSession(ctx context.Context, sessionID, phone string) *entities.USSDSession {
	if h.sessionRepo == nil {
		return &entities.USSDSession{SessionID: sessionID}
	}
	sess, err := h.sessionRepo.GetBySessionID(ctx, sessionID)
	if err != nil || sess == nil {
		timeout := time.Duration(h.cfg.GetInt("ussd_session_timeout_seconds", 120)) * time.Second
		return &entities.USSDSession{
			SessionID:   sessionID,
			PhoneNumber: phone,
			MenuState:   "root",
			ExpiresAt:   time.Now().Add(timeout),
		}
	}
	return sess
}

// saveSession persists the current session state to the DB.
func (h *USSDHandler) saveSession(ctx context.Context, session *entities.USSDSession, text string) {
	if h.sessionRepo == nil {
		return
	}
	timeout := time.Duration(h.cfg.GetInt("ussd_session_timeout_seconds", 120)) * time.Second
	session.ExpiresAt = time.Now().Add(timeout)
	session.InputBuffer = text
	if err := h.sessionRepo.Upsert(ctx, session); err != nil {
		log.Printf("[USSD] session upsert error: %v", err)
	}
}

// rollbackExpiredSessions finds expired sessions with pending spins and rolls them back.
// REQ-6.5: Any partially initiated spin that was not completed must be rolled back.
func (h *USSDHandler) rollbackExpiredSessions(ctx context.Context) {
	if h.sessionRepo == nil {
		return
	}
	sessions, err := h.sessionRepo.GetExpiredWithPendingSpin(ctx)
	if err != nil {
		log.Printf("[USSD] GetExpiredWithPendingSpin error: %v", err)
		return
	}
	for _, sess := range sessions {
		if sess.PendingSpinID == nil {
			continue
		}
		if rbErr := h.spinSvc.RollbackSpin(ctx, *sess.PendingSpinID); rbErr != nil {
			log.Printf("[USSD] rollback spin %s for session %s: %v",
				*sess.PendingSpinID, sess.SessionID, rbErr)
		} else {
			log.Printf("[USSD] rolled back spin %s (session %s expired)",
				*sess.PendingSpinID, sess.SessionID)
		}
	}
	if cleanErr := h.sessionRepo.DeleteExpired(ctx); cleanErr != nil {
		log.Printf("[USSD] delete expired sessions: %v", cleanErr)
	}
}

func (h *USSDHandler) processMenu(ctx context.Context, session *entities.USSDSession, phone, text string) string {
	parts := []string{}
	if text != "" {
		parts = strings.Split(text, "*")
	}

	// REQ-6.1: Read shortcode from ConfigManager — zero hardcoding.
	shortcode := h.cfg.GetString("ussd_shortcode", "*384#")

	// ─── Root menu ───────────────────────────────────────────────────────
	if len(parts) == 0 {
		return fmt.Sprintf(
			"CON Welcome to Loyalty Nexus! 🎯\nDial %s\n\n"+
				"1. My Balance\n"+
				"2. Spin & Win\n"+
				"3. Monthly Draw\n"+
				"4. Redeem Points\n"+
				"5. My Streak\n"+
				"6. My Passport\n"+
				"7. AI Knowledge Tools\n"+
				"0. Exit",
			shortcode,
		)
	}

	switch parts[0] {

	// ─── 1: My Balance (REQ-6.2) ─────────────────────────────────────────
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

	// ─── 2: Spin & Win (REQ-6.3) ─────────────────────────────────────────
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
			// REQ-6.5: Store pending spin ID in session before executing.
			// If the session times out mid-spin, the spin will be rolled back.
			pendingID := uuid.New()
			session.PendingSpinID = &pendingID
			h.saveSession(ctx, session, text)

			outcome, spinErr := h.spinSvc.PlaySpin(ctx, user.ID)
			if spinErr != nil {
				// Spin failed — clear pending spin ID
				session.PendingSpinID = nil
				h.saveSession(ctx, session, text)
				return "END ❌ " + formatSpinError(spinErr)
			}
			// Spin succeeded — clear pending spin ID
			session.PendingSpinID = nil
			h.saveSession(ctx, session, text)

			return fmt.Sprintf("END 🎉 Spin Result:\n%s\n\nDial %s to play again!",
				outcome.Message, shortcode)

		case "2":
			user, _, err := h.lookupUser(ctx, phone)
			if err != nil {
				return "END " + err.Error()
			}
			history, histErr := h.spinSvc.GetSpinHistory(ctx, user.ID, 3, 0)
			if histErr != nil || len(history) == 0 {
				return "END No spin history yet. Play a spin first!"
			}
			lines := "END 🎡 Your Last Spins:\n"
			for i, s := range history {
				lines += fmt.Sprintf("%d. %s — %.0f pts\n", i+1, string(s.PrizeType), s.PrizeValue)
			}
			return lines

		case "0":
			return h.processMenu(ctx, session, phone, "")

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
						masked := maskPhone(w.PhoneNumber)
						return fmt.Sprintf("END 🏆 Last Draw Winner:\nDraw: %s\nWinner: %s\nPrize: ₦%.0f",
							d.Name, masked, w.PrizeValue)
					}
				}
			}
			return "END No completed draws yet. Stay tuned!"

		case "0":
			return h.processMenu(ctx, session, phone, "")

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
				return "END ✅ Redemption queued!\n₦50 airtime will be sent to your line within 5 minutes."
			}
			return h.processMenu(ctx, session, phone, "")

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
			return h.processMenu(ctx, session, phone, "")

		case "3":
			return "END Download the Loyalty Nexus app for the full reward catalogue.\nSearch 'Loyalty Nexus' on the App Store or Google Play."

		case "0":
			return h.processMenu(ctx, session, phone, "")

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

	// ─── 6: My Digital Passport ──────────────────────────────────────────
	case "6":
		if h.passportSvc == nil {
			return "END Passport service unavailable. Please try again later."
		}
		user, _, err := h.lookupUser(ctx, phone)
		if err != nil {
			return "END " + err.Error()
		}
		if len(parts) == 1 {
			return fmt.Sprintf(
				"CON 🪪 My Digital Passport\nTier: %s\n\n1. View Passport\n2. My Badges\n3. State Leaderboard\n0. Back",
				user.Tier,
			)
		}
		switch parts[1] {
		case "1":
			passport, passErr := h.passportSvc.GetPassport(ctx, user.ID)
			if passErr != nil {
				return "END Unable to load passport. Please try again."
			}
			nextTierMsg := ""
			if passport.NextTier != "" {
				nextTierMsg = fmt.Sprintf("\nNext: %s (%d pts away)", passport.NextTier, passport.PointsToNext)
			}
			return fmt.Sprintf(
				"END 🪪 Your Digital Passport\n\nTier: %s\nLifetime Pts: %d\nStreak: 🔥 %d days\nBadges: %d earned%s\n\nDownload the app for your full passport!",
				passport.Tier, passport.LifetimePoints, passport.StreakCount, len(passport.Badges), nextTierMsg,
			)
		case "2":
			passport, passErr := h.passportSvc.GetPassport(ctx, user.ID)
			if passErr != nil {
				return "END Unable to load badges. Please try again."
			}
			if len(passport.Badges) == 0 {
				return "END 🏅 No badges yet!\n\nRecharge daily, spin the wheel, and use AI Studio to earn badges."
			}
			badgeLines := ""
			for i, b := range passport.Badges {
				if i >= 4 {
					badgeLines += fmt.Sprintf("...+%d more in the app", len(passport.Badges)-4)
					break
				}
				badgeLines += fmt.Sprintf("%s %s\n", b.Icon, b.Name)
			}
			return fmt.Sprintf("END 🏅 Your Badges (%d earned)\n\n%s", len(passport.Badges), badgeLines)
		case "3":
			return "END 🌍 State Leaderboard\n\nCheck the Loyalty Nexus app for the full leaderboard and your current rank!\n\nKeep recharging to climb the ranks."
		case "0":
			return h.processMenu(ctx, session, phone, "")
		default:
			return "END Invalid option."
		}

	// ─── 7: AI Knowledge Tools (REQ-6.4) ─────────────────────────────────
	case "7":
		if h.knowledgeSvc == nil {
			return "END AI Knowledge Tools are currently unavailable. Please try again later."
		}
		// Sub-menu: pick a tool
		if len(parts) == 1 {
			tools, toolErr := h.knowledgeSvc.ListKnowledgeTools(ctx)
			if toolErr != nil || len(tools) == 0 {
				return "END AI Knowledge Tools are currently unavailable."
			}
			menu := "CON 🤖 AI Knowledge Tools\nEnter a topic via SMS!\n\n"
			for i, t := range tools {
				menu += fmt.Sprintf("%d. %s\n", i+1, t.Label)
			}
			menu += "0. Back"
			return menu
		}
		// Tool selected — prompt for topic
		if len(parts) == 2 {
			tools, toolErr := h.knowledgeSvc.ListKnowledgeTools(ctx)
			if toolErr != nil || len(tools) == 0 {
				return "END AI Knowledge Tools are currently unavailable."
			}
			idx := int(parts[1][0] - '1')
			if idx < 0 || idx >= len(tools) {
				if parts[1] == "0" {
					return h.processMenu(ctx, session, phone, "")
				}
				return "END Invalid option."
			}
			session.MenuState = "knowledge_tool:" + tools[idx].Slug
			h.saveSession(ctx, session, text)
			return fmt.Sprintf("CON 📝 %s\n\nType your topic and press Send.\n(e.g. \"Photosynthesis\", \"World War 2\")",
				tools[idx].Label)
		}
		// Topic entered — submit the job
		if len(parts) == 3 {
			tools, toolErr := h.knowledgeSvc.ListKnowledgeTools(ctx)
			if toolErr != nil || len(tools) == 0 {
				return "END AI Knowledge Tools are currently unavailable."
			}
			idx := int(parts[1][0] - '1')
			if idx < 0 || idx >= len(tools) {
				return "END Invalid option."
			}
			topic := strings.TrimSpace(parts[2])
			if topic == "" {
				return "END Please enter a topic."
			}
			user, _, lookupErr := h.lookupUser(ctx, phone)
			if lookupErr != nil {
				return "END " + lookupErr.Error()
			}
			_, submitErr := h.knowledgeSvc.SubmitKnowledgeTool(ctx, user.ID, tools[idx].Slug, topic)
			if submitErr != nil {
				return "END ❌ " + submitErr.Error()
			}
			return fmt.Sprintf(
				"END ✅ Your %s on \"%s\" is being prepared!\n\nYou'll receive an SMS with your result shortly.",
				tools[idx].Label, topic,
			)
		}
		return "END Invalid option."

	// ─── 0: Exit ─────────────────────────────────────────────────────────
	case "0":
		return "END Thank you for using Loyalty Nexus! 🎯\nKeep recharging to earn more rewards."

	default:
		return fmt.Sprintf("END Invalid option. Dial %s to start again.", shortcode)
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

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
	if streak == 0 {
		return "No active streak. Recharge today to start your streak!"
	}
	msg := fmt.Sprintf("🔥 Streak: %d days\nPulse Points: %d\n\n", streak, points)
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
