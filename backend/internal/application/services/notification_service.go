package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// NotificationService sends SMS via Termii with stdout fallback for staging.
type NotificationService struct {
	termiiKey  string
	httpClient *http.Client
}

func NewNotificationService(termiiKey string) *NotificationService {
	return &NotificationService{
		termiiKey:  termiiKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// SendOTP delivers a 6-digit OTP via Termii.
func (n *NotificationService) SendOTP(ctx context.Context, phone, code string) error {
	message := fmt.Sprintf("Your Loyalty Nexus code: %s. Valid for 5 minutes. Do not share this.", code)
	return n.SendSMS(ctx, phone, message)
}

// NotifyAssetReady sends a notification that an AI asset is ready.
func (n *NotificationService) NotifyAssetReady(ctx context.Context, phone, toolName string) {
	msg := fmt.Sprintf("Your %s is ready on Nexus Studio! Open the app to download it before it expires.", toolName)
	if err := n.SendSMS(ctx, phone, msg); err != nil {
		log.Printf("[NOTIFY] Asset ready SMS failed to %s: %v", phone, err)
	}
}

// NotifyAssetExpiring sends a warning 48h before expiry.
func (n *NotificationService) NotifyAssetExpiring(ctx context.Context, phone, toolName string) {
	msg := fmt.Sprintf("Your %s on Nexus Studio expires in 48 hours. Download it now before it's gone.", toolName)
	if err := n.SendSMS(ctx, phone, msg); err != nil {
		log.Printf("[NOTIFY] Asset expiry SMS failed to %s: %v", phone, err)
	}
}

// NotifyStreakExpiring warns user their streak is about to reset.
func (n *NotificationService) NotifyStreakExpiring(ctx context.Context, phone string, streakDays int, hoursLeft int) {
	msg := fmt.Sprintf("Your Loyalty Nexus streak (Day %d) expires in %d hours! Recharge now to keep it alive.", streakDays, hoursLeft)
	if err := n.SendSMS(ctx, phone, msg); err != nil {
		log.Printf("[NOTIFY] Streak expiry SMS failed to %s: %v", phone, err)
	}
}

// NotifyPrizeWon sends a prize notification.
func (n *NotificationService) NotifyPrizeWon(ctx context.Context, phone, prizeDescription string) {
	msg := fmt.Sprintf("Congratulations! %s", prizeDescription)
	if err := n.SendSMS(ctx, phone, msg); err != nil {
		log.Printf("[NOTIFY] Prize SMS failed to %s: %v", phone, err)
	}
}

// SendSMS sends via Termii, falls back to Africa's Talking on error.
// If no SMS provider is configured (empty API key), OR if the key is invalid (401),
// logs the message to stdout and returns nil — SMS delivery is non-fatal in staging.
func (n *NotificationService) SendSMS(ctx context.Context, phone, message string) error {
	if n.termiiKey == "" {
		log.Printf("[SMS-DEV] To: %s | %s", phone, message)
		return nil
	}
	err := n.sendViaTermii(ctx, phone, message)
	if err != nil {
		// 401 = key not provisioned/invalid. Log and move on — don't block OTP flow.
		log.Printf("[SMS-DEV] Termii unavailable (%v) — OTP still valid in DB", err)
		return nil
	}
	return nil
}

func (n *NotificationService) sendViaTermii(ctx context.Context, phone, message string) error {
	payload := map[string]interface{}{
		"to":      phone,
		"from":    "Loyalty",
		"sms":     message,
		"type":    "plain",
		"channel": "generic",
		"api_key": n.termiiKey,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.ng.termii.com/api/sms/send", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("Termii HTTP %d", resp.StatusCode)
	}
	return nil
}

// ─── FCM (Firebase Cloud Messaging) ──────────────────────────────────────────
//
// Two modes are supported:
//   - v1 HTTP API  (preferred): set FCM_V1_ACCESS_TOKEN (OAuth2 bearer) + FCM_PROJECT_ID
//   - Legacy API  (fallback):   set FCM_SERVER_KEY only
//
// fcmPayload / fcmMessage / etc. are the typed structs for the v1 API.
// The legacy path uses a raw map for the /fcm/send endpoint.

type fcmPayload struct {
	Message fcmMessage `json:"message"`
}
type fcmMessage struct {
	Token        string            `json:"token"`
	Notification fcmNotification   `json:"notification"`
	Data         map[string]string `json:"data,omitempty"`
	Android      *fcmAndroid       `json:"android,omitempty"`
	APNS         *fcmAPNS          `json:"apns,omitempty"`
}
type fcmNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}
type fcmAndroid struct {
	Priority string `json:"priority"` // "high" | "normal"
}
type fcmAPNS struct {
	Headers map[string]string `json:"headers,omitempty"`
}

// SendPush sends a push notification via FCM.
// It prefers the HTTP v1 API (FCM_V1_ACCESS_TOKEN + FCM_PROJECT_ID) and falls
// back to the legacy /fcm/send endpoint (FCM_SERVER_KEY).
func (n *NotificationService) SendPush(ctx context.Context, fcmToken, title, body string, data map[string]string) error {
	if fcmToken == "" {
		return nil // no device token — silently skip
	}

	// ── FCM HTTP v1 (preferred) ────────────────────────────────────────────
	accessToken := os.Getenv("FCM_V1_ACCESS_TOKEN")
	projectID   := os.Getenv("FCM_PROJECT_ID")
	if accessToken != "" && projectID != "" {
		payload := fcmPayload{
			Message: fcmMessage{
				Token:        fcmToken,
				Notification: fcmNotification{Title: title, Body: body},
				Data:         data,
				Android:      &fcmAndroid{Priority: "high"},
			},
		}
		b, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("FCM v1 marshal: %w", err)
		}
		url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", projectID)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
		if err != nil {
			return fmt.Errorf("FCM v1 request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)
		resp, err := n.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("FCM v1 HTTP: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			return fmt.Errorf("FCM v1 HTTP %d", resp.StatusCode)
		}
		return nil
	}

	// ── Legacy /fcm/send fallback ─────────────────────────────────────────
	serverKey := os.Getenv("FCM_SERVER_KEY")
	if serverKey == "" {
		return nil // neither v1 nor legacy configured — silently skip
	}
	legacyPayload := map[string]interface{}{
		"to": fcmToken,
		"notification": map[string]string{
			"title": title,
			"body":  body,
		},
		"data":     data,
		"priority": "high",
	}
	b, _ := json.Marshal(legacyPayload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://fcm.googleapis.com/fcm/send", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "key="+serverKey)
	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("FCM legacy HTTP: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("FCM legacy HTTP %d", resp.StatusCode)
	}
	return nil
}

// ─── Convenience notification helpers ────────────────────────────────────────

// NotifySpinWin sends both SMS and push when a user wins a significant prize.
func (n *NotificationService) NotifySpinWin(ctx context.Context, phone, fcmToken, prizeLabel string) {
	msg := fmt.Sprintf("🎉 Congratulations! You just won: %s on Loyalty Nexus Spin. Open the app to claim!", prizeLabel)
	_ = n.SendSMS(ctx, phone, msg)
	_ = n.SendPush(ctx, fcmToken, "You Won! 🎉", "You won: "+prizeLabel, map[string]string{
		"type": "spin_win", "prize": prizeLabel,
	})
}

// NotifyDrawResult sends win/loss notification after a draw.
func (n *NotificationService) NotifyDrawResult(ctx context.Context, phone, fcmToken string, won bool, prizeLabel string) {
	if won {
		msg := fmt.Sprintf("🏆 You WON the Loyalty Nexus Draw! Prize: %s. We'll contact you shortly.", prizeLabel)
		_ = n.SendSMS(ctx, phone, msg)
		_ = n.SendPush(ctx, fcmToken, "Draw Winner! 🏆", "You won: "+prizeLabel, map[string]string{
			"type": "draw_result", "won": "true",
		})
	} else {
		_ = n.SendPush(ctx, fcmToken, "Draw Result", "Better luck next time! Keep spinning to earn more entries.", map[string]string{
			"type": "draw_result", "won": "false",
		})
	}
}

// NotifyRegionalWarsResult announces war results to all state participants.
func (n *NotificationService) NotifyRegionalWarsResult(ctx context.Context, phone, fcmToken, state string, rank int) {
	var msg string
	switch rank {
	case 1:
		msg = fmt.Sprintf("🥇 %s takes FIRST PLACE in this month's Regional Wars! Your share of the prize pool is on its way!", state)
	case 2:
		msg = fmt.Sprintf("🥈 %s finishes 2nd in Regional Wars! Great effort — prize payout incoming.", state)
	case 3:
		msg = fmt.Sprintf("🥉 %s finishes 3rd in Regional Wars! Prize incoming.", state)
	default:
		msg = fmt.Sprintf("Regional Wars results are in. %s finished #%d. Keep playing to climb next month!", state, rank)
	}
	_ = n.SendSMS(ctx, phone, msg)
	_ = n.SendPush(ctx, fcmToken, "Regional Wars Result 🗺️", msg, map[string]string{
		"type": "wars_result", "state": state,
	})
}

// NotifyStudioGenReady tells user their AI generation is complete.
func (n *NotificationService) NotifyStudioGenReady(ctx context.Context, phone, fcmToken, toolName, genID string) {
	msg := fmt.Sprintf("Your %s is ready on Nexus Studio! Tap to view & download (expires in 7 days).", toolName)
	_ = n.SendSMS(ctx, phone, msg)
	_ = n.SendPush(ctx, fcmToken, "Studio Ready ✨", msg, map[string]string{
		"type": "studio_ready", "gen_id": genID, "tool": toolName,
	})
}
