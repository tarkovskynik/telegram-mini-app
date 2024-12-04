package main

import (
	"fmt"
	"log"

	"UD_telegram_miniapp/internal/api"
	"UD_telegram_miniapp/internal/repository"
	"UD_telegram_miniapp/internal/service"
	"UD_telegram_miniapp/pkg/auth"
	"UD_telegram_miniapp/pkg/logger"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

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
	dailyQuestService := service.NewDailyQuestService(repo)
	socialQuestService := service.NewSocialQuestService(repo)

	telegramAuth := auth.NewTelegramAuth(cfg.TelegramAuth.TelegramBotToken, cfg.Server.DebugMode)

	router := gin.New()
	router.Use(gin.Recovery())
	if cfg.Server.DebugMode {
		router.Use(gin.Logger())
	}

	//config := cors.DefaultConfig()
	//config.AllowAllOrigins = true
	//config.AllowMethods = []string{
	//	http.MethodHead,
	//	http.MethodGet,
	//	http.MethodPost,
	//	http.MethodPut,
	//	http.MethodPatch,
	//	http.MethodDelete,
	//}
	//config.AllowHeaders = []string{"*"}
	//config.AllowCredentials = true
	//config.MaxAge = 12 * time.Hour
	//
	//router.Use(cors.New(config))

	a := router.Group("/api/v1")
	api.NewUserRoutes(a, userService, telegramAuth)
	api.NewDailyQuestRoutes(a, dailyQuestService, telegramAuth)
	api.NewSocialQuestRoutes(a, socialQuestService, telegramAuth)
	api.NewReferralQuestRoutes(a, socialQuestService, telegramAuth)

	//game
	api.NewGameRoutes(a, repo, telegramAuth)

	//farm game
	api.NewFarmGameRoutes(a, repo, telegramAuth)

	//store
	conf := service.PaymentConfig{
		BotToken: cfg.TelegramAuth.TelegramBotToken,
		Debug:    true,
	}
	store, err := service.NewPaymentService(conf)
	if err != nil {
		zapLogger.Fatal("Failed to initialize payment service", zap.Error(err))
	}
	api.NewStoreRoutes(a, telegramAuth, store, repo)

	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	zapLogger.Info("Starting server", zap.String("addr", addr))
	if err := router.Run(addr); err != nil {
		zapLogger.Fatal("Failed to start server", zap.Error(err))
	}
}
