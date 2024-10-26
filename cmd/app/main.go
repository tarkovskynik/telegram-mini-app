package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"UD_telegram_miniapp/internal/api"
	"UD_telegram_miniapp/internal/repository"
	"UD_telegram_miniapp/internal/service"
	"UD_telegram_miniapp/pkg/auth"
	"UD_telegram_miniapp/pkg/logger"
	"go.uber.org/zap"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	fmt.Println("config main: ", cfg)

	err = logger.Initialize(cfg.LogLevel)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()
	zapLogger := logger.Logger()

	repo, err := repository.New(cfg.Database)
	if err != nil {
		zapLogger.Fatal("Failed to initialize repository", zap.Error(err))
	}
	defer repo.Close()

	userService := service.NewUserService(repo)
	fmt.Println("token main: ", cfg.TelegramAuth.TelegramBotToken)
	telegramAuth := auth.NewTelegramAuth(cfg.TelegramAuth.TelegramBotToken)

	router := gin.New()
	router.Use(gin.Recovery())

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{
		http.MethodHead,
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
	}
	config.AllowHeaders = []string{"*"}
	config.AllowCredentials = true
	config.MaxAge = 12 * time.Hour

	router.Use(cors.New(config))

	a := router.Group("/api/v1")
	api.NewUserRoutes(a, userService, telegramAuth)

	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	zapLogger.Info("Starting server", zap.String("addr", addr))
	if err := router.Run(addr); err != nil {
		zapLogger.Fatal("Failed to start server", zap.Error(err))
	}
}
