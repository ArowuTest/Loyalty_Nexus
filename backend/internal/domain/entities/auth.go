package entities

import (
	"time"
	"github.com/google/uuid"
)

type OTPPurpose string
const (
	OTPLogin      OTPPurpose = "login"
	OTPMoMoLink   OTPPurpose = "momo_link"
	OTPPrizeClaim OTPPurpose = "prize_claim"
)

type OTPStatus string
const (
	OTPPending  OTPStatus = "pending"
	OTPVerified OTPStatus = "verified"
	OTPExpired  OTPStatus = "expired"
)

type AuthOTP struct {
	ID          uuid.UUID  `gorm:"column:id;primaryKey;type:uuid"       db:"id"          json:"id"`
	PhoneNumber string     `gorm:"column:phone_number;not null"          db:"phone_number" json:"-"`
	Code        string     `gorm:"column:code;not null"                  db:"code"        json:"-"`     // AES-256 encrypted at rest
	Purpose     OTPPurpose `gorm:"column:purpose;default:login"          db:"purpose"     json:"purpose"`
	Status      OTPStatus  `gorm:"column:status;default:pending"         db:"status"      json:"status"`
	ExpiresAt   time.Time  `gorm:"column:expires_at;not null"            db:"expires_at"  json:"expires_at"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime"      db:"created_at"  json:"created_at"`
}

func (AuthOTP) TableName() string { return "auth_otps" }

// AdminRole defines the RBAC roles for admin users.
type AdminRole string
const (
	RoleSuperAdmin  AdminRole = "super_admin"  // Full platform access
	RoleFinance     AdminRole = "finance"       // Approve claims, view financials, adjust points
	RoleOperations  AdminRole = "operations"    // Manage draws, notifications, users, wars
	RoleContent     AdminRole = "content"       // Manage studio tools, prizes, config
)

// AdminUser represents a platform admin with email/password credentials and RBAC role.
type AdminUser struct {
	ID           uuid.UUID  `gorm:"column:id;primaryKey;type:uuid"          db:"id"            json:"id"`
	Email        string     `gorm:"column:email;uniqueIndex"                db:"email"         json:"email"`
	PasswordHash string     `gorm:"column:password_hash"                    db:"password_hash" json:"-"`
	FullName     string     `gorm:"column:full_name"                        db:"full_name"     json:"full_name"`
	Role         AdminRole  `gorm:"column:role"                             db:"role"            json:"role"`
	IsActive     bool       `gorm:"column:is_active"                        db:"is_active"        json:"is_active"`
	LastLoginAt  *time.Time `gorm:"column:last_login_at"                    db:"last_login_at" json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime"        db:"created_at"    json:"created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;autoUpdateTime"        db:"updated_at"    json:"updated_at"`
}

func (AdminUser) TableName() string { return "admin_users" }

// JWTClaims used for both user and admin tokens.
type JWTClaims struct {
	UserID      string    `json:"uid"`
	PhoneNumber string    `json:"phone,omitempty"`
	Email       string    `json:"email,omitempty"`   // populated for admin tokens
	Role        AdminRole `json:"role,omitempty"`    // populated for admin tokens
	IsAdmin     bool      `json:"is_admin"`
}

// HasPermission returns true if the admin role is allowed to perform the action.
// super_admin can do everything. Other roles have specific domains.
func (c *JWTClaims) HasPermission(action string) bool {
	if !c.IsAdmin {
		return false
	}
	if c.Role == RoleSuperAdmin {
		return true
	}
	permissions := map[AdminRole][]string{
		RoleFinance: {
			"claims:approve", "claims:reject", "claims:view",
			"points:adjust", "points:view",
			"dashboard:view", "users:view",
		},
		RoleOperations: {
			"users:view", "users:suspend",
			"draws:manage", "draws:execute",
			"notifications:send",
			"wars:manage",
			"dashboard:view", "fraud:view", "fraud:resolve",
		},
		RoleContent: {
			"prizes:manage", "studio:manage",
			"config:manage", "passport:view",
			"dashboard:view",
		},
	}
	for _, perm := range permissions[c.Role] {
		if perm == action {
			return true
		}
	}
	return false
}
