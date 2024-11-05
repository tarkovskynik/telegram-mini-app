package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"UD_telegram_miniapp/internal/model"
	"UD_telegram_miniapp/internal/service"
	"UD_telegram_miniapp/pkg/auth"
	"UD_telegram_miniapp/pkg/logger"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type referralQuestRoutes struct {
	qs service.SocialQuestServiceI
	a  *auth.TelegramAuth
}

func NewReferralQuestRoutes(handler *gin.RouterGroup, qs *service.SocialQuestService, a *auth.TelegramAuth) {
	r := &referralQuestRoutes{qs: qs, a: a}
	h := handler.Group("/referralquests")
	{
		admin := h.Group("/admin")
		admin.Use(a.TelegramAuthMiddleware())
		{
			admin.POST("/new", r.CreateReferralQuest)
		}
	}

	public := h.Group("/")
	public.Use(a.TelegramAuthMiddleware())
	{
		public.GET("/:telegram_id", r.GetUserQuestStatuses)
		public.GET("/:telegram_id/:quest_id", r.GetQuestStatus)
		public.POST("/:telegram_id/:quest_id/claim", r.ClaimQuest)
	}
}

type CreateReferralQuestRequest struct {
	ReferralsRequired int `json:"referrals_required" binding:"required,min=1"`
	PointReward       int `json:"point_reward" binding:"required,min=1"`
}

func (r *referralQuestRoutes) CreateReferralQuest(c *gin.Context) {
	log := logger.Logger()

	var req CreateReferralQuestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("failed to bind request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	quest := &model.ReferralQuest{
		QuestID:           uuid.New(),
		ReferralsRequired: req.ReferralsRequired,
		PointReward:       req.PointReward,
		CreatedAt:         time.Now(),
	}

	questID, err := r.qs.CreateReferralQuest(c.Request.Context(), quest)
	if err != nil {
		log.Error("failed to create referral quest", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create referral quest"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"quest_id":           questID,
		"referrals_required": req.ReferralsRequired,
		"point_reward":       req.PointReward,
	})
}

type ReferralQuestStatusResponse struct {
	QuestID           uuid.UUID  `json:"quest_id"`
	ReferralsRequired int        `json:"referrals_required"`
	PointReward       int        `json:"point_reward"`
	CurrentReferrals  int        `json:"current_referrals"`
	Completed         bool       `json:"completed"`
	ReadyToClaim      bool       `json:"ready_to_claim"`
	StartedAt         *time.Time `json:"started_at"`
	FinishedAt        *time.Time `json:"finished_at"`
}

func (r *referralQuestRoutes) GetUserQuestStatuses(c *gin.Context) {
	log := logger.Logger()

	telegramID := c.Param("telegram_id")
	id, err := strconv.ParseInt(telegramID, 10, 64)
	if err != nil {
		log.Error("failed to parse telegram_id", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid telegram_id"})
		return
	}

	statuses, err := r.qs.GetUserQuestStatuses(c.Request.Context(), id)
	if err != nil {
		log.Error("failed to get quest statuses", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get quest statuses"})
		return
	}

	out := make([]ReferralQuestStatusResponse, len(statuses))
	for i, status := range statuses {
		out[i] = ReferralQuestStatusResponse{
			QuestID:           status.QuestID,
			ReferralsRequired: status.ReferralsRequired,
			PointReward:       status.PointReward,
			CurrentReferrals:  status.CurrentReferrals,
			Completed:         status.Completed,
			ReadyToClaim:      status.ReadyToClaim,
			StartedAt:         status.StartedAt,
			FinishedAt:        status.FinishedAt,
		}
	}

	c.JSON(http.StatusOK, out)
}

func (r *referralQuestRoutes) GetQuestStatus(c *gin.Context) {
	log := logger.Logger()

	telegramID := c.Param("telegram_id")
	questIDStr := c.Param("quest_id")

	id, err := strconv.ParseInt(telegramID, 10, 64)
	if err != nil {
		log.Error("failed to parse telegram_id", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid telegram_id"})
		return
	}

	questID, err := uuid.Parse(questIDStr)
	if err != nil {
		log.Error("failed to parse quest_id", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quest_id"})
		return
	}

	status, err := r.qs.GetReferralQuestStatus(c.Request.Context(), id, questID)
	if err != nil {
		log.Error("failed to get quest status", zap.Error(err))
		if errors.Is(err, service.ErrQuestNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "quest not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get quest status"})
		return
	}

	out := ReferralQuestStatusResponse{
		QuestID:           status.QuestID,
		ReferralsRequired: status.ReferralsRequired,
		PointReward:       status.PointReward,
		CurrentReferrals:  status.CurrentReferrals,
		Completed:         status.Completed,
		ReadyToClaim:      status.ReadyToClaim,
		StartedAt:         status.StartedAt,
		FinishedAt:        status.FinishedAt,
	}

	c.JSON(http.StatusOK, out)
}

func (r *referralQuestRoutes) ClaimQuest(c *gin.Context) {
	log := logger.Logger()

	telegramID := c.Param("telegram_id")
	questIDStr := c.Param("quest_id")

	id, err := strconv.ParseInt(telegramID, 10, 64)
	if err != nil {
		log.Error("failed to parse telegram_id", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid telegram_id"})
		return
	}

	questID, err := uuid.Parse(questIDStr)
	if err != nil {
		log.Error("failed to parse quest_id", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quest_id"})
		return
	}

	if err := r.qs.ClaimReferralQuest(c.Request.Context(), id, questID); err != nil {
		log.Error("failed to claim quest", zap.Error(err))
		switch {
		case errors.Is(err, service.ErrQuestNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "quest not found"})
		case errors.Is(err, service.ErrQuestAlreadyClaimed):
			c.JSON(http.StatusConflict, gin.H{"error": "quest already claimed"})
		case errors.Is(err, service.ErrNotEnoughReferrals):
			c.JSON(http.StatusForbidden, gin.H{"error": "not enough referrals to claim this quest"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to claim quest"})
		}
		return
	}

	c.Status(http.StatusOK)
}
