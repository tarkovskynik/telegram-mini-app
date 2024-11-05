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

func (s *SocialQuestService) CreateReferralQuest(ctx context.Context, quest *model.ReferralQuest) (uuid.UUID, error) {
	questID, err := s.repo.CreateReferralQuest(ctx, quest)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create referral quest: %w", err)
	}

	return questID, nil
}

func (s *SocialQuestService) GetUserQuestStatuses(ctx context.Context, telegramID int64) ([]*model.ReferralQuestStatus, error) {
	statuses, err := s.repo.GetUserReferralQuestStatuses(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("failed to get quest statuses: %w", err)
	}

	if len(statuses) == 0 {
		return []*model.ReferralQuestStatus{}, nil
	}

	for _, status := range statuses {
		status.ReadyToClaim = !status.Completed && status.CurrentReferrals >= status.ReferralsRequired
	}

	return statuses, nil
}

func (s *SocialQuestService) GetReferralQuestStatus(ctx context.Context, telegramID int64, questID uuid.UUID) (*model.ReferralQuestStatus, error) {
	status, err := s.repo.GetSingleQuestStatus(ctx, telegramID, questID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrQuestNotFound
		}
		return nil, fmt.Errorf("failed to get quest status: %w", err)
	}

	status.ReadyToClaim = !status.Completed && status.CurrentReferrals >= status.ReferralsRequired

	return status, nil
}

func (s *SocialQuestService) ClaimReferralQuest(ctx context.Context, telegramID int64, questID uuid.UUID) error {
	status, err := s.GetReferralQuestStatus(ctx, telegramID, questID)
	if err != nil {
		return fmt.Errorf("failed to get quest status: %w", err)
	}

	if !status.ReadyToClaim {
		if status.Completed {
			return ErrQuestAlreadyClaimed
		}
		return ErrNotEnoughReferrals
	}

	err = s.repo.ClaimReferralQuest(ctx, telegramID, questID, status.PointReward)
	if err != nil {
		return fmt.Errorf("failed to claim quest: %w", err)
	}

	return nil
}
