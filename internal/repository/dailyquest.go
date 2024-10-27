package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"UD_telegram_miniapp/internal/model"

	"github.com/Masterminds/squirrel"
)

type DailyQuest struct {
	UserTelegramID         int64      `db:"user_telegram_id"`
	LastClaimedAt          *time.Time `db:"last_claimed_at"`
	ConsecutiveDaysClaimed int        `db:"consecutive_days_claimed"`
}

func (r *Repository) GetDailyQuestStatus(ctx context.Context, telegramID int64) (*model.DailyQuest, error) {
	var quest DailyQuest

	if _, err := r.GetUserByTelegramID(ctx, telegramID); err != nil {
		return nil, err
	}

	query, args, err := squirrel.
		Select("user_telegram_id", "last_claimed_at", "consecutive_days_claimed").
		From("daily_quests").
		Where(squirrel.Eq{"user_telegram_id": telegramID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	err = r.db.Get(&quest, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &model.DailyQuest{
		UserTelegramID:         quest.UserTelegramID,
		LastClaimedAt:          quest.LastClaimedAt,
		ConsecutiveDaysClaimed: quest.ConsecutiveDaysClaimed,
	}, nil
}

func (r *Repository) UpdateDailyQuestStatus(ctx context.Context, quest *model.DailyQuest) error {
	query, args, err := squirrel.
		Update("daily_quests").
		SetMap(map[string]interface{}{
			"last_claimed_at":          quest.LastClaimedAt,
			"consecutive_days_claimed": quest.ConsecutiveDaysClaimed,
		}).
		Where(squirrel.Eq{"user_telegram_id": quest.UserTelegramID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}
