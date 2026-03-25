package services

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"time"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrOTPNotFound   = errors.New("OTP not found or already used")
	ErrOTPExpired    = errors.New("OTP has expired")
	ErrOTPInvalid    = errors.New("OTP code is incorrect")
	ErrUserBanned    = errors.New("account is suspended")
)

type AuthService struct {
	authRepo    repositories.AuthRepository
	userRepo    repositories.UserRepository
	notifySvc   *NotificationService
	cfg         *config.ConfigManager
	jwtSecret   []byte
	aesKey      []byte
}

func NewAuthService(
	ar repositories.AuthRepository,
	ur repositories.UserRepository,
	ns *NotificationService,
	cfg *config.ConfigManager,
) *AuthService {
	jwtSecret := []byte(mustEnv("JWT_SECRET"))
	aesHex := mustEnv("AES_256_KEY") // 32-byte hex (64 chars)
	aesKey, err := hex.DecodeString(aesHex)
	if err != nil || len(aesKey) != 32 {
		panic("AES_256_KEY must be a 64-char hex string (32 bytes)")
	}
	return &AuthService{
		authRepo:  ar,
		userRepo:  ur,
		notifySvc: ns,
		cfg:       cfg,
		jwtSecret: jwtSecret,
		aesKey:    aesKey,
	}
}

// SendOTP generates a 6-digit OTP, encrypts it with AES-256-GCM, and delivers via Termii.
func (s *AuthService) SendOTP(ctx context.Context, phone, purpose string) error {
	// Generate 6-digit code using CSPRNG
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return fmt.Errorf("failed to generate OTP: %w", err)
	}
	code := fmt.Sprintf("%06d", n.Int64()+100000)

	// Encrypt for storage (AES-256-GCM)
	encrypted, err := s.encrypt(code)
	if err != nil {
		return fmt.Errorf("failed to encrypt OTP: %w", err)
	}

	otp := &entities.AuthOTP{
		ID:          uuid.New(),
		PhoneNumber: phone,
		Code:        encrypted,
		Purpose:     entities.OTPPurpose(purpose),
		Status:      entities.OTPPending,
		ExpiresAt:   time.Now().Add(5 * time.Minute),
		CreatedAt:   time.Now(),
	}

	if err := s.authRepo.CreateOTP(ctx, otp); err != nil {
		return fmt.Errorf("failed to save OTP: %w", err)
	}

	// Deliver SMS via Termii
	return s.notifySvc.SendOTP(ctx, phone, code)
}

// VerifyOTP checks the OTP and returns a JWT on success.
// If the user does not exist, they are auto-registered (first-time flow).
func (s *AuthService) VerifyOTP(ctx context.Context, phone, code, purpose string) (string, bool, error) {
	otp, err := s.authRepo.FindLatestPendingOTP(ctx, phone, purpose)
	if err != nil {
		return "", false, ErrOTPNotFound
	}
	if time.Now().After(otp.ExpiresAt) {
		_ = s.authRepo.ExpireOTP(ctx, otp.ID)
		return "", false, ErrOTPExpired
	}

	// Decrypt and compare
	decrypted, err := s.decrypt(otp.Code)
	if err != nil || decrypted != code {
		return "", false, ErrOTPInvalid
	}

	_ = s.authRepo.MarkOTPUsed(ctx, otp.ID)

	// Auto-register if new user
	isNew := false
	user, err := s.userRepo.FindByPhoneNumber(ctx, phone)
	if err != nil {
		user, err = s.registerNewUser(ctx, phone)
		if err != nil {
			return "", false, fmt.Errorf("registration failed: %w", err)
		}
		isNew = true
	}

	if !user.IsActive {
		return "", false, ErrUserBanned
	}

	token, err := s.issueJWT(user)
	if err != nil {
		return "", false, fmt.Errorf("JWT issue failed: %w", err)
	}

	return token, isNew, nil
}

// ValidateJWT parses and validates a JWT, returning claims.
func (s *AuthService) ValidateJWT(tokenStr string) (*entities.JWTClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	return &entities.JWTClaims{
		UserID:      fmt.Sprintf("%v", claims["uid"]),
		PhoneNumber: fmt.Sprintf("%v", claims["phone"]),
		IsAdmin:     false,
	}, nil
}

func (s *AuthService) registerNewUser(ctx context.Context, phone string) (*entities.User, error) {
	user := &entities.User{
		ID:               uuid.New(),
		PhoneNumber:      phone,
		UserCode:         generateUserCode(),
		Tier:             entities.TierBronze,
		IsActive:         true,
		DeviceType:       "smartphone",
		SubscriptionTier: "free",
		KYCStatus:        "unverified",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) issueJWT(user *entities.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid":   user.ID.String(),
		"phone": user.PhoneNumber,
		"tier":  user.Tier,
		"exp":   time.Now().Add(30 * 24 * time.Hour).Unix(), // 30-day session
		"iat":   time.Now().Unix(),
	})
	return token.SignedString(s.jwtSecret)
}

// encrypt uses AES-256-GCM. Returns base64-encoded ciphertext.
func (s *AuthService) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.aesKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

func (s *AuthService) decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(s.aesKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := data[:nonceSize], data[nonceSize:]
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

func generateUserCode() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failure is extremely unlikely; fall back to timestamp-based code
		return fmt.Sprintf("NXS%08X", uint32(time.Now().UnixNano()))
	}
	return "NXS" + fmt.Sprintf("%08X", b)
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("required environment variable not set: " + key)
	}
	return v
}
