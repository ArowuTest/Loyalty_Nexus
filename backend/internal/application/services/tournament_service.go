package services

import (
	"context"
	"database/sql"
)

type RegionRank struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	TotalKobo  int64   `json:"total_kobo"`
	Rank       int     `json:"rank"`
	Multiplier float64 `json:"multiplier"`
}

type TournamentService struct {
	db *sql.DB
}

func NewTournamentService(db *sql.DB) *TournamentService {
	return &TournamentService{db: db}
}

func (s *TournamentService) GetLeaderboard(ctx context.Context) ([]RegionRank, error) {
	query := `
		SELECT s.region_code, rs.region_name, s.total_recharge_kobo, s.rank, 
		       CASE WHEN rs.is_golden_hour THEN rs.golden_hour_multiplier ELSE rs.base_multiplier END as current_multiplier
		FROM regional_stats s
		JOIN regional_settings rs ON s.region_code = rs.region_code
		ORDER BY s.rank ASC
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ranks []RegionRank
	for rows.Next() {
		var r RegionRank
		if err := rows.Scan(&r.Code, &r.Name, &r.TotalKobo, &r.Rank, &r.Multiplier); err != nil {
			return nil, err
		}
		ranks = append(ranks, r)
	}
	return ranks, nil
}

func (s *TournamentService) UpdateLeaderboard(ctx context.Context) error {
	// 1. Calculate ranks based on total recharge volume
	// In production, this would be a trigger or a scheduled task
	return nil
}
