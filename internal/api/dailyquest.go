package api

import (
	"errors"
	"net/http"
	"time"

	"UD_telegram_miniapp/internal/service"
	"UD_telegram_miniapp/pkg/auth"
	"UD_telegram_miniapp/pkg/logger"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

type dailyQuestRoutes struct {
	ds service.DailyQuestServiceI
	a  *auth.TelegramAuth
}

func NewDailyQuestRoutes(handler *gin.RouterGroup, ds service.DailyQuestServiceI, a *auth.TelegramAuth) {
	r := &dailyQuestRoutes{ds: ds, a: a}
	h := handler.Group("/daily-quests")
	h.Use(a.TelegramAuthMiddleware())
	{
		h.GET("/me", r.GetDailyQuestStatus)
		h.PUT("/me", r.ClaimDailyQuest)
	}
}

type DayRewardResponse struct {
	Day    int `json:"day"`
	Reward int `json:"reward"`
}

type DailyQuestStatusResponse struct {
	UserTelegramID         int64               `json:"user_telegram_id"`
	LastClaimedAt          *time.Time          `json:"last_claimed_at,omitempty"`
	NextClaimAvailable     *time.Time          `json:"next_claim_available,omitempty"`
	IsAvailable            bool                `json:"is_available"`
	HasNeverBeenClaimed    bool                `json:"has_never_been_claimed"`
	ConsecutiveDaysClaimed int                 `json:"consecutive_days_claimed"`
	DailyRewards           []DayRewardResponse `json:"daily_rewards"`
}

func (r *dailyQuestRoutes) GetDailyQuestStatus(c *gin.Context) {
	log := logger.Logger()

	userData, exists := c.Get("telegram_user")
	if !exists {
		log.Error("telegram user data not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	u, ok := userData.(*auth.TelegramUserData)
	if !ok {
		log.Error("invalid type assertion for telegram user data")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	status, err := r.ds.GetStatus(c.Request.Context(), u.ID)
	if err != nil {
		log.Error("failed to get daily quest status", zap.Error(err))
		if errors.Is(err, service.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get daily quest status"})
		return
	}

	rewards := make([]DayRewardResponse, len(status.DailyRewards))
	for i, reward := range status.DailyRewards {
		rewards[i] = DayRewardResponse{
			Day:    reward.Day,
			Reward: reward.Reward,
		}
	}

	response := DailyQuestStatusResponse{
		UserTelegramID:         status.UserTelegramID,
		IsAvailable:            status.IsAvailable,
		HasNeverBeenClaimed:    status.HasNeverBeenClaimed,
		ConsecutiveDaysClaimed: status.ConsecutiveDaysClaimed,
		DailyRewards:           rewards,
	}

	if !(status.LastClaimedAt == nil) {
		response.LastClaimedAt = status.LastClaimedAt
	}
	if !(status.NextClaimAvailable == nil) {
		response.NextClaimAvailable = status.NextClaimAvailable
	}

	c.JSON(http.StatusOK, response)
}

func (r *dailyQuestRoutes) ClaimDailyQuest(c *gin.Context) {
	log := logger.Logger()

	userData, exists := c.Get("telegram_user")
	if !exists {
		log.Error("telegram user data not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	u, ok := userData.(*auth.TelegramUserData)
	if !ok {
		log.Error("invalid type assertion for telegram user data")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	err := r.ds.Claim(c.Request.Context(), u.ID)
	if err != nil {
		log.Error("failed to claim daily quest", zap.Error(err))
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		case errors.Is(err, service.ErrClaimNotAvailable):
			c.JSON(http.StatusForbidden, gin.H{
				"error": "The required time has not yet passed since your last claim",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to claim daily quest"})
		}
		return
	}

	r.GetDailyQuestStatus(c)
}
