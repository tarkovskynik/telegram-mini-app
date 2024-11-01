package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"UD_telegram_miniapp/internal/model"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type questWithValidations struct {
	QuestID         uuid.UUID      `db:"quest_id"`
	Image           string         `db:"image"`
	Title           string         `db:"title"`
	Description     string         `db:"description"`
	PointReward     int            `db:"point_reward"`
	CreatedAt       time.Time      `db:"created_at"`
	ValidationIDs   pq.Int64Array  `db:"validation_ids"`
	ValidationNames pq.StringArray `db:"validation_names"`
	Completed       bool           `db:"completed"`
	StartedAt       *time.Time     `db:"started_at"`
	FinishedAt      *time.Time     `db:"finished_at"`
}

type questValidation struct {
	ValidationID   int    `db:"validation_id"`
	ValidationName string `db:"validation_name"`
}

func (r *Repository) GetQuestsData(ctx context.Context, telegramID int64) ([]*model.SocialQuest, []*model.UserSocialQuest, error) {
	query := squirrel.Select(
		"sq.quest_id",
		"sq.image",
		"sq.title",
		"sq.description",
		"sq.point_reward",
		"sq.created_at",
		"array_agg(qv.validation_id) FILTER (WHERE qv.validation_id IS NOT NULL) as validation_ids",
		"array_agg(sqvk.validation_name) FILTER (WHERE sqvk.validation_name IS NOT NULL) as validation_names",
		"usq.completed",
		"usq.started_at",
		"usq.finished_at",
	).
		From("social_quests sq").
		LeftJoin("quest_validations qv ON qv.quest_id = sq.quest_id").
		LeftJoin("social_quest_validation_kinds sqvk ON qv.validation_id = sqvk.validation_id").
		LeftJoin("users_social_quests usq ON usq.social_quest_id = sq.quest_id AND usq.user_telegram_id = ?", telegramID).
		GroupBy("sq.quest_id", "usq.completed", "usq.started_at", "usq.finished_at").
		OrderBy("sq.quest_id").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, nil, err
	}

	var dbQuests []*questWithValidations
	err = r.db.SelectContext(ctx, &dbQuests, sqlQuery, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*model.SocialQuest{}, []*model.UserSocialQuest{}, nil
		}
		return nil, nil, err
	}
	fmt.Println("dbQuests repo:")
	for _, q := range dbQuests {
		fmt.Println("validations required", q.ValidationIDs)
	}

	quests := make([]*model.SocialQuest, len(dbQuests))
	userQuests := make([]*model.UserSocialQuest, len(dbQuests))

	for i, q := range dbQuests {
		validations := make([]model.QuestValidation, len(q.ValidationIDs))
		for j := range q.ValidationIDs {
			validations[j] = model.QuestValidation{
				ValidationID:   int(q.ValidationIDs[j]),
				ValidationName: q.ValidationNames[j],
			}
		}

		quests[i] = &model.SocialQuest{
			QuestID:     q.QuestID,
			Image:       q.Image,
			Title:       q.Title,
			Description: q.Description,
			PointReward: q.PointReward,
			CreatedAt:   q.CreatedAt,
			Validations: validations,
		}

		userQuests[i] = &model.UserSocialQuest{
			QuestID:    q.QuestID,
			UserID:     telegramID,
			Completed:  q.Completed,
			StartedAt:  q.StartedAt,
			FinishedAt: q.FinishedAt,
		}
	}

	return quests, userQuests, nil
}

func (r *Repository) GetUserValidationsStatus(ctx context.Context, telegramID int64) (model.UserValidationsStatus, error) {
	query := squirrel.Select(
		"svk.validation_id",
		"svk.validation_name",
	).
		From("user_validations uv").
		Join("social_quest_validation_kinds svk ON svk.validation_id = uv.validation_id").
		Where(squirrel.Eq{"user_telegram_id": telegramID}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build validation status query: %w", err)
	}

	var dbValidations []*questValidation
	err = r.db.SelectContext(ctx, &dbValidations, sqlQuery, args...)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to get user validations: %w", err)
		}
		return make(model.UserValidationsStatus), nil
	}

	validationStatus := make(model.UserValidationsStatus)
	for _, v := range dbValidations {
		validation := model.QuestValidation{
			ValidationID:   v.ValidationID,
			ValidationName: v.ValidationName,
		}
		validationStatus[validation] = struct{}{}
	}

	return validationStatus, nil
}

func (r *Repository) GetQuestDataByID(ctx context.Context, telegramID int64, questID uuid.UUID) (*model.SocialQuest, *model.UserSocialQuest, error) {
	query := squirrel.Select(
		"sq.quest_id",
		"sq.image",
		"sq.title",
		"sq.description",
		"sq.point_reward",
		"sq.created_at",
		"array_agg(qv.validation_id) FILTER (WHERE qv.validation_id IS NOT NULL) as validation_ids",
		"array_agg(sqvk.validation_name) FILTER (WHERE sqvk.validation_name IS NOT NULL) as validation_names",
		"usq.completed",
		"usq.started_at",
		"usq.finished_at",
	).
		From("social_quests sq").
		LeftJoin("quest_validations qv ON qv.quest_id = sq.quest_id").
		LeftJoin("social_quest_validation_kinds sqvk ON qv.validation_id = sqvk.validation_id").
		LeftJoin("users_social_quests usq ON usq.social_quest_id = sq.quest_id AND usq.user_telegram_id = ?", telegramID).
		Where(squirrel.Eq{"sq.quest_id": questID}).
		GroupBy("sq.quest_id", "usq.completed", "usq.started_at", "usq.finished_at").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build query: %w", err)
	}

	var dbQuest questWithValidations
	err = r.db.GetContext(ctx, &dbQuest, sqlQuery, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, fmt.Errorf("failed to get quest: %w", err)
	}

	validations := make([]model.QuestValidation, len(dbQuest.ValidationIDs))
	for i := range dbQuest.ValidationIDs {
		validations[i] = model.QuestValidation{
			ValidationID:   int(dbQuest.ValidationIDs[i]),
			ValidationName: dbQuest.ValidationNames[i],
		}
	}

	quest := &model.SocialQuest{
		QuestID:     dbQuest.QuestID,
		Image:       dbQuest.Image,
		Title:       dbQuest.Title,
		Description: dbQuest.Description,
		PointReward: dbQuest.PointReward,
		CreatedAt:   dbQuest.CreatedAt,
		Validations: validations,
	}

	userQuest := &model.UserSocialQuest{
		QuestID:    dbQuest.QuestID,
		UserID:     telegramID,
		Completed:  dbQuest.Completed,
		StartedAt:  dbQuest.StartedAt,
		FinishedAt: dbQuest.FinishedAt,
	}

	return quest, userQuest, nil
}

func (r *Repository) ClaimQuest(ctx context.Context, telegramID int64, questID uuid.UUID) error {
	updateQuery, args, err := squirrel.
		Update("users_social_quests").
		Set("completed", true).
		Set("finished_at", time.Now()).
		Where(squirrel.And{
			squirrel.Eq{
				"user_telegram_id": telegramID,
				"social_quest_id":  questID,
				"completed":        false,
			},
			squirrel.NotEq{"started_at": nil},
		}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	result, err := r.db.ExecContext(ctx, updateQuery, args...)
	if err != nil {
		return fmt.Errorf("failed to update quest status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		var status questWithValidations

		checkQuery, args, err := squirrel.
			Select("started_at", "completed").
			From("users_social_quests").
			Where(squirrel.Eq{
				"user_telegram_id": telegramID,
				"social_quest_id":  questID,
			}).
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return fmt.Errorf("failed to build check query: %w", err)
		}

		err = r.db.GetContext(ctx, &status, checkQuery, args...)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("failed to check quest status: %w", err)
		}

		if status.StartedAt == nil {
			return ErrQuestNotStarted
		}
		if status.Completed {
			return ErrQuestAlreadyClaimed
		}
	}

	return nil
}

func (r *Repository) CreateSocialQuest(ctx context.Context, quest *model.SocialQuest) error {
	return r.Transaction(ctx, func(tx *sqlx.Tx) error {
		questQuery, args, err := squirrel.
			Insert("social_quests").
			SetMap(map[string]interface{}{
				"quest_id":     quest.QuestID,
				"image":        quest.Image,
				"title":        quest.Title,
				"description":  quest.Description,
				"point_reward": quest.PointReward,
				"created_at":   time.Now(),
			}).
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return fmt.Errorf("failed to build quest insert query: %w", err)
		}

		if _, err := tx.ExecContext(ctx, questQuery, args...); err != nil {
			return fmt.Errorf("failed to insert social quest: %w", err)
		}

		if len(quest.Validations) > 0 {
			validationBuilder := squirrel.
				Insert("quest_validations").
				Columns("quest_id", "validation_id").
				PlaceholderFormat(squirrel.Dollar)

			for _, validation := range quest.Validations {
				validationBuilder = validationBuilder.Values(quest.QuestID, validation.ValidationID)
			}

			validationQuery, validationArgs, err := validationBuilder.ToSql()
			if err != nil {
				return fmt.Errorf("failed to build validation insert query: %w", err)
			}

			if _, err := tx.ExecContext(ctx, validationQuery, validationArgs...); err != nil {
				return fmt.Errorf("failed to insert quest validations: %w", err)
			}
		}

		userQuery, userArgs, err := squirrel.
			Select("telegram_id").
			From("users").
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return fmt.Errorf("failed to build users select query: %w", err)
		}

		var userIDs []int64
		if err := tx.SelectContext(ctx, &userIDs, userQuery, userArgs...); err != nil {
			return fmt.Errorf("failed to get user IDs: %w", err)
		}

		if len(userIDs) > 0 {
			userQuestBuilder := squirrel.
				Insert("users_social_quests").
				Columns("user_telegram_id", "social_quest_id", "completed").
				PlaceholderFormat(squirrel.Dollar)

			for _, userID := range userIDs {
				userQuestBuilder = userQuestBuilder.Values(userID, quest.QuestID, false)
			}

			userQuestQuery, userQuestArgs, err := userQuestBuilder.ToSql()
			if err != nil {
				return fmt.Errorf("failed to build user_social_quests insert query: %w", err)
			}

			if _, err := tx.ExecContext(ctx, userQuestQuery, userQuestArgs...); err != nil {
				return fmt.Errorf("failed to insert user_social_quests entries: %w", err)
			}
		}

		return nil
	})
}

func (r *Repository) CreateValidationKind(ctx context.Context, validation *model.QuestValidationKind) error {
	query, args, err := squirrel.
		Insert("social_quest_validation_kinds").
		Columns("validation_id", "validation_name").
		Values(
			squirrel.Expr("default"),
			validation.ValidationName,
		).PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to create validation kind: %w", err)
	}

	return nil
}

func (r *Repository) ListValidationKinds(ctx context.Context) ([]*model.QuestValidationKind, error) {
	query, args, err := squirrel.
		Select("validation_id", "validation_name").
		From("social_quest_validation_kinds").
		OrderBy("validation_id").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var validations []questValidation
	err = r.db.SelectContext(ctx, &validations, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list validation kinds: %w", err)
	}

	validationList := make([]*model.QuestValidationKind, len(validations))
	for i, v := range validations {
		validationList[i] = &model.QuestValidationKind{
			ValidationID:   v.ValidationID,
			ValidationName: v.ValidationName,
		}
	}

	return validationList, nil
}

func (r *Repository) AddQuestValidation(ctx context.Context, questID uuid.UUID, validationID int) error {
	return r.Transaction(ctx, func(tx *sqlx.Tx) error {
		questQuery, questArgs, err := squirrel.
			Select("1").
			From("social_quests").
			Where(squirrel.Eq{"quest_id": questID}).
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return err
		}

		var exists bool
		err = tx.GetContext(ctx, &exists, questQuery, questArgs...)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrQuestNotFound
			}
			return err
		}

		validationQuery, validationArgs, err := squirrel.
			Select("1").
			From("social_quest_validation_kinds").
			Where(squirrel.Eq{"validation_id": validationID}).
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return err
		}

		err = tx.GetContext(ctx, &exists, validationQuery, validationArgs...)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrValidationNotFound
			}
			return err
		}

		existsQuery, existsArgs, err := squirrel.
			Select("1").
			From("quest_validations").
			Where(squirrel.Eq{
				"quest_id":      questID,
				"validation_id": validationID,
			}).
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return err
		}

		err = tx.GetContext(ctx, &exists, existsQuery, existsArgs...)
		if err == nil {
			return ErrValidationAlreadyExists
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		insertQuery, insertArgs, err := squirrel.
			Insert("quest_validations").
			Columns("quest_id", "validation_id").
			Values(questID, validationID).
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, insertQuery, insertArgs...)
		return err
	})
}

func (r *Repository) RemoveQuestValidation(ctx context.Context, questID uuid.UUID, validationID int) error {
	query, args, err := squirrel.
		Delete("quest_validations").
		Where(squirrel.Eq{
			"quest_id":      questID,
			"validation_id": validationID,
		}).
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
