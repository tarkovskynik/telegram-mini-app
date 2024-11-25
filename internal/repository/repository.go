package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	"UD_telegram_miniapp/pkg/logger"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var (
	ErrNotFound            = errors.New("not found")
	ErrQuestNotStarted     = errors.New("quest not started")
	ErrQuestAlreadyClaimed = errors.New("quest already claimed")

	ErrQuestNotFound           = errors.New("quest not found")
	ErrValidationNotFound      = errors.New("validation not found")
	ErrValidationAlreadyExists = errors.New("validation already exists for quest")

	ErrAlreadyClaimed = errors.New("already claimed")
)

type Repository struct {
	db    *sqlx.DB
	Cache map[int64]time.Time
	sync.Mutex
}

func (r *Repository) Close() error {
	return r.db.Close()
}

func (r *Repository) Transaction(ctx context.Context, t func(tx *sqlx.Tx) error) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	err = t(tx)
	if err != nil {
		txErr := tx.Rollback()
		if txErr != nil {
			return errors.Wrapf(err, "rollback error: %v", txErr)
		}
		return err
	}
	return tx.Commit()
}

type Config struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func New(cfg Config) (*Repository, error) {
	url := cfg.GetDatabaseURL()
	db, err := sqlx.Connect("pgx", url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Logger().Info("Connected to database successfully")

	cache := make(map[int64]time.Time)
	return &Repository{
		db:    db,
		Cache: cache,
	}, nil
}

func (c *Config) GetDatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Name,
	)
}
