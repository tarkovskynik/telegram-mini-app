package model

import (
	"time"

	"github.com/google/uuid"
)

type SocialQuest struct {
	QuestID     uuid.UUID
	Image       string
	Title       string
	Description string
	PointReward int
	CreatedAt   time.Time
}
