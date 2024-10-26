package service

import (
	"context"
	"errors"

	"UD_telegram_miniapp/internal/model"
)

var (
	ErrClaimNotAvailable = errors.New("the required time has not yet passed since your last claim")
	ErrUserNotFound      = errors.New("user not found")
)

type Service struct {
	*UserService
	*DailyQuestService
}

func NewService(userService *UserService, dailyQuestService *DailyQuestService) *Service {
	return &Service{
		UserService:       userService,
		DailyQuestService: dailyQuestService,
	}
}

type UserServiceI interface {
	RegisterUser(ctx context.Context, user *model.User) error
	GetUserByTelegramID(ctx context.Context, telegramID int64) (*model.User, error)
	UpdateUserPoints(ctx context.Context, telegramID int64, points int) error
	UpdateUserWaitlistStatus(ctx context.Context, telegramID int64, status bool) error
	GetUserWaitlistStatus(ctx context.Context, telegramID int64) (*bool, error)
	GetLeaderboard(ctx context.Context) ([]*model.User, error)
}

type UserRepository interface {
	CreateUser(ctx context.Context, user *model.User) error
	GetUserByTelegramID(ctx context.Context, telegramID int64) (*model.User, error)
	UpdateUserPoints(ctx context.Context, telegramID int64, points int) error
	UpdateUserWaitlistStatus(ctx context.Context, telegramID int64, status bool) error
	GetUserWaitlistStatus(ctx context.Context, telegramID int64) (*bool, error)
	GetTopUsers(ctx context.Context, limit int) ([]*model.User, error)
}

type DailyQuestServiceI interface {
	GetStatus(ctx context.Context, telegramID int64) (*model.DailyQuest, error)
	Claim(ctx context.Context, telegramID int64) error
}

type DailyQuestRepository interface {
	GetDailyQuestStatus(ctx context.Context, telegramID int64) (*model.DailyQuest, error)
	UpdateDailyQuestStatus(ctx context.Context, quest *model.DailyQuest) error
	UpdateUserPoints(ctx context.Context, telegramID int64, points int) error
}
