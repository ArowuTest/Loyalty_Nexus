package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"loyalty-nexus/internal/domain/entities"
)

// AdminAuthService handles admin authentication (email + bcrypt password),
// RBAC token issuance, and refresh token lifecycle management.
type AdminAuthService struct {
	db        *gorm.DB
	jwtSecret []byte
}

func NewAdminAuthService(db *gorm.DB) *AdminAuthService {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "change-this-in-production"
	}
	svc := &AdminAuthService{db: db, jwtSecret: []byte(secret)}
	svc.seedDefaultAdmin()
	return svc
}

// ─── Token durations ─────────────────────────────────────────────────────────
const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
)

// ─── LoginResult holds both tokens returned on successful login ───────────────
type LoginResult struct {
	AccessToken  string
	RefreshToken string
	Admin        *entities.AdminUser
}

// Login verifies email + password and returns access + refresh tokens on success.
func (s *AdminAuthService) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	var admin entities.AdminUser
	if err := s.db.WithContext(ctx).
		Where("email = ? AND is_active = true", email).
		First(&admin).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[AdminAuth] Login failed: no active admin with email=%s", email)
			return nil, errors.New("invalid credentials")
		}
		log.Printf("[AdminAuth] Login DB error for email=%s: %v", email, err)
		return nil, fmt.Errorf("db error: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		log.Printf("[AdminAuth] Login failed: password mismatch for email=%s", email)
		return nil, errors.New("invalid credentials")
	}

	// Update last login
	s.db.WithContext(ctx).Model(&admin).Update("last_login_at", time.Now())

	accessToken, err := s.mintAdminJWT(&admin)
	if err != nil {
		return nil, fmt.Errorf("access token mint failed: %w", err)
	}

	refreshToken, err := s.issueRefreshToken(ctx, admin.ID, "", "")
	if err != nil {
		return nil, fmt.Errorf("refresh token issue failed: %w", err)
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Admin:        &admin,
	}, nil
}

// Refresh validates a refresh token and issues a new access token + rotated refresh token.
func (s *AdminAuthService) Refresh(ctx context.Context, rawRefreshToken string) (*LoginResult, error) {
	tokenHash := hashToken(rawRefreshToken)

	var rt struct {
		ID        uuid.UUID  `gorm:"column:id"`
		AdminID   uuid.UUID  `gorm:"column:admin_id"`
		ExpiresAt time.Time  `gorm:"column:expires_at"`
		RevokedAt *time.Time `gorm:"column:revoked_at"`
	}
	if err := s.db.WithContext(ctx).
		Table("admin_refresh_tokens").
		Where("token_hash = ?", tokenHash).
		First(&rt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("refresh token not found")
		}
		return nil, fmt.Errorf("db error: %w", err)
	}

	if rt.RevokedAt != nil {
		return nil, errors.New("refresh token has been revoked")
	}
	if time.Now().After(rt.ExpiresAt) {
		return nil, errors.New("refresh token expired")
	}

	// Revoke the used refresh token (rotation — one-time use)
	now := time.Now()
	s.db.WithContext(ctx).
		Table("admin_refresh_tokens").
		Where("id = ?", rt.ID).
		Update("revoked_at", now)

	// Load the admin
	var admin entities.AdminUser
	if err := s.db.WithContext(ctx).
		Where("id = ? AND is_active = true", rt.AdminID).
		First(&admin).Error; err != nil {
		return nil, errors.New("admin not found or deactivated")
	}

	// Issue new access + refresh tokens
	accessToken, err := s.mintAdminJWT(&admin)
	if err != nil {
		return nil, fmt.Errorf("access token mint failed: %w", err)
	}

	newRefreshToken, err := s.issueRefreshToken(ctx, admin.ID, "", "")
	if err != nil {
		return nil, fmt.Errorf("refresh token issue failed: %w", err)
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		Admin:        &admin,
	}, nil
}

// Logout revokes all refresh tokens for the given admin.
func (s *AdminAuthService) Logout(ctx context.Context, adminID uuid.UUID) error {
	now := time.Now()
	return s.db.WithContext(ctx).
		Table("admin_refresh_tokens").
		Where("admin_id = ? AND revoked_at IS NULL", adminID).
		Update("revoked_at", now).Error
}

// RevokeRefreshToken revokes a specific refresh token by its raw value.
func (s *AdminAuthService) RevokeRefreshToken(ctx context.Context, rawToken string) error {
	tokenHash := hashToken(rawToken)
	now := time.Now()
	return s.db.WithContext(ctx).
		Table("admin_refresh_tokens").
		Where("token_hash = ? AND revoked_at IS NULL", tokenHash).
		Update("revoked_at", now).Error
}

// CreateAdmin creates a new admin user (super_admin only operation, enforced at handler layer).
func (s *AdminAuthService) CreateAdmin(ctx context.Context, email, password, fullName string, role entities.AdminRole) (*entities.AdminUser, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash error: %w", err)
	}
	admin := &entities.AdminUser{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
		FullName:     fullName,
		Role:         role,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := s.db.WithContext(ctx).Create(admin).Error; err != nil {
		return nil, fmt.Errorf("create admin: %w", err)
	}
	return admin, nil
}

// UpdatePassword allows an admin to change their own password.
func (s *AdminAuthService) UpdatePassword(ctx context.Context, adminID uuid.UUID, oldPassword, newPassword string) error {
	var admin entities.AdminUser
	if err := s.db.WithContext(ctx).Where("id = ?", adminID).First(&admin).Error; err != nil {
		return errors.New("admin not found")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(oldPassword)); err != nil {
		return errors.New("current password incorrect")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&admin).Updates(map[string]interface{}{
		"password_hash": string(hash),
		"updated_at":    time.Now(),
	}).Error
}

// ListAdmins returns all admin users (super_admin only).
func (s *AdminAuthService) ListAdmins(ctx context.Context) ([]entities.AdminUser, error) {
	var admins []entities.AdminUser
	err := s.db.WithContext(ctx).Order("created_at asc").Find(&admins).Error
	return admins, err
}

// DeactivateAdmin marks an admin as inactive and revokes all their refresh tokens.
func (s *AdminAuthService) DeactivateAdmin(ctx context.Context, adminID uuid.UUID) error {
	// Revoke all active refresh tokens
	_ = s.Logout(ctx, adminID)
	return s.db.WithContext(ctx).
		Model(&entities.AdminUser{}).
		Where("id = ?", adminID).
		Updates(map[string]interface{}{"is_active": false, "updated_at": time.Now()}).Error
}

// ValidateAdminJWT parses an admin token and returns claims.
func (s *AdminAuthService) ValidateAdminJWT(tokenStr string) (*entities.JWTClaims, error) {
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
	isAdmin, _ := claims["is_admin"].(bool)
	if !isAdmin {
		return nil, errors.New("not an admin token")
	}
	role, _ := claims["role"].(string)
	email, _ := claims["email"].(string)
	return &entities.JWTClaims{
		UserID:  fmt.Sprintf("%v", claims["uid"]),
		Email:   email,
		Role:    entities.AdminRole(role),
		IsAdmin: true,
	}, nil
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

func (s *AdminAuthService) mintAdminJWT(admin *entities.AdminUser) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid":      admin.ID.String(),
		"email":    admin.Email,
		"role":     string(admin.Role),
		"is_admin": true,
		"exp":      time.Now().Add(accessTokenTTL).Unix(),
		"iat":      time.Now().Unix(),
	})
	return token.SignedString(s.jwtSecret)
}

// issueRefreshToken generates a cryptographically random refresh token,
// stores its SHA-256 hash in the DB, and returns the raw token to the caller.
func (s *AdminAuthService) issueRefreshToken(ctx context.Context, adminID uuid.UUID, userAgent, ipAddress string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("rand error: %w", err)
	}
	rawHex := hex.EncodeToString(raw)
	tokenHash := hashToken(rawHex)

	row := map[string]interface{}{
		"id":         uuid.New(),
		"admin_id":   adminID,
		"token_hash": tokenHash,
		"expires_at": time.Now().Add(refreshTokenTTL),
		"created_at": time.Now(),
		"user_agent": userAgent,
		"ip_address": ipAddress,
	}
	if err := s.db.WithContext(ctx).Table("admin_refresh_tokens").Create(row).Error; err != nil {
		return "", fmt.Errorf("store refresh token: %w", err)
	}
	return rawHex, nil
}

// hashToken returns the SHA-256 hex digest of a raw token string.
func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// MintIntegrationTestToken issues a short-lived admin JWT for integration tests only.
func (s *AdminAuthService) MintIntegrationTestToken(adminID uuid.UUID) (string, error) {
	admin := &entities.AdminUser{
		ID:    adminID,
		Email: "test-admin@localhost",
		Role:  entities.RoleSuperAdmin,
	}
	return s.mintAdminJWT(admin)
}

func (s *AdminAuthService) seedDefaultAdmin() {
	email := os.Getenv("ADMIN_SEED_EMAIL")
	password := os.Getenv("ADMIN_SEED_PASSWORD")
	if email == "" || password == "" {
		log.Println("[AdminAuth] ADMIN_SEED_EMAIL/ADMIN_SEED_PASSWORD not set — skipping admin seed.")
		return
	}

	// If ADMIN_SEED_FORCE_RESET=true, update the password for the existing admin with this email.
	if os.Getenv("ADMIN_SEED_FORCE_RESET") == "true" {
		var admin entities.AdminUser
		if err := s.db.Where("email = ?", email).First(&admin).Error; err == nil {
			hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err == nil {
				s.db.Model(&admin).Update("password_hash", string(hashed))
				log.Printf("[AdminAuth] ✅ Admin password force-reset for: %s", email)
			}
			return
		}
	}

	var count int64
	s.db.Model(&entities.AdminUser{}).Count(&count)
	if count > 0 {
		return
	}
	if _, err := s.CreateAdmin(context.Background(), email, password, "Platform Admin", entities.RoleSuperAdmin); err != nil {
		log.Printf("[AdminAuth] Failed to seed default admin: %v", err)
		return
	}
	log.Printf("[AdminAuth] ✅ Default super_admin seeded: %s", email)
}
