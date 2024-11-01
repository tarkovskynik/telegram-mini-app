package service

import (
	"context"
	"errors"

	"UD_telegram_miniapp/internal/model"

	"github.com/google/uuid"
)

var (
	ErrClaimNotAvailable = errors.New("the required time has not yet passed since your last claim")
	ErrUserNotFound      = errors.New("user not found")

	ErrQuestNotFound       = errors.New("quest not found")
	ErrQuestNotStarted     = errors.New("quest has not been started")
	ErrQuestAlreadyClaimed = errors.New("quest already claimed")

	ErrValidationNameExists    = errors.New("validation name already exists")
	ErrValidationNotFound      = errors.New("validation not found")
	ErrValidationAlreadyExists = errors.New("validation already exists for quest")
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

type SocialQuestServiceI interface {
	GetUserQuests(ctx context.Context, telegramID int64) (
		[]*model.SocialQuest,
		[]*model.UserSocialQuest,
		model.UserValidationsStatus,
		error)
	GetQuestByID(ctx context.Context, telegramID int64, questID uuid.UUID) (
		*model.SocialQuest,
		*model.UserSocialQuest,
		model.UserValidationsStatus,
		error)
	ClaimQuest(ctx context.Context, telegramID int64, questID uuid.UUID) error
	CreateSocialQuest(ctx context.Context, quest *model.SocialQuest) (uuid.UUID, error)
	CreateValidationKind(ctx context.Context, validation *model.QuestValidationKind) error
	ListValidationKinds(ctx context.Context) ([]*model.QuestValidationKind, error)
	AddQuestValidation(ctx context.Context, questID uuid.UUID, validationID int) error
	RemoveQuestValidation(ctx context.Context, questID uuid.UUID, validationID int) error
}

//go:generate mockery --name DailyQuestRepository --output ./mocks --structname MockDailyQuestRepository
type DailyQuestRepository interface {
	GetDailyQuestStatus(ctx context.Context, telegramID int64) (*model.DailyQuest, error)
	UpdateDailyQuestStatus(ctx context.Context, quest *model.DailyQuest) error
	UpdateUserPoints(ctx context.Context, telegramID int64, points int) error
}

type SocialQuestRepository interface {
	GetQuestsData(ctx context.Context, telegramID int64) ([]*model.SocialQuest, []*model.UserSocialQuest, error)
	GetUserValidationsStatus(ctx context.Context, telegramID int64) (model.UserValidationsStatus, error)
	GetQuestDataByID(ctx context.Context, telegramID int64, questID uuid.UUID) (*model.SocialQuest, *model.UserSocialQuest, error)
	ClaimQuest(ctx context.Context, telegramID int64, questID uuid.UUID) error
	CreateSocialQuest(ctx context.Context, quest *model.SocialQuest) error
	CreateValidationKind(ctx context.Context, validation *model.QuestValidationKind) error
	ListValidationKinds(ctx context.Context) ([]*model.QuestValidationKind, error)
	RemoveQuestValidation(ctx context.Context, questID uuid.UUID, validationID int) error
	AddQuestValidation(ctx context.Context, questID uuid.UUID, validationID int) error
}
