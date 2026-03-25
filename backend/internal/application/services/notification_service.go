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

// NotificationService sends SMS via Termii (primary) and Africa's Talking (fallback).
type NotificationService struct {
	termiiKey string
	atKey     string
	httpClient *http.Client
}

func NewNotificationService(termiiKey string) *NotificationService {
	return &NotificationService{
		termiiKey: termiiKey,
		atKey:     os.Getenv("AFRICAS_TALKING_API_KEY"),
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
func (n *NotificationService) SendSMS(ctx context.Context, phone, message string) error {
	err := n.sendViaTermii(ctx, phone, message)
	if err != nil {
		log.Printf("[NOTIFY] Termii failed, trying Africa's Talking: %v", err)
		return n.sendViaAfricasTalking(ctx, phone, message)
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

func (n *NotificationService) sendViaAfricasTalking(ctx context.Context, phone, message string) error {
	if n.atKey == "" {
		return fmt.Errorf("Africa's Talking key not configured")
	}
	payload := map[string]string{
		"username": "loyalty_nexus",
		"to":       phone,
		"message":  message,
		"apiKey":   n.atKey,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.africastalking.com/version1/messaging", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apiKey", n.atKey)

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("Africa's Talking HTTP %d", resp.StatusCode)
	}
	return nil
}
