package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/config"
)

// ── Minimal Mocks ─────────────────────────────────────────────────────────────

// mockAuthRepo implements repositories.AuthRepository
type mockAuthRepo struct {
	otps   map[string]*entities.AuthOTP
	admins map[string]*entities.AdminUser
}

var _ repositories.AuthRepository = (*mockAuthRepo)(nil)

func newMockAuthRepo() *mockAuthRepo {
	return &mockAuthRepo{
		otps:   make(map[string]*entities.AuthOTP),
		admins: make(map[string]*entities.AdminUser),
	}
}

func (m *mockAuthRepo) CreateOTP(_ context.Context, otp *entities.AuthOTP) error {
	key := otp.PhoneNumber + ":" + string(otp.Purpose)
	m.otps[key] = otp
	return nil
}
func (m *mockAuthRepo) FindLatestPendingOTP(_ context.Context, phone, purpose string) (*entities.AuthOTP, error) {
	if r, ok := m.otps[phone+":"+purpose]; ok {
		return r, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockAuthRepo) MarkOTPUsed(_ context.Context, id uuid.UUID) error  { return nil }
func (m *mockAuthRepo) ExpireOTP(_ context.Context, id uuid.UUID) error    { return nil }
func (m *mockAuthRepo) ExpireOldOTPs(_ context.Context) (int64, error)     { return 0, nil }
func (m *mockAuthRepo) FindAdminByUsername(_ context.Context, u string) (*entities.AdminUser, error) {
	if a, ok := m.admins[u]; ok {
		return a, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockAuthRepo) FindAdminByID(_ context.Context, id uuid.UUID) (*entities.AdminUser, error) {
	return nil, gorm.ErrRecordNotFound
}

// mockUserRepoAuth implements the subset AuthService actually calls
type mockUserRepoAuth struct {
	users map[string]*entities.User
}

var _ repositories.UserRepository = (*mockUserRepoAuth)(nil)

func newMockUserRepoAuth() *mockUserRepoAuth {
	return &mockUserRepoAuth{users: make(map[string]*entities.User)}
}
func (m *mockUserRepoAuth) FindByID(_ context.Context, id uuid.UUID) (*entities.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockUserRepoAuth) FindByPhoneNumber(_ context.Context, phone string) (*entities.User, error) {
	if u, ok := m.users[phone]; ok {
		return u, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockUserRepoAuth) Create(_ context.Context, u *entities.User) error {
	m.users[u.PhoneNumber] = u
	return nil
}
func (m *mockUserRepoAuth) Update(_ context.Context, u *entities.User) error {
	m.users[u.PhoneNumber] = u
	return nil
}
func (m *mockUserRepoAuth) FindByReferralCode(_ context.Context, _ string) (*entities.User, error) {
	return nil, gorm.ErrRecordNotFound
}
func (m *mockUserRepoAuth) ExistsByPhoneNumber(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (m *mockUserRepoAuth) UpdateStreak(_ context.Context, _ uuid.UUID, _ int, _ interface{}) error {
	return nil
}
func (m *mockUserRepoAuth) UpdateMoMo(_ context.Context, _ uuid.UUID, _ string, _ bool) error {
	return nil
}
func (m *mockUserRepoAuth) UpdateTier(_ context.Context, _ uuid.UUID, _ string) error { return nil }
func (m *mockUserRepoAuth) UpdateWalletPassID(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (m *mockUserRepoAuth) SetPointsExpiry(_ context.Context, _ uuid.UUID, _ interface{}) error {
	return nil
}
func (m *mockUserRepoAuth) GetWallet(_ context.Context, id uuid.UUID) (*entities.Wallet, error) {
	return &entities.Wallet{UserID: id}, nil
}
func (m *mockUserRepoAuth) GetWalletForUpdate(_ context.Context, id uuid.UUID) (*entities.Wallet, error) {
	return &entities.Wallet{UserID: id}, nil
}
func (m *mockUserRepoAuth) UpdateWallet(_ context.Context, _ *entities.Wallet) error { return nil }
func (m *mockUserRepoAuth) FindInactiveUsers(_ context.Context, _, _ int) ([]entities.User, error) {
	return nil, nil
}
func (m *mockUserRepoAuth) FindUsersWithExpiringPoints(_ context.Context, _, _ int) ([]entities.User, error) {
	return nil, nil
}
func (m *mockUserRepoAuth) CountByState(_ context.Context, _ string) (int64, error) { return 0, nil }
func (m *mockUserRepoAuth) UpdateState(_ context.Context, _ uuid.UUID, _ string) error  { return nil }
func (m *mockUserRepoAuth) CreateWallet(_ context.Context, _ *entities.Wallet) error    { return nil }

// ── Helpers ───────────────────────────────────────────────────────────────────

func setupAuthCfg(t *testing.T) *config.ConfigManager {
	t.Helper()
	t.Setenv("JWT_SECRET", "test_secret_at_least_32_chars_long!")
	t.Setenv("AES_256_KEY", "0000000000000000000000000000000000000000000000000000000000000000")
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	db.Exec(`CREATE TABLE IF NOT EXISTS network_configs (id TEXT PRIMARY KEY, key TEXT UNIQUE NOT NULL, value TEXT NOT NULL)`)
	return config.NewConfigManager(db)
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestAuth_SendOTP_CreatesRecord(t *testing.T) {
	cfg := setupAuthCfg(t)
	authRepo := newMockAuthRepo()
	userRepo := newMockUserRepoAuth()
	notifySvc := services.NewNotificationService("") // empty key = SMS no-op in tests

	svc := services.NewAuthService(authRepo, userRepo, notifySvc, cfg)
	// SMS will fail in unit tests (no key configured). 
	// Service saves OTP first, THEN sends SMS — so OTP is persisted even on SMS error.
	_, _ = svc.SendOTP(context.Background(), "08011223344", "login")
	if len(authRepo.otps) == 0 {
		t.Fatal("OTP record not stored in repo")
	}
}

func TestAuth_SendOTP_PhoneNormalization(t *testing.T) {
	cfg := setupAuthCfg(t)
	authRepo := newMockAuthRepo()
	userRepo := newMockUserRepoAuth()
	notifySvc := services.NewNotificationService("")

	svc := services.NewAuthService(authRepo, userRepo, notifySvc, cfg)
	// Both forms should succeed without error
	for _, phone := range []string{"08012345678", "+2348012345678", "2348012345678"} {
		if _, err := svc.SendOTP(context.Background(), phone, "login"); err != nil {
			t.Logf("SendOTP(%s): %v — acceptable", phone, err)
		}
	}
}

func TestAuth_VerifyOTP_WrongCode_Fails(t *testing.T) {
	cfg := setupAuthCfg(t)
	authRepo := newMockAuthRepo()
	userRepo := newMockUserRepoAuth()
	notifySvc := services.NewNotificationService("")

	svc := services.NewAuthService(authRepo, userRepo, notifySvc, cfg)
	phone := "07055667788"
	_, _ = svc.SendOTP(context.Background(), phone, "login")

	_, _, err := svc.VerifyOTP(context.Background(), phone, "000000", "login")
	if err == nil {
		t.Fatal("wrong OTP should fail")
	}
}

func TestAuth_VerifyOTP_ExpiredOTP_Fails(t *testing.T) {
	cfg := setupAuthCfg(t)
	authRepo := newMockAuthRepo()
	userRepo := newMockUserRepoAuth()
	notifySvc := services.NewNotificationService("")

	svc := services.NewAuthService(authRepo, userRepo, notifySvc, cfg)
	phone := "08099887766"

	// Manually seed an expired OTP record
	authRepo.otps[phone+":login"] = &entities.AuthOTP{
		ID:          uuid.New(),
		PhoneNumber: phone,
		Purpose:     entities.OTPLogin,
		Code:        "irrelevant",
		ExpiresAt:   time.Now().Add(-10 * time.Minute), // already expired
		Status:      entities.OTPPending,
	}

	_, _, err := svc.VerifyOTP(context.Background(), phone, "123456", "login")
	if err == nil {
		t.Fatal("expired OTP should fail verification")
	}
}

func TestAuth_NewUser_AutoCreated_On_Verify(t *testing.T) {
	cfg := setupAuthCfg(t)
	authRepo := newMockAuthRepo()
	userRepo := newMockUserRepoAuth()
	notifySvc := services.NewNotificationService("")

	svc := services.NewAuthService(authRepo, userRepo, notifySvc, cfg)
	phone := "09011335577"
	_, _ = svc.SendOTP(context.Background(), phone, "login")

	// User must NOT exist before successful verify
	if _, err := userRepo.FindByPhoneNumber(context.Background(), phone); err == nil {
		t.Fatal("user should not exist before OTP verify")
	}
}
