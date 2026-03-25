package persistence

import (
	"context"
	"database/sql"
	"loyalty-nexus/internal/domain/repositories"
)

type PostgresHLRRepository struct {
	db *sql.DB
}

func NewPostgresHLRRepository(db *sql.DB) *PostgresHLRRepository {
	return &PostgresHLRRepository{db: db}
}

func (r *PostgresHLRRepository) FindByMSISDN(ctx context.Context, msisdn string) (*repositories.NetworkCache, error) {
	var c repositories.NetworkCache
	query := "SELECT msisdn, network, lookup_source, is_valid FROM network_cache WHERE msisdn = $1"
	err := r.db.QueryRowContext(ctx, query, msisdn).Scan(&c.MSISDN, &c.Network, &c.LookupSource, &c.IsValid)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *PostgresHLRRepository) Save(ctx context.Context, cache *repositories.NetworkCache) error {
	query := `
		INSERT INTO network_cache (msisdn, network, lookup_source, is_valid, last_verified, cache_expires)
		VALUES ($1, $2, $3, $4, now(), now() + interval '60 days')
		ON CONFLICT (msisdn) DO UPDATE SET network = $2, lookup_source = $3, is_valid = $4, last_verified = now(), cache_expires = now() + interval '60 days'
	`
	_, err := r.db.ExecContext(ctx, query, cache.MSISDN, cache.Network, cache.LookupSource, cache.IsValid)
	return err
}

func (r *PostgresHLRRepository) Invalidate(ctx context.Context, msisdn string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE network_cache SET is_valid = false WHERE msisdn = $1", msisdn)
	return err
}
