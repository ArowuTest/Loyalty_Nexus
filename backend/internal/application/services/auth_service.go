package services

import (
	"context"
	"fmt"
	"log"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"time"
)

type AuthService struct {
	authRepo  repositories.AuthRepository
	userRepo  repositories.UserRepository
	notifySvc *NotificationService
	jwtSecret []byte
}

func NewAuthService(ar repositories.AuthRepository, ur repositories.UserRepository, ns *NotificationService, secret string) *AuthService {
	return &AuthService{
		authRepo:  ar,
		userRepo:  ur,
		notifySvc: ns,
		jwtSecret: []byte(secret),
	}
}

func (s *AuthService) SendLoginOTP(ctx context.Context, msisdn string) error {
	code := s.generateNumericOTP(6)
	
	otp := &entities.AuthOTP{
		MSISDN:    msisdn,
		Code:      code,
		Purpose:   entities.PurposeLogin,
		Status:    "pending",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	if err := s.authRepo.CreateOTP(ctx, otp); err != nil {
		return err
	}

	// Delivering via Template SMS (REQ-5.7.1)
	return s.notifySvc.SendTemplateSMS(ctx, msisdn, "otp_delivery", map[string]string{"code": code})
}

// ... existing logic ...
func (s *AuthService) VerifyLogin(ctx context.Context, msisdn, code string) (string, error) {
	otp, err := s.authRepo.FindPendingOTP(ctx, msisdn, code, entities.PurposeLogin)
	if err != nil {
		return "", fmt.Errorf("invalid or expired code")
	}

	s.authRepo.MarkOTPUsed(ctx, otp.ID)

	user, err := s.userRepo.FindByMSISDN(ctx, msisdn)
	if err != nil {
		user = &entities.User{
			MSISDN: msisdn,
			Tier:   "BRONZE",
		}
		s.userRepo.Create(ctx, user)
	}

	return "mock-jwt-token", nil
}

func (s *AuthService) generateNumericOTP(length int) string {
	return "123456" // mock
}
