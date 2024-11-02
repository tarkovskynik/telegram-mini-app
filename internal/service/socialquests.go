package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"UD_telegram_miniapp/internal/model"
	"UD_telegram_miniapp/internal/repository"

	"github.com/google/uuid"
)

type SocialQuestService struct {
	repo SocialQuestRepository
}

func NewSocialQuestService(repo SocialQuestRepository) *SocialQuestService {
	return &SocialQuestService{
		repo: repo,
	}
}

func (s *SocialQuestService) GetUserQuests(ctx context.Context, telegramID int64) (
	[]*model.SocialQuest,
	[]*model.UserSocialQuest,
	model.UserValidationsStatus,
	error) {

	quests, userQuests, err := s.repo.GetQuestsData(ctx, telegramID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get quests data: %w", err)
	}

	validationStatus, err := s.repo.GetUserValidationsStatus(ctx, telegramID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get validation status: %w", err)
	}

	return quests, userQuests, validationStatus, nil
}

func (s *SocialQuestService) GetQuestByID(ctx context.Context, telegramID int64, questID uuid.UUID) (
	*model.SocialQuest,
	*model.UserSocialQuest,
	model.UserValidationsStatus,
	error) {

	quest, userQuest, err := s.repo.GetQuestDataByID(ctx, telegramID, questID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, nil, ErrQuestNotFound
		}
		return nil, nil, nil, fmt.Errorf("failed to get quest data: %w", err)
	}

	validationStatus, err := s.repo.GetUserValidationsStatus(ctx, telegramID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get validation status: %w", err)
	}

	return quest, userQuest, validationStatus, nil
}

func (s *SocialQuestService) ClaimQuest(ctx context.Context, telegramID int64, questID uuid.UUID) error {
	sq, _, _, err := s.GetQuestByID(ctx, telegramID, questID)
	if err != nil {
		return err
	}

	err = s.repo.ClaimQuest(ctx, telegramID, questID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return ErrQuestNotFound
		case errors.Is(err, repository.ErrQuestNotStarted):
			return ErrQuestNotStarted
		case errors.Is(err, repository.ErrQuestAlreadyClaimed):
			return ErrQuestAlreadyClaimed
		default:
			return fmt.Errorf("failed to claim quest: %w", err)
		}
	}

	err = s.repo.UpdateUserPoints(ctx, telegramID, sq.PointReward)
	if err != nil {
		return fmt.Errorf("failed to update user points: %w", err)
	}

	return nil
}

func (s *SocialQuestService) CreateSocialQuest(ctx context.Context, quest *model.SocialQuest) (uuid.UUID, error) {
	if err := s.repo.CreateSocialQuest(ctx, quest); err != nil {
		return uuid.Nil, fmt.Errorf("failed to create social quest: %w", err)
	}

	return quest.QuestID, nil
}

func (s *SocialQuestService) CreateValidationKind(ctx context.Context, validation *model.QuestValidationKind) error {
	if validation.ValidationName == "" {
		return fmt.Errorf("validation name is required")
	}

	validation.ValidationName = strings.ToUpper(validation.ValidationName)
	return s.repo.CreateValidationKind(ctx, validation)
}

func (s *SocialQuestService) ListValidationKinds(ctx context.Context) ([]*model.QuestValidationKind, error) {
	return s.repo.ListValidationKinds(ctx)
}

func (s *SocialQuestService) AddQuestValidation(ctx context.Context, questID uuid.UUID, validationID int) error {
	err := s.repo.AddQuestValidation(ctx, questID, validationID)
	if err != nil {
		return fmt.Errorf("failed to add quest validation: %w", err)
	}
	return nil
}

func (s *SocialQuestService) RemoveQuestValidation(ctx context.Context, questID uuid.UUID, validationID int) error {
	err := s.repo.RemoveQuestValidation(ctx, questID, validationID)
	if err != nil {
		return fmt.Errorf("failed to remove quest validation: %w", err)
	}
	return nil
}
