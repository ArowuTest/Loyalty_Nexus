package services

import (
	"context"
	"loyalty-nexus/internal/domain/repositories"
)

type HLRService struct {
	repo repositories.HLRRepository
}

func NewHLRService(repo repositories.HLRRepository) *HLRService {
	return &HLRService{repo: repo}
}

func (s *HLRService) GetNetwork(ctx context.Context, phone string) (string, error) {
	cached, err := s.repo.GetCached(ctx, phone)
	if err == nil && cached != nil {
		return cached.Network, nil
	}
	// Prefix-based fallback (no external API call cost)
	return prefixLookup(phone), nil
}

// prefixLookup uses Nigerian MNO number prefixes.
func prefixLookup(phone string) string {
	prefixes := map[string]string{
		"0803": "MTN", "0806": "MTN", "0703": "MTN", "0706": "MTN",
		"0813": "MTN", "0816": "MTN", "0810": "MTN", "0814": "MTN", "0903": "MTN", "0906": "MTN",
		"0802": "AIRTEL", "0808": "AIRTEL", "0708": "AIRTEL", "0812": "AIRTEL",
		"0701": "AIRTEL", "0901": "AIRTEL", "0902": "AIRTEL", "0904": "AIRTEL", "0907": "AIRTEL",
		"0805": "GLO", "0807": "GLO", "0705": "GLO", "0815": "GLO", "0905": "GLO",
		"0811": "9MOBILE", "0809": "9MOBILE", "0818": "9MOBILE", "0817": "9MOBILE",
	}
	prefix := phone
	if len(phone) >= 11 {
		prefix = phone[:4]
	}
	if len(phone) >= 13 { // 234XXXXXXXXXX
		prefix = "0" + phone[3:7]
	}
	if network, ok := prefixes[prefix]; ok {
		return network
	}
	return "MTN" // Default for Nigeria
}
