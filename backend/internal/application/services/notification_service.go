package services

import (
	"context"
	"fmt"
	"log"
	"time"
)

type NotificationService struct {
	termiiAPIKey string
}

func NewNotificationService(apiKey string) *NotificationService {
	return &NotificationService{termiiAPIKey: apiKey}
}

func (s *NotificationService) SendSMS(ctx context.Context, msisdn, message string) error {
	// In production: POST to https://api.ng.termii.com/api/sms/send
	log.Printf("[Termii SMS] To: %s | Msg: %s", msisdn, message)
	return nil
}

func (s *NotificationService) NotifyAssetReady(ctx context.Context, msisdn, toolName string) error {
	msg := fmt.Sprintf("Your Loyalty Nexus %s is ready! Open the app gallery to view and download it.", toolName)
	return s.SendSMS(ctx, msisdn, msg)
}
