package model

import "time"

type DailyQuest struct {
	UserTelegramID         int64
	LastClaimedAt          *time.Time
	NextClaimAvailable     *time.Time
	IsAvailable            bool
	HasNeverBeenClaimed    bool
	ConsecutiveDaysClaimed int
	DailyRewards           []DayReward
}

type DayReward struct {
	Day    int
	Reward int
}
