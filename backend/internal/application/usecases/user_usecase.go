package usecases

import (
	"context"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
)

type UserUseCase struct {
	repo repositories.UserRepository
}

func NewUserUseCase(r repositories.UserRepository) *UserUseCase {
	return &UserUseCase{repo: r}
}

func (u *UserUseCase) GetProfile(ctx context.Context, msisdn string) (*entities.User, error) {
	return u.repo.FindByMSISDN(ctx, msisdn)
}
