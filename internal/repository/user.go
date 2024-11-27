package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"UD_telegram_miniapp/internal/model"
	"UD_telegram_miniapp/pkg/logger"
	"go.uber.org/zap"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type User struct {
	TelegramID       int64     `db:"telegram_id"`
	Handle           string    `db:"handle"`
	Username         string    `db:"username"`
	ReferrerID       *int64    `db:"referrer_id"`
	Referrals        int       `db:"referrals"`
	Points           int       `db:"points"`
	JoinWaitlist     *bool     `db:"join_waitlist"`
	RegistrationDate time.Time `db:"registration_date"`
	AuthDate         time.Time `db:"last_auth_date"`
}

type userReferral struct {
	TelegramID       int64  `db:"telegram_id"`
	TelegramUsername string `db:"username"`
	ReferralCount    int    `db:"referrals"`
	Points           int    `db:"points"`
}

func (r *Repository) CreateUser(ctx context.Context, user *model.User) error {
	return r.Transaction(ctx, func(tx *sqlx.Tx) error {
		query, args, err := squirrel.
			Insert("users").
			SetMap(map[string]interface{}{
				"telegram_id":       user.TelegramID,
				"handle":            user.Handle,
				"username":          user.Username,
				"referrer_id":       user.ReferrerID,
				"registration_date": user.RegistrationDate,
				"last_auth_date":    user.AuthDate,
				"points":            user.Points,
				"referrals":         user.Referrals,
				"join_waitlist":     user.JoinWaitlist,
			}).
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return fmt.Errorf("failed to build user insert query: %w", err)
		}

		_, err = tx.ExecContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to insert user: %w", err)
		}

		if user.ReferrerID != nil {
			updateQuery, updateArgs, err := squirrel.
				Update("users").
				Set("referrals", squirrel.Expr("referrals + 1")).
				Where(squirrel.Eq{"telegram_id": user.ReferrerID}).
				PlaceholderFormat(squirrel.Dollar).
				ToSql()
			if err != nil {
				return fmt.Errorf("failed to build referrer update query: %w", err)
			}

			_, err = tx.ExecContext(ctx, updateQuery, updateArgs...)
			if err != nil {
				return fmt.Errorf("failed to update referrer: %w", err)
			}
		}

		questQuery, questArgs, err := squirrel.
			Insert("daily_quests").
			SetMap(map[string]interface{}{
				"user_telegram_id":         user.TelegramID,
				"consecutive_days_claimed": 0,
			}).
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return fmt.Errorf("failed to build daily quest insert query: %w", err)
		}

		_, err = tx.ExecContext(ctx, questQuery, questArgs...)
		if err != nil {
			return fmt.Errorf("failed to insert daily quest: %w", err)
		}

		socialQuestsQuery, socialQuestsArgs, err := squirrel.
			Select("quest_id").
			From("social_quests").
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return fmt.Errorf("failed to build social quests select query: %w", err)
		}

		var questIDs []uuid.UUID
		err = tx.Select(&questIDs, socialQuestsQuery, socialQuestsArgs...)
		if err != nil {
			return fmt.Errorf("failed to get social quests: %w", err)
		}

		if len(questIDs) > 0 {
			builder := squirrel.
				Insert("users_social_quests").
				Columns("user_telegram_id", "social_quest_id", "completed")

			for _, questID := range questIDs {
				builder = builder.Values(user.TelegramID, questID, false)
			}

			query, args, err := builder.PlaceholderFormat(squirrel.Dollar).ToSql()
			if err != nil {
				return fmt.Errorf("failed to build social quests insert query: %w", err)
			}

			_, err = tx.ExecContext(ctx, query, args...)
			if err != nil {
				return fmt.Errorf("failed to insert user social quests: %w", err)
			}
		}

		referralQuestsQuery, referralQuestArgs, err := squirrel.
			Select("quest_id").
			From("referral_quests").
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return fmt.Errorf("failed to build referral quests select query: %w", err)
		}

		var referralQuestIDs []uuid.UUID
		err = tx.Select(&referralQuestIDs, referralQuestsQuery, referralQuestArgs...)
		if err != nil {
			return fmt.Errorf("failed to get referral quests: %w", err)
		}

		if len(referralQuestIDs) > 0 {
			builder := squirrel.
				Insert("referral_quests_users").
				Columns("user_telegram_id", "quest_id", "completed")

			for _, questID := range referralQuestIDs {
				builder = builder.Values(user.TelegramID, questID, false)
			}

			query, args, err := builder.PlaceholderFormat(squirrel.Dollar).ToSql()
			if err != nil {
				return fmt.Errorf("failed to build referral quests insert query: %w", err)
			}

			_, err = tx.ExecContext(ctx, query, args...)
			if err != nil {
				return fmt.Errorf("failed to insert user referral quests: %w", err)
			}
		}

		return nil
	})
}

func (r *Repository) GetUserByTelegramID(ctx context.Context, telegramID int64) (*model.User, error) {
	var user User
	query, args, err := squirrel.
		Select("*").
		From("users").
		Where(squirrel.Eq{"telegram_id": telegramID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	err = r.db.GetContext(ctx, &user, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	harvestedPoints, err := r.GetHarvestPoints(telegramID)
	if err != nil {
		return nil, err
	}
	user.Points += harvestedPoints

	return &model.User{
		TelegramID:       user.TelegramID,
		Handle:           user.Handle,
		Username:         user.Username,
		ReferrerID:       user.ReferrerID,
		Referrals:        user.Referrals,
		Points:           user.Points,
		JoinWaitlist:     user.JoinWaitlist,
		RegistrationDate: user.RegistrationDate,
		AuthDate:         user.AuthDate,
	}, nil
}

func (r *Repository) getUserWithTx(ctx context.Context, tx *sqlx.Tx, telegramID int64) (*model.User, error) {
	var user User
	query, args, err := squirrel.
		Select("*").
		From("users").
		Where(squirrel.Eq{"telegram_id": telegramID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	err = tx.GetContext(ctx, &user, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &model.User{
		TelegramID:       user.TelegramID,
		Handle:           user.Handle,
		Username:         user.Username,
		ReferrerID:       user.ReferrerID,
		Referrals:        user.Referrals,
		Points:           user.Points,
		JoinWaitlist:     user.JoinWaitlist,
		RegistrationDate: user.RegistrationDate,
		AuthDate:         user.AuthDate,
	}, nil
}

func (r *Repository) UpdateUserPoints(ctx context.Context, telegramID int64, points int) error {
	return r.Transaction(ctx, func(tx *sqlx.Tx) error {
		user, err := r.getUserWithTx(ctx, tx, telegramID)
		if err != nil {
			return err
		}

		updateQuery, updateArgs, err := squirrel.
			Update("users").
			Set("points", squirrel.Expr("points + ?", points)).
			Where(squirrel.Eq{"telegram_id": telegramID}).
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return err
		}

		_, err = tx.Exec(updateQuery, updateArgs...)
		if err != nil {
			return err
		}

		if user.ReferrerID != nil {
			referrerPoints := int(math.Ceil(float64(points) * 0.1))

			updateReferrerQuery, referrerArgs, err := squirrel.
				Update("users").
				Set("points", squirrel.Expr("points + ?", referrerPoints)).
				Where(squirrel.Eq{"telegram_id": user.ReferrerID}).
				PlaceholderFormat(squirrel.Dollar).
				ToSql()
			if err != nil {
				return err
			}

			_, err = tx.Exec(updateReferrerQuery, referrerArgs...)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *Repository) updateUserPointsWithTx(ctx context.Context, tx *sqlx.Tx, telegramID int64, points int) error {
	user, err := r.getUserWithTx(ctx, tx, telegramID)
	if err != nil {
		return err
	}

	updateQuery, updateArgs, err := squirrel.
		Update("users").
		Set("points", squirrel.Expr("points + ?", points)).
		Where(squirrel.Eq{"telegram_id": telegramID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, updateQuery, updateArgs...)
	if err != nil {
		return err
	}

	if user.ReferrerID != nil {
		referrerPoints := int(math.Ceil(float64(points) * 0.1))

		updateReferrerQuery, referrerArgs, err := squirrel.
			Update("users").
			Set("points", squirrel.Expr("points + ?", referrerPoints)).
			Where(squirrel.Eq{"telegram_id": user.ReferrerID}).
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, updateReferrerQuery, referrerArgs...)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Repository) UpdateUserWaitlistStatus(ctx context.Context, telegramID int64, status bool) error {
	return r.Transaction(ctx, func(tx *sqlx.Tx) error {
		_, err := r.getUserWithTx(ctx, tx, telegramID)
		if err != nil {
			return err
		}

		updateQuery, updateArgs, err := squirrel.
			Update("users").
			Set("join_waitlist", status).
			Where(squirrel.Eq{"telegram_id": telegramID}).
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, updateQuery, updateArgs...)
		if err != nil {
			return err
		}

		return nil
	})
}

func (r *Repository) GetUserWaitlistStatus(ctx context.Context, telegramID int64) (*bool, error) {
	var status bool

	user, err := r.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, err
	}

	status = *user.JoinWaitlist

	return &status, nil
}

func (r *Repository) GetTopUsers(ctx context.Context, limit int) ([]*model.User, error) {
	var users []User

	err := r.Transaction(ctx, func(tx *sqlx.Tx) error {
		query, args, err := squirrel.
			Select("telegram_id", "username", "points", "referrals").
			From("users").
			OrderBy("points DESC").
			Limit(uint64(limit)).
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return err
		}

		err = tx.SelectContext(ctx, &users, query, args...)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	userList := make([]*model.User, len(users))
	for i, user := range users {
		userList[i] = &model.User{
			TelegramID: user.TelegramID,
			Username:   user.Username,
			Points:     user.Points,
			Referrals:  user.Referrals,
		}
	}

	return userList, nil
}

func (r *Repository) GetUserReferrals(ctx context.Context, telegramID int64) ([]*model.UserReferral, error) {
	query := squirrel.Select(
		"telegram_id",
		"username",
		"referrals",
		"points",
	).
		From("users").
		Where(squirrel.Eq{"referrer_id": telegramID}).
		OrderBy("points DESC").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var referrals []*userReferral
	err = r.db.SelectContext(ctx, &referrals, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get user referrals: %w", err)
	}

	refs := make([]*model.UserReferral, len(referrals))
	for i, ref := range referrals {
		refs[i] = &model.UserReferral{
			TelegramID:       ref.TelegramID,
			TelegramUsername: ref.TelegramUsername,
			ReferralCount:    ref.ReferralCount,
			Points:           ref.Points,
		}
	}

	return refs, nil
}

// game
func (r *Repository) GetPlayerEnergy(ctx context.Context, playerID int64) (total int, remaining int, err error) {
	query, args, err := squirrel.
		Select("p.total_energy").
		From("players p").
		Where(squirrel.Eq{"p.user_id": playerID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to build query: %w", err)
	}

	err = r.db.GetContext(ctx, &total, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			const defaultTotalEnergy = 3
			const defaultCooldownSettingID = 1

			insertQuery, insertArgs, err := squirrel.
				Insert("players").
				Columns("user_id", "total_energy", "cooldown_setting_id").
				Values(playerID, defaultTotalEnergy, defaultCooldownSettingID).
				Suffix("RETURNING total_energy").
				PlaceholderFormat(squirrel.Dollar).
				ToSql()
			if err != nil {
				return 0, 0, fmt.Errorf("failed to build insert query: %w", err)
			}

			err = r.db.GetContext(ctx, &total, insertQuery, insertArgs...)
			if err != nil {
				return 0, 0, fmt.Errorf("failed to create new player: %w", err)
			}
			remaining = total
			return total, remaining, nil
		}
		return 0, 0, fmt.Errorf("failed to get player total energy: %w", err)
	}

	cooldownQuery, cooldownArgs, err := squirrel.
		Select("COUNT(eu.energy_number)").
		From("energy_uses eu").
		Join("players p ON p.user_id = eu.user_id").
		Join("cooldown_settings cs ON cs.id = p.cooldown_setting_id").
		Where(squirrel.And{
			squirrel.Eq{"eu.user_id": playerID},
			squirrel.Expr("eu.used_at > NOW() - (cs.cooldown_hours || ' hours')::interval"),
		}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to build cooldown query: %w", err)
	}

	var usedEnergy int
	err = r.db.GetContext(ctx, &usedEnergy, cooldownQuery, cooldownArgs...)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get energy uses count: %w", err)
	}

	remaining = total - usedEnergy
	return total, remaining, nil
}

func (r *Repository) GetEnergyStatus(ctx context.Context, playerID int64) (energyNumb int, usedAt time.Time, err error) {
	query, args, err := squirrel.
		Select("energy_number", "used_at").
		From("energy_uses").
		Where(squirrel.And{
			squirrel.Eq{"user_id": playerID},
		}).
		PlaceholderFormat(squirrel.Dollar).
		OrderBy("used_at DESC").
		Limit(1).
		ToSql()

	if err != nil {
		return -1, time.Time{}, fmt.Errorf("failed to build energy status query: %w", err)
	}

	status := struct {
		EnergyNumb int       `db:"energy_number"`
		UsedAt     time.Time `db:"used_at"`
	}{}

	err = r.db.GetContext(ctx, &status, query, args...)
	if err != nil {
		return -1, time.Time{}, fmt.Errorf("failed to get energy status: %w", err)
	}

	return status.EnergyNumb, status.UsedAt, nil
}

func (r *Repository) UpdatePlayerEnergy(ctx context.Context, userID int64) error {
	return r.Transaction(ctx, func(tx *sqlx.Tx) error {
		query, args, err := squirrel.
			Select("COUNT(*)").
			From("energy_uses eu").
			Where(squirrel.And{
				squirrel.Eq{"eu.user_id": userID},
				squirrel.Expr("eu.used_at > NOW() - make_interval(hours => cs.cooldown_hours)"),
			}).
			Join("players p ON p.user_id = eu.user_id").
			Join("cooldown_settings cs ON cs.id = p.cooldown_setting_id").
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return fmt.Errorf("failed to build used energy query: %w", err)
		}

		var usedEnergy int
		err = tx.GetContext(ctx, &usedEnergy, query, args...)
		if err != nil {
			return fmt.Errorf("failed to get used energy count: %w", err)
		}

		insertQuery, insertArgs, err := squirrel.
			Insert("energy_uses").
			Columns("user_id", "energy_number", "used_at").
			Values(userID, usedEnergy+1, squirrel.Expr("NOW()")).
			Suffix(`ON CONFLICT (user_id, energy_number) DO UPDATE SET used_at = NOW()`).
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return fmt.Errorf("failed to build insert query: %w", err)
		}

		_, err = tx.ExecContext(ctx, insertQuery, insertArgs...)
		if err != nil {
			return fmt.Errorf("failed to record energy use: %w", err)
		}

		return nil
	})
}

func (r *Repository) ResetEnergy(ctx context.Context, userID int64) error {
	log := logger.Logger()

	query, args, err := squirrel.
		Delete("energy_uses").
		Where(squirrel.Eq{"user_id": userID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		log.Error("failed to build reset energy query",
			zap.Int64("user_id", userID),
			zap.Error(err))
		return fmt.Errorf("failed to build reset energy query: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Error("failed to execute reset energy query",
			zap.Int64("user_id", userID),
			zap.Error(err))
		return fmt.Errorf("failed to reset user energy: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Error("failed to get affected rows count",
			zap.Int64("user_id", userID),
			zap.Error(err))
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	log.Info("successfully reset user energy",
		zap.Int64("user_id", userID),
		zap.Int64("records_deleted", rows))

	return nil
}
