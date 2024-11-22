package repository

import (
	"database/sql"
	"errors"
	"time"
)

type FarmStatus struct {
	CanHarvest       bool    `json:"canHarvest"`
	TimeUntilHarvest float64 `json:"timeUntilHarvest"`
}

const CooldownDuration = 8 * time.Hour

func (r *Repository) Status(player int64) (FarmStatus, error) {
	var status FarmStatus
	var lastHarvestedAt sql.NullTime

	err := r.db.QueryRow(`
        SELECT last_harvested_at 
        FROM farm_game 
        WHERE player = $1`,
		player,
	).Scan(&lastHarvestedAt)

	if err == sql.ErrNoRows {
		return FarmStatus{CanHarvest: true, TimeUntilHarvest: 0}, nil
	}
	if err != nil {
		return status, err
	}

	if !lastHarvestedAt.Valid || time.Since(lastHarvestedAt.Time) >= CooldownDuration {
		status.CanHarvest = true
		status.TimeUntilHarvest = 0
	} else {
		status.CanHarvest = false
		status.TimeUntilHarvest = CooldownDuration.Seconds() - time.Since(lastHarvestedAt.Time).Seconds()
	}

	return status, nil
}

func (r *Repository) Harvest(player int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var lastHarvestedAt sql.NullTime
	err = tx.QueryRow(`
        SELECT last_harvested_at 
        FROM farm_game 
        WHERE player = $1 
        FOR UPDATE`,
		player,
	).Scan(&lastHarvestedAt)

	if err == sql.ErrNoRows {
		_, err = tx.Exec(`
            INSERT INTO farm_game (player, last_harvested_at) 
            VALUES ($1, NOW())`,
			player,
		)
	} else if err != nil {
		return err
	} else {
		if lastHarvestedAt.Valid && time.Since(lastHarvestedAt.Time) < CooldownDuration {
			return errors.New("cooldown period not finished")
		}

		_, err = tx.Exec(`
            UPDATE farm_game 
            SET last_harvested_at = NOW() 
            WHERE player = $1`,
			player,
		)
	}
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
        UPDATE users 
        SET points = points + 1000 
        WHERE telegram_id = $1`,
		player,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}