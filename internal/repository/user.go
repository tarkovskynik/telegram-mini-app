package repository

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"time"

	"UD_telegram_miniapp/internal/model"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

type User struct {
	TelegramID       int64     `db:"telegram_id"`
	Handle           string    `db:"handle"`
	Username         string    `db:"username"`
	ReferrerID       *int64    `db:"referrer_id"`
	Referrals        int       `db:"referrals"`
	Points           int       `db:"points"`
	ProfileImage     string    `db:"profile_image"`
	JoinWaitlist     *bool     `db:"join_waitlist"`
	RegistrationDate time.Time `db:"registration_date"`
	AuthDate         time.Time `db:"last_auth_date"`
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
				"profile_image":     user.ProfileImage,
				"registration_date": user.RegistrationDate,
				"last_auth_date":    user.AuthDate,
				"points":            user.Points,
				"referrals":         user.Referrals,
				"join_waitlist":     user.JoinWaitlist,
			}).
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return err
		}

		_, err = tx.Exec(query, args...)
		if err != nil {
			return err
		}

		if user.ReferrerID != nil {
			updateQuery, updateArgs, err := squirrel.
				Update("users").
				Set("referrals", squirrel.Expr("referrals + 1")).
				Where(squirrel.Eq{"telegram_id": user.ReferrerID}).
				PlaceholderFormat(squirrel.Dollar).
				ToSql()
			if err != nil {
				return err
			}

			_, err = tx.Exec(updateQuery, updateArgs...)
			if err != nil {
				return err
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
			return err
		}

		_, err = tx.Exec(questQuery, questArgs...)
		if err != nil {
			return err
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
		ProfileImage:     user.ProfileImage,
		JoinWaitlist:     user.JoinWaitlist,
		RegistrationDate: user.RegistrationDate,
		AuthDate:         user.AuthDate,
	}, nil
}

func (r *Repository) getUserWithTx(tx *sqlx.Tx, telegramID int64) (*model.User, error) {
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

	err = tx.Get(&user, query, args...)
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
		ProfileImage:     user.ProfileImage,
		JoinWaitlist:     user.JoinWaitlist,
		RegistrationDate: user.RegistrationDate,
		AuthDate:         user.AuthDate,
	}, nil
}

func (r *Repository) UpdateUserPoints(ctx context.Context, telegramID int64, points int) error {
	return r.Transaction(ctx, func(tx *sqlx.Tx) error {
		user, err := r.getUserWithTx(tx, telegramID)
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

func (r *Repository) UpdateUserWaitlistStatus(ctx context.Context, telegramID int64, status bool) error {
	return r.Transaction(ctx, func(tx *sqlx.Tx) error {
		_, err := r.getUserWithTx(tx, telegramID)
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

		_, err = tx.Exec(updateQuery, updateArgs...)
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
	var users []*model.User

	err := r.Transaction(ctx, func(tx *sqlx.Tx) error {
		query, args, err := squirrel.
			Select("username", "points", "profile_image", "referrals").
			From("users").
			OrderBy("points DESC").
			Limit(uint64(limit)).
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return err
		}

		err = tx.Select(&users, query, args...)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return users, nil
}
