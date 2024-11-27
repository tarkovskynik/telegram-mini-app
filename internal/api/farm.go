package api

import (
	"context"
	"net/http"

	"UD_telegram_miniapp/internal/repository"
	"UD_telegram_miniapp/pkg/auth"
	"UD_telegram_miniapp/pkg/logger"

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
}

type StatusResponse struct {
	CanHarvest       bool    `json:"canHarvest"`
	TimeUntilHarvest float64 `json:"timeUntilHarvest"`
	Points           int     `json:"points"`
}

type HarvestResponse struct {
	Success bool   `json:"success"`
	Points  int    `json:"points"`
	Error   string `json:"error,omitempty"`
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
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
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

	user, err := r.repo.GetUserByTelegramID(context.TODO(), u.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get points"})
		return
	}

	_, exist := r.repo.Cache[u.ID]
	if !exist {
		c.JSON(http.StatusOK, StatusResponse{
			CanHarvest:       status.CanHarvest,
			TimeUntilHarvest: status.TimeUntilHarvest,
		})
		return
	}

	c.JSON(http.StatusOK, StatusResponse{
		CanHarvest:       status.CanHarvest,
		TimeUntilHarvest: status.TimeUntilHarvest,
		Points:           user.Points,
	})
}
