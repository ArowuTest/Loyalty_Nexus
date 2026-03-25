package services

import (
	"context"
	"fmt"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/infrastructure/external"
)

type PassportService struct {
	walletAdapter external.WalletPassport
	userRepo      UserRepository // assuming this is available
}

func NewPassportService(wa external.WalletPassport) *PassportService {
	return &PassportService{walletAdapter: wa}
}

func (s *PassportService) GetIssuanceURLs(ctx context.Context, userID string, points int64) (map[string]string, error) {
	apple, _ := s.walletAdapter.IssueApplePass(ctx, userID, points)
	google, _ := s.walletAdapter.IssueGooglePass(ctx, userID, points)

	return map[string]string{
		"apple":  apple,
		"google": google,
	}, nil
}

func (s *PassportService) SyncWallet(ctx context.Context, userID string, points int64) error {
	return s.walletAdapter.PushUpdate(ctx, userID, points)
}
