package services

import (
	"context"
	"strings"
)

type NetworkInfo struct {
	Network  string
	Prefix   string
	IsMobile bool
}

type HLRService struct{}

func NewHLRService() *HLRService {
	return &HLRService{}
}

func (s *HLRService) DetectNetwork(ctx context.Context, msisdn string) (*NetworkInfo, error) {
	// Normalize MSISDN (Remove 234 prefix if present)
	normalized := strings.TrimPrefix(msisdn, "234")
	if len(normalized) > 3 {
		prefix := normalized[:3]
		switch prefix {
		case "803", "806", "810", "813", "814", "816", "903", "906", "913", "916":
			return &NetworkInfo{Network: "MTN", Prefix: prefix, IsMobile: true}, nil
		case "805", "807", "811", "815", "905", "915":
			return &NetworkInfo{Network: "GLO", Prefix: prefix, IsMobile: true}, nil
		case "802", "808", "812", "902", "904", "907", "912":
			return &NetworkInfo{Network: "AIRTEL", Prefix: prefix, IsMobile: true}, nil
		case "809", "817", "818", "909", "908":
			return &NetworkInfo{Network: "9MOBILE", Prefix: prefix, IsMobile: true}, nil
		}
	}
	return &NetworkInfo{Network: "UNKNOWN", IsMobile: false}, nil
}
