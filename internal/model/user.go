package model

import "time"

type User struct {
	TelegramID       int64
	Handle           string
	Username         string
	ReferrerID       *int64
	Referrals        int
	Points           int
	AvatarProxyPath  string
	JoinWaitlist     *bool
	RegistrationDate time.Time
	AuthDate         time.Time
}

type UserReferral struct {
	TelegramID       int64
	TelegramUsername string
	AvatarProxyPath  string
	ReferralCount    int
	Points           int
}
