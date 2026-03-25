package services

import (
	"context"
	"fmt"
	"loyalty-nexus/internal/domain/repositories"
)

type HLRService struct {
	repo repositories.HLRRepository
	// In production: hlrClient external.HLRClient
}

func NewHLRService(r repositories.HLRRepository) *HLRService {
	return &HLRService{repo: r}
}

func (s *HLRService) DetectNetwork(ctx context.Context, msisdn string, userSelection *string) (string, error) {
	// 1. Try Trusted Cache
	cache, _ := s.repo.FindByMSISDN(ctx, msisdn)
	if cache != nil && cache.IsValid && (cache.LookupSource == "hlr_api" || cache.LookupSource == "user_selection") {
		return cache.Network, nil
	}

	// 2. Try HLR API (Primary Source)
	// mock success
	network := "MTN"
	s.repo.Save(ctx, &repositories.NetworkCache{
		MSISDN: msisdn,
		Network: network,
		LookupSource: "hlr_api",
		IsValid: true,
	})
	return network, nil
}
