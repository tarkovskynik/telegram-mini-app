package model

import "time"

type User struct {
	TelegramID       int64
	Handle           string
	Username         string
	ReferrerID       *int64
	Referrals        int
	Points           int
	ProfileImage     string
	JoinWaitlist     *bool
	RegistrationDate time.Time
	AuthDate         time.Time
}

type UserReferral struct {
	TelegramUsername string
	ReferralCount    int
	Points           int
}
