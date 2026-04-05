package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"time"
	"github.com/google/uuid"
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

func (s *AuthService) SendLoginOTP(ctx context.Context, phoneNumber string) error {
	code := s.generateNumericOTP(6)

	otp := &entities.AuthOTP{
		PhoneNumber: phoneNumber,
		Code:        code,
		Purpose:     entities.OTPLogin,
		Status:      "pending",
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	}

	if err := s.authRepo.CreateOTP(ctx, otp); err != nil {
		return err
	}

	return s.notifySvc.SendTemplateSMS(ctx, phoneNumber, "otp_delivery", map[string]string{"code": code})
}

func (s *AuthService) VerifyLogin(ctx context.Context, phoneNumber, code string) (string, error) {
	otp, err := s.authRepo.FindLatestPendingOTP(ctx, phoneNumber, code, entities.OTPLogin)
	if err != nil {
		return "", fmt.Errorf("invalid or expired code")
	}

	s.authRepo.MarkOTPUsed(ctx, otp.ID)

	user, err := s.userRepo.FindByPhoneNumber(ctx, phoneNumber)
	if err != nil {
		user = &entities.User{
			ID:          uuid.New(),
			PhoneNumber: phoneNumber,
			UserCode:    fmt.Sprintf("NEX%s", uuid.New().String()[:6]),
			Tier:        "BRONZE",
		}
		s.userRepo.Create(ctx, user)
	}

	_ = user
	return "mock-jwt-token", nil
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
