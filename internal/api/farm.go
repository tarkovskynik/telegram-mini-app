package api

import (
	"net/http"
	"strings"

	"UD_telegram_miniapp/internal/repository"
	"UD_telegram_miniapp/pkg/auth"
	"UD_telegram_miniapp/pkg/logger"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

type farmGameRoutes struct {
	repo *repository.Repository
	a    *auth.TelegramAuth
}

func NewFarmGameRoutes(handler *gin.RouterGroup, repo *repository.Repository, a *auth.TelegramAuth) {
	r := &farmGameRoutes{repo: repo, a: a}
	h := handler.Group("/farm")
	h.Use(a.TelegramAuthMiddleware())

	h.POST("/harvest", r.startHarvest)
	h.GET("/status", r.status)
	h.PATCH("/claim", r.claim)
}

type StatusResponse struct {
	IsInProgress      bool  `json:"is_in_progress"`
	StartedAtUnix     int64 `json:"started_at_unix"`
	PointReward       int   `json:"point_reward"`
	IsPreviousClaimed bool  `json:"is_previous_claimed"`
}

type HarvestResponse struct {
	Error string `json:"error"`
}

func (r *farmGameRoutes) startHarvest(c *gin.Context) {
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

	err := r.repo.StartHarvest(u.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"error": nil,
	})

}

func (r *farmGameRoutes) status(c *gin.Context) {
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

	status, err := r.repo.Status(u.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get status"})
		return
	}

	c.JSON(http.StatusOK, StatusResponse{
		IsInProgress:      status.IsInProgress,
		StartedAtUnix:     status.StartedAt.Time.Unix(),
		PointReward:       status.PointReward,
		IsPreviousClaimed: status.IsPreviousClaimed,
	})
}

func (r *farmGameRoutes) claim(c *gin.Context) {
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

	points, err := r.repo.ClaimPoints(u.ID)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "no farming session found"):
			c.JSON(http.StatusBadRequest, gin.H{"error": "No farming session found"})
		case strings.Contains(err.Error(), "farming session not yet complete"):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case strings.Contains(err.Error(), "reward already claimed"):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Reward already claimed"})
		default:
			log.Error("failed to claim points", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to claim points"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"points_earned": points,
		"message":       "Successfully claimed points",
	})
}
