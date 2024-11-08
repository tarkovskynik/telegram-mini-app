package auth

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"UD_telegram_miniapp/pkg/logger"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	initdata "github.com/telegram-mini-apps/init-data-golang"
)

const expTime = 24 * time.Hour

type TelegramAuth struct {
	botToken  string
	debugMode bool
}

func NewTelegramAuth(botToken string, debugMode bool) *TelegramAuth {
	return &TelegramAuth{
		botToken:  botToken,
		debugMode: debugMode,
	}
}

func (t *TelegramAuth) TelegramAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		log := logger.Logger()

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Info("missing authorization header")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header is required"})
			return
		}

		if !strings.HasPrefix(authHeader, "Telegram ") {
			log.Info("invalid authorization header format")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}

		initData := strings.TrimPrefix(authHeader, "Telegram ")
		if !t.debugMode {
			if err := initdata.Validate(initData, t.botToken, expTime); err != nil {
				log.Info("invalid telegram init data", zap.Error(err))
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid telegram auth data"})
				return
			}
		}

		telegramUserData, err := ExtractTelegramData(initData)
		if err != nil {
			log.Error("failed to extract telegram data", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid telegram data"})
			return
		}

		c.Set("telegram_user", telegramUserData)
		c.Next()
	}
}

func (t *TelegramAuth) GetBotToken() string {
	return t.botToken
}

type TelegramUserData struct {
	ID       int64
	Username string
	AuthDate time.Time
}

func ExtractTelegramData(initData string) (*TelegramUserData, error) {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return nil, err
	}

	authDateUnix, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if err != nil {
		return nil, err
	}

	authDate := time.Unix(authDateUnix, 0)

	var userData struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
	}

	if err := json.Unmarshal([]byte(values.Get("user")), &userData); err != nil {
		return nil, err
	}

	return &TelegramUserData{
		ID:       userData.ID,
		Username: userData.Username,
		AuthDate: authDate,
	}, nil
}
