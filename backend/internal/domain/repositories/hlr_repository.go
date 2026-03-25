package repositories

import (
	"context"
	"loyalty-nexus/internal/domain/entities"
)

type NetworkCache struct {
	MSISDN       string
	Network      string
	LookupSource string
	IsValid      bool
}

type HLRRepository interface {
	FindByMSISDN(ctx context.Context, msisdn string) (*NetworkCache, error)
	Save(ctx context.Context, cache *NetworkCache) error
	Invalidate(ctx context.Context, msisdn string) error
}
