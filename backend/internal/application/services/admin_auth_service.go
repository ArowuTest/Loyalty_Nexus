package services

import (
	"context"
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

// AdminAuthService handles admin authentication (email + bcrypt password) and RBAC token issuance.
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
	// Seed default super_admin on startup if no admin exists
	svc.seedDefaultAdmin()
	return svc
}

// Login verifies email + password and returns a signed JWT on success.
func (s *AdminAuthService) Login(ctx context.Context, email, password string) (string, *entities.AdminUser, error) {
	var admin entities.AdminUser
	if err := s.db.WithContext(ctx).
		Where("email = ? AND is_active = true", email).
		First(&admin).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, errors.New("invalid credentials")
		}
		return "", nil, fmt.Errorf("db error: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	// Update last login
	s.db.WithContext(ctx).Model(&admin).Update("last_login_at", time.Now())

	token, err := s.mintAdminJWT(&admin)
	if err != nil {
		return "", nil, fmt.Errorf("token mint failed: %w", err)
	}
	return token, &admin, nil
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

// DeactivateAdmin marks an admin as inactive.
func (s *AdminAuthService) DeactivateAdmin(ctx context.Context, adminID uuid.UUID) error {
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

func (s *AdminAuthService) mintAdminJWT(admin *entities.AdminUser) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid":      admin.ID.String(),
		"email":    admin.Email,
		"role":     string(admin.Role),
		"is_admin": true,
		"exp":      time.Now().Add(12 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	})
	return token.SignedString(s.jwtSecret)
}

// seedDefaultAdmin creates a super_admin from env vars if no admin exists.
// ADMIN_SEED_EMAIL and ADMIN_SEED_PASSWORD must be set.
func (s *AdminAuthService) seedDefaultAdmin() {
	var count int64
	s.db.Model(&entities.AdminUser{}).Count(&count)
	if count > 0 {
		return
	}
	email := os.Getenv("ADMIN_SEED_EMAIL")
	password := os.Getenv("ADMIN_SEED_PASSWORD")
	if email == "" || password == "" {
		log.Println("[AdminAuth] No admins exist and ADMIN_SEED_EMAIL/ADMIN_SEED_PASSWORD not set. " +
			"Set these env vars to create the first super_admin on startup.")
		return
	}
	if _, err := s.CreateAdmin(context.Background(), email, password, "Platform Admin", entities.RoleSuperAdmin); err != nil {
		log.Printf("[AdminAuth] Failed to seed default admin: %v", err)
		return
	}
	log.Printf("[AdminAuth] ✅ Default super_admin seeded: %s", email)
}
