package middleware

import (
	"net/http"

	"UD_telegram_miniapp/internal/service"
	"UD_telegram_miniapp/pkg/auth"
	"UD_telegram_miniapp/pkg/logger"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

type Authorization struct {
	userService service.UserServiceI
}

func NewAuthorization(userService service.UserServiceI) *Authorization {
	return &Authorization{
		userService: userService,
	}
}

func (a *Authorization) AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		log := logger.Logger()

		userData, exists := c.Get("telegram_user")
		if !exists {
			log.Error("telegram user data not found in context")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		telegramUser, ok := userData.(*auth.TelegramUserData)
		if !ok {
			log.Error("invalid type assertion for telegram user data")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		user, err := a.userService.GetUserByTelegramID(c.Request.Context(), telegramUser.ID)
		if err != nil {
			log.Error("failed to get user data", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		if !user.IsAdmin {
			log.Info("unauthorized access attempt to admin endpoint",
				zap.Int64("telegram_id", telegramUser.ID))
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}

		c.Set("is_admin", true)
		c.Next()
	}
}
