package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type FarmStatus struct {
	CanHarvest       bool    `json:"canHarvest"`
	TimeUntilHarvest float64 `json:"timeUntilHarvest"`
}

const CooldownDuration = 8 * time.Hour

func (r *Repository) StartHarvest(playerID int64) error {
	r.Lock()
	defer r.Unlock()

	if _, exists := r.Cache[playerID]; exists {
		return errors.New("harvest already in progress")
	}

	startTime := time.Now().UTC()
	r.Cache[playerID] = startTime

	_, err := r.db.Exec(`
        UPDATE farm_game 
        SET last_harvested_at = $1 
        WHERE player = $2`,
		startTime, playerID)
	return err
}

func (r *Repository) GetHarvestPoints(telegramID int64) (int, error) {
	r.Lock()
	defer r.Unlock()

	startTime, exists := r.Cache[telegramID]
	if !exists {
		return 0, nil
	}

	elapsed := time.Now().UTC().Sub(startTime)
	if elapsed >= CooldownDuration {
		delete(r.Cache, telegramID)
		err := r.UpdateUserPoints(context.TODO(), telegramID, 1000)
		if err != nil {
			return 0, err
		}

		return 1000, nil
	}

	return int(float64(1000) * float64(elapsed) / float64(8*time.Hour)), nil
}

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

	if !lastHarvestedAt.Valid || time.Now().UTC().Sub(lastHarvestedAt.Time.UTC()) >= CooldownDuration {
		status.CanHarvest = true
		status.TimeUntilHarvest = 0
	} else {
		status.CanHarvest = false
		status.TimeUntilHarvest = CooldownDuration.Seconds() - time.Now().UTC().Sub(lastHarvestedAt.Time.UTC()).Seconds()
	}

	return status, nil
}

//func (r *Repository) Harvest(player int64) error {
//	tx, err := r.db.Begin()
//	if err != nil {
//		return err
//	}
//	defer tx.Rollback()
//
//	var lastHarvestedAt sql.NullTime
//	err = tx.QueryRow(`
//        SELECT last_harvested_at
//        FROM farm_game
//        WHERE player = $1
//        FOR UPDATE`,
//		player,
//	).Scan(&lastHarvestedAt)
//
//	now := time.Now().UTC()
//	if err == sql.ErrNoRows {
//		_, err = tx.Exec(`
//            INSERT INTO farm_game (player, last_harvested_at)
//            VALUES ($1, $2)`,
//			player, now,
//		)
//	} else if err != nil {
//		return err
//	} else {
//		if lastHarvestedAt.Valid && time.Now().UTC().Sub(lastHarvestedAt.Time.UTC()) < CooldownDuration {
//			return errors.New("cooldown period not finished")
//		}
//
//		_, err = tx.Exec(`
//            UPDATE farm_game
//            SET last_harvested_at = $1
//            WHERE player = $2`,
//			now, player,
//		)
//	}
//	if err != nil {
//		return err
//	}
//
//	_, err = tx.Exec(`
//        UPDATE users
//        SET points = points + 1000
//        WHERE telegram_id = $1`,
//		player,
//	)
//	if err != nil {
//		return err
//	}
//
//	return tx.Commit()
//}
