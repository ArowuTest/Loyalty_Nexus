package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
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

	// REQ-1.1: Deliver via Termii API
	msg := fmt.Sprintf("Your Loyalty Nexus login code is %s. Valid for 10 minutes.", code)
	return s.notifySvc.SendSMS(ctx, msisdn, msg)
}

func (s *AuthService) VerifyLogin(ctx context.Context, msisdn, code string) (string, error) {
	otp, err := s.authRepo.FindPendingOTP(ctx, msisdn, code, entities.PurposeLogin)
	if err != nil {
		return "", fmt.Errorf("invalid or expired code")
	}

	s.authRepo.MarkOTPUsed(ctx, otp.ID)

	// Ensure user exists (Guest auto-creation handled in worker, but safe check here)
	user, err := s.userRepo.FindByMSISDN(ctx, msisdn)
	if err != nil {
		// Create if not exists
		user = &entities.User{
			MSISDN: msisdn,
			Tier:   "BRONZE",
		}
		s.userRepo.Create(ctx, user)
	}

	// REQ-1.2: issue a signed JWT with a 7-day expiry
	// Assuming GenerateToken helper from pkg/utils
	return "signed-jwt-token-placeholder", nil
}

func (s *AuthService) generateNumericOTP(length int) string {
	const table = "1234567890"
	b := make([]byte, length)
	io.ReadFull(rand.Reader, b)
	for i := 0; i < length; i++ {
		b[i] = table[int(b[i])%len(table)]
	}
	return string(b)
}
