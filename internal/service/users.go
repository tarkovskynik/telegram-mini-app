package service

import (
	"context"
	"fmt"

	"UD_telegram_miniapp/internal/model"
)

type UserService struct {
	repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
	return &UserService{
		repo: repo,
	}
}

func (s *UserService) RegisterUser(ctx context.Context, user *model.User) error {
	err := s.repo.CreateUser(ctx, user)
	if err != nil {
		return err
	}

	return nil
}

func (s *UserService) GetUserByTelegramID(ctx context.Context, telegramID int64) (*model.User, error) {
	user, err := s.repo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by telegram ID: %w", err)
	}
	return user, nil
}

func (s *UserService) UpdateUserPoints(ctx context.Context, telegramID int64, points int) error {
	err := s.repo.UpdateUserPoints(ctx, telegramID, points)
	if err != nil {
		return fmt.Errorf("failed to update user points: %w", err)
	}
	return nil
}

func (s *UserService) UpdateUserWaitlistStatus(ctx context.Context, telegramID int64, status bool) error {
	err := s.repo.UpdateUserWaitlistStatus(ctx, telegramID, status)
	if err != nil {
		return fmt.Errorf("failed to update waitlist status: %w", err)
	}
	return nil
}

func (s *UserService) GetUserWaitlistStatus(ctx context.Context, telegramID int64) (*bool, error) {
	status, err := s.repo.GetUserWaitlistStatus(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("failed to get waitlist status: %w", err)
	}
	return status, nil
}

func (s *UserService) GetLeaderboard(ctx context.Context) ([]*model.User, error) {
	users, err := s.repo.GetTopUsers(ctx, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to get top users: %w", err)
	}
	return users, nil
}
