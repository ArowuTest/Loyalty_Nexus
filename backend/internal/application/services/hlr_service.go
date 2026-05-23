package services

import (
	"context"
	"loyalty-nexus/internal/domain/repositories"
)

// HLRService provides network lookup from cached recharge history only.
// Prefix-based lookup was intentionally removed: Nigerian Number Portability (NNP)
// means prefix → operator mapping is unreliable. Use VTURechargeService.DetectNetworkSmart
// for authoritative 3-tier detection (cache → VTPass merchant-verify → user selection).
type HLRService struct {
	repo repositories.HLRRepository
}

func NewHLRService(repo repositories.HLRRepository) *HLRService {
	return &HLRService{repo: repo}
}

// GetNetwork returns the network from the local cache only.
// Returns ("", nil) on cache miss — callers must handle the empty string case.
func (s *HLRService) GetNetwork(ctx context.Context, phone string) (string, error) {
	cached, err := s.repo.GetCached(ctx, phone)
	if err == nil && cached != nil {
		return cached.Network, nil
	}
	return "", nil
}
