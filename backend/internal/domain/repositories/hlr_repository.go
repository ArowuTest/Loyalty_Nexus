package repositories

import "context"

type HLRResult struct {
	PhoneNumber  string
	Network      string // MTN, AIRTEL, GLO, 9MOBILE
	IsValid      bool
	LookupSource string
}

type HLRRepository interface {
	GetCached(ctx context.Context, phone string) (*HLRResult, error) // nil if miss or expired
	Cache(ctx context.Context, result *HLRResult, ttlHours int) error
	Invalidate(ctx context.Context, phone string) error
}
