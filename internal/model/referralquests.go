package model

import (
	"time"

	"github.com/google/uuid"
)

type ReferralQuest struct {
	QuestID           uuid.UUID
	ReferralsRequired int
	PointReward       int
	CreatedAt         time.Time
}

type UserReferralQuest struct {
	UserTelegramID int64
	QuestID        uuid.UUID
	Completed      bool
	StartedAt      *time.Time
	FinishedAt     *time.Time
}

type ReferralQuestStatus struct {
	QuestID           uuid.UUID
	ReferralsRequired int
	PointReward       int
	CurrentReferrals  int
	Completed         bool
	ReadyToClaim      bool
	StartedAt         *time.Time
	FinishedAt        *time.Time
}
