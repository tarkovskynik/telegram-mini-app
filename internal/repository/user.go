package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"UD_telegram_miniapp/internal/model"

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
