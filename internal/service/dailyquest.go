package service

import (
	"context"
	"errors"
	"time"

	"UD_telegram_miniapp/internal/model"
	"UD_telegram_miniapp/internal/repository"
)

const (
	BaseReward = 500
)

var DailyBonuses = []int{0, 140, 280, 400, 500, 600, 700}

type DailyQuestService struct {
	repo DailyQuestRepository
}

func NewDailyQuestService(repo DailyQuestRepository) *DailyQuestService {
	return &DailyQuestService{
		repo: repo,
	}
}

func (s *DailyQuestService) GetStatus(ctx context.Context, telegramID int64) (*model.DailyQuest, error) {
	quest, err := s.repo.GetDailyQuestStatus(ctx, telegramID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	now := time.Now().UTC()
	hasNeverBeenClaimed := quest.LastClaimedAt == nil

	status := &model.DailyQuest{
		UserTelegramID:         telegramID,
		LastClaimedAt:          quest.LastClaimedAt,
		ConsecutiveDaysClaimed: quest.ConsecutiveDaysClaimed,
		HasNeverBeenClaimed:    hasNeverBeenClaimed,
		DailyRewards:           make([]model.DayReward, 7),
	}

	if hasNeverBeenClaimed {
		status.NextClaimAvailable = nil
		status.IsAvailable = true
		status.ConsecutiveDaysClaimed = 0
	} else {
		nextClaimAvailable := quest.LastClaimedAt.Add(24 * time.Hour)
		status.NextClaimAvailable = &nextClaimAvailable
		status.IsAvailable = now.After(*status.NextClaimAvailable)

		if now.After(nextClaimAvailable.Add(24 * time.Hour)) {
			status.ConsecutiveDaysClaimed = 0

			err = s.repo.UpdateDailyQuestStatus(ctx, &model.DailyQuest{
				UserTelegramID:         telegramID,
				LastClaimedAt:          quest.LastClaimedAt,
				ConsecutiveDaysClaimed: 0,
			})
			if err != nil {
				return nil, err
			}
		}
	}

	for i := 0; i < 7; i++ {
		status.DailyRewards[i] = model.DayReward{
			Day:    i + 1,
			Reward: BaseReward + DailyBonuses[i],
		}
	}

	return status, nil
}

func (s *DailyQuestService) Claim(ctx context.Context, telegramID int64) error {
	status, err := s.GetStatus(ctx, telegramID)
	if err != nil {
		return err
	}

	if !status.IsAvailable {
		return ErrClaimNotAvailable
	}

	now := time.Now().UTC()
	newConsecutiveDays := status.ConsecutiveDaysClaimed + 1
	if newConsecutiveDays > 7 {
		newConsecutiveDays = 1
	}

	reward := BaseReward + DailyBonuses[newConsecutiveDays-1]

	err = s.repo.UpdateDailyQuestStatus(ctx, &model.DailyQuest{
		UserTelegramID:         telegramID,
		LastClaimedAt:          &now,
		ConsecutiveDaysClaimed: newConsecutiveDays,
	})
	if err != nil {
		return err
	}

	err = s.repo.UpdateUserPoints(ctx, telegramID, reward)
	if err != nil {
		return err
	}

	return nil
}
