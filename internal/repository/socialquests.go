package repository

import (
	"time"

	"github.com/google/uuid"
)

type SocialQuest struct {
	QuestID     uuid.UUID `db:"quest_id"`
	Image       string    `db:"image"`
	Title       string    `db:"title"`
	Description string    `db:"description"`
	PointReward int       `db:"point_reward"`
	CreatedAt   time.Time `db:"created_at"`
}
