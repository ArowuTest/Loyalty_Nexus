package services

import (
	"context"
	"fmt"
	"log"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReferralService struct {
	userRepo repositories.UserRepository
	db       *gorm.DB
}

func NewReferralService(ur repositories.UserRepository, db *gorm.DB) *ReferralService {
	return &ReferralService{userRepo: ur, db: db}
}

// ProcessReferral links a new user to a referrer (REQ-5.2.10)
func (s *ReferralService) ProcessReferral(ctx context.Context, referredUserID uuid.UUID, referralCode string) error {
	var referrer entities.User
	if err := s.db.WithContext(ctx).Where("user_code = ?", referralCode).First(&referrer).Error; err != nil {
		return fmt.Errorf("invalid referral code")
	}

	if referrer.ID == referredUserID {
		return fmt.Errorf("self-referral not allowed")
	}

	return s.db.WithContext(ctx).Model(&entities.User{}).
		Where("id = ?", referredUserID).
		Update("referred_by_id", referrer.ID).Error
}
