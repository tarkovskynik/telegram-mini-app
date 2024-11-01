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
	Validations []QuestValidation
}

type UserSocialQuest struct {
	QuestID    uuid.UUID
	UserID     int64
	Completed  bool
	StartedAt  *time.Time
	FinishedAt *time.Time
}

type UserValidationsStatus map[QuestValidation]struct{}

type QuestValidation struct {
	ValidationID   int
	ValidationName string
}

type QuestValidationKind struct {
	ValidationID   int
	ValidationName string
}
