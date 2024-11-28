package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
)

type FarmStatus struct {
	IsInProgress      bool
	StartedAt         sql.NullTime
	PointReward       int
	IsPreviousClaimed bool
}

const CooldownDuration = 8 * time.Hour
const DefaultReward = 800

func (r *Repository) StartHarvest(playerID int64) error {
	var status FarmStatus
	selectQuery, selectArgs, err := squirrel.Select("is_in_progress", "started_at", "is_previous_claimed").
		From("farm_game").
		Where(squirrel.Eq{"player": playerID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build select query: %w", err)
	}

	err = r.db.QueryRowContext(context.TODO(), selectQuery, selectArgs...).
		Scan(&status.IsInProgress, &status.StartedAt, &status.IsPreviousClaimed)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			startTime := time.Now().UTC()
			insertQuery, insertArgs, err := squirrel.Insert("farm_game").
				Columns("player", "is_in_progress", "started_at", "is_previous_claimed").
				Values(playerID, true, startTime, false).
				PlaceholderFormat(squirrel.Dollar).
				ToSql()
			if err != nil {
				return fmt.Errorf("failed to build insert query: %w", err)
			}

			_, err = r.db.ExecContext(context.TODO(), insertQuery, insertArgs...)
			if err != nil {
				return fmt.Errorf("failed to start harvest: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get farm status: %w", err)
	}

	if status.IsInProgress {
		return fmt.Errorf("cannot start harvest: farming already in progress")
	}

	if !status.IsPreviousClaimed {
		return fmt.Errorf("cannot start harvest: previous reward must be claimed first")
	}

	startTime := time.Now().UTC()
	insertQuery, insertArgs, err := squirrel.Insert("farm_game").
		Columns("player", "is_in_progress", "started_at", "is_previous_claimed").
		Values(playerID, true, startTime, true).
		Suffix("ON CONFLICT (player) DO UPDATE SET is_in_progress = EXCLUDED.is_in_progress, started_at = EXCLUDED.started_at").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	_, err = r.db.ExecContext(context.TODO(), insertQuery, insertArgs...)
	if err != nil {
		return fmt.Errorf("failed to start harvest: %w", err)
	}

	return nil
}

func (r *Repository) Status(player int64) (FarmStatus, error) {
	var status FarmStatus

	selectQuery, selectArgs, err := squirrel.Select(
		"is_in_progress",
		"started_at",
		"is_previous_claimed",
	).
		From("farm_game").
		Where(squirrel.Eq{"player": player}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return status, fmt.Errorf("failed to build select query: %w", err)
	}

	err = r.db.QueryRowContext(context.Background(), selectQuery, selectArgs...).
		Scan(&status.IsInProgress, &status.StartedAt, &status.IsPreviousClaimed)
	if errors.Is(err, sql.ErrNoRows) {
		return FarmStatus{
			IsInProgress:      false,
			StartedAt:         sql.NullTime{},
			IsPreviousClaimed: true,
			PointReward:       DefaultReward,
		}, nil
	}
	if err != nil {
		return status, fmt.Errorf("failed to get farm status: %w", err)
	}

	if status.StartedAt.Valid && time.Now().UTC().Sub(status.StartedAt.Time.UTC()) >= CooldownDuration {
		status.IsInProgress = false
		status.StartedAt = sql.NullTime{}

		updateQuery, updateArgs, err := squirrel.Update("farm_game").
			Set("is_in_progress", false).
			Set("started_at", nil).
			Where(squirrel.Eq{"player": player}).
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return status, fmt.Errorf("failed to build update query: %w", err)
		}

		_, err = r.db.ExecContext(context.Background(), updateQuery, updateArgs...)
		if err != nil {
			return status, fmt.Errorf("failed to update expired status: %w", err)
		}
	}

	status.PointReward = DefaultReward

	return status, nil
}

func (r *Repository) ClaimPoints(playerID int64) (int, error) {
	var status FarmStatus
	selectQuery, selectArgs, err := squirrel.Select(
		"is_in_progress",
		"started_at",
		"is_previous_claimed",
	).
		From("farm_game").
		Where(squirrel.Eq{"player": playerID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build select query: %w", err)
	}

	err = r.db.QueryRowContext(context.Background(), selectQuery, selectArgs...).
		Scan(&status.IsInProgress, &status.StartedAt, &status.IsPreviousClaimed)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("no farming session found")
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get farm status: %w", err)
	}

	timeSinceStart := time.Now().UTC().Sub(status.StartedAt.Time.UTC())
	if timeSinceStart < CooldownDuration {
		return 0, fmt.Errorf("farming session not yet complete, %v remaining", CooldownDuration-timeSinceStart)
	}

	if status.IsPreviousClaimed {
		return 0, fmt.Errorf("reward already claimed")
	}

	updateQuery, updateArgs, err := squirrel.Update("farm_game").
		Set("is_in_progress", false).
		Set("is_previous_claimed", true).
		Set("started_at", nil).
		Where(squirrel.Eq{"player": playerID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build update query: %w", err)
	}

	result, err := r.db.ExecContext(context.Background(), updateQuery, updateArgs...)
	if err != nil {
		return 0, fmt.Errorf("failed to update farm status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return 0, fmt.Errorf("no farm record found to update")
	}

	updatePointsQuery, updatePointsArgs, err := squirrel.Update("users").
		Set("points", squirrel.Expr("points + ?", DefaultReward)).
		Where(squirrel.Eq{"telegram_id": playerID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build points update query: %w", err)
	}

	_, err = r.db.ExecContext(context.Background(), updatePointsQuery, updatePointsArgs...)
	if err != nil {
		return 0, fmt.Errorf("failed to update user points: %w", err)
	}

	return DefaultReward, nil
}

//func (r *Repository) GetHarvestPoints(telegramID int64) (int, error) {
//	r.Lock()
//	defer r.Unlock()
//
//	startTime, exists := r.Cache[telegramID]
//	if !exists {
//		return 0, nil
//	}
//
//	elapsed := time.Now().UTC().Sub(startTime)
//	if elapsed >= CooldownDuration {
//		delete(r.Cache, telegramID)
//		err := r.UpdateUserPoints(context.TODO(), telegramID, 1000)
//		if err != nil {
//			return 0, err
//		}
//
//		return 1000, nil
//	}
//
//	time.Now().Add(8 * time.Hour).Sub(time.Now())
//
//	return int(float64(1000) * float64(elapsed) / float64(8*time.Hour)), nil
//}
