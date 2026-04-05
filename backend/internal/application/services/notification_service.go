package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"gorm.io/gorm"
)

type NotificationService struct {
	db           *gorm.DB
	termiiAPIKey string
}

func NewNotificationService(db *gorm.DB, apiKey string) *NotificationService {
	return &NotificationService{db: db, termiiAPIKey: apiKey}
}

func (s *NotificationService) SendSMS(ctx context.Context, msisdn, message string) error {
	log.Printf("[Termii SMS] To: %s | Msg: %s", msisdn, message)
	// In production: POST to Termii REST API
	return nil
}

// SendTemplateSMS fetches a template from DB, replaces placeholders, and sends (REQ-5.7.1)
func (s *NotificationService) SendTemplateSMS(ctx context.Context, msisdn, slug string, params map[string]string) error {
	var t struct {
		ContentTemplate string
	}
	err := s.db.WithContext(ctx).Table("notification_templates").
		Where("slug = ? AND is_active = true", slug).
		First(&t).Error
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	msg := t.ContentTemplate
	for k, v := range params {
		msg = strings.ReplaceAll(msg, "{{"+k+"}}", v)
	}

	return s.SendSMS(ctx, msisdn, msg)
}

func (s *NotificationService) NotifyAssetReady(ctx context.Context, msisdn, toolName string) error {
	return s.SendTemplateSMS(ctx, msisdn, "asset_completion", map[string]string{"tool_name": toolName})
}
