package api

import (
	"fmt"
	"net/http"
	"strconv"

	"UD_telegram_miniapp/internal/model"
	"UD_telegram_miniapp/internal/service"
	"UD_telegram_miniapp/pkg/auth"
	"UD_telegram_miniapp/pkg/logger"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type userRoutes struct {
	us service.UserServiceI
	a  *auth.TelegramAuth
}

func NewUserRoutes(handler *gin.RouterGroup, us service.UserServiceI, a *auth.TelegramAuth) {
	r := &userRoutes{us: us, a: a}
	h := handler.Group("/users")
	h.Use(a.TelegramAuthMiddleware())
	{
		h.POST("/", r.RegisterUser)
		h.GET("/:telegram_id", r.GetUserByTelegramID)
		h.GET("/:telegram_id/waitlist", r.GetUserWaitlistStatus)
		h.PATCH("/:telegram_id/waitlist", r.UpdateUserWaitlistStatus)
		h.GET("/leaderboard", r.GetLeaderboard)
		h.GET("/:telegram_id/referrals", r.GetUserReferrals)

		h.GET("/:telegram_id/avatar", r.GetUserAvatar)
	}
}

type RegisterUserRequest struct {
	Handle     string `json:"handle"`
	ProfileImg string `json:"profile_img"`
	Referrer   *int64 `json:"referrer"`
}

type RegisterUserResponse struct {
	TelegramID int64  `json:"telegram_id"`
	Handle     string `json:"handle"`
}

func (r *userRoutes) RegisterUser(c *gin.Context) {
	log := logger.Logger()

	var req RegisterUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("failed to bind request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	userData, exists := c.Get("telegram_user")
	if !exists {
		log.Error("telegram user data not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	user, ok := userData.(*auth.TelegramUserData)
	if !ok {
		log.Error("invalid type assertion for telegram user data")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	u := &model.User{
		TelegramID:       user.ID,
		Handle:           req.Handle,
		Username:         user.Username,
		ReferrerID:       req.Referrer,
		ProfileImage:     req.ProfileImg,
		RegistrationDate: user.AuthDate,
	}

	err := r.us.RegisterUser(c.Request.Context(), u)
	if err != nil {
		log.Error("failed to register user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register user"})
		return
	}

	out := RegisterUserResponse{
		TelegramID: u.TelegramID,
		Handle:     u.Handle,
	}

	c.JSON(http.StatusCreated, out)
}

func (r *userRoutes) GetUserByTelegramID(c *gin.Context) {
	log := logger.Logger()

	telegramID := c.Param("telegram_id")
	id, err := strconv.ParseInt(telegramID, 10, 64)
	if err != nil {
		log.Error("failed to parse telegram_id", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid telegram_id"})
		return
	}

	user, err := r.us.GetUserByTelegramID(c.Request.Context(), id)
	if err != nil {
		log.Error("failed to get user", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "no user associated with the provided telegram_id"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"telegram_id":       user.TelegramID,
		"handle":            user.Handle,
		"username":          user.Username,
		"referrer_id":       user.ReferrerID,
		"referrals":         user.Referrals,
		"points":            user.Points,
		"profile_image":     user.ProfileImage,
		"join_waitlist":     user.JoinWaitlist,
		"registration_date": user.RegistrationDate,
		"auth_date":         user.AuthDate,
	})
}

func (r *userRoutes) GetUserWaitlistStatus(c *gin.Context) {
	log := logger.Logger()

	telegramID := c.Param("telegram_id")
	id, err := strconv.ParseInt(telegramID, 10, 64)
	if err != nil {
		log.Error("failed to parse telegram_id", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid telegram_id"})
		return
	}

	status, err := r.us.GetUserWaitlistStatus(c.Request.Context(), id)
	if err != nil {
		log.Error("failed to get user waitlist status", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "user's telegram_id not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"current_join_waitlist_status": status,
	})
}

type UpdateWaitlistRequest struct {
	JoinWaitlist bool `json:"join_waitlist"`
}

func (r *userRoutes) UpdateUserWaitlistStatus(c *gin.Context) {
	log := logger.Logger()

	telegramID := c.Param("telegram_id")
	id, err := strconv.ParseInt(telegramID, 10, 64)
	if err != nil {
		log.Error("failed to parse telegram_id", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid telegram_id"})
		return
	}

	var req UpdateWaitlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("failed to bind request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	err = r.us.UpdateUserWaitlistStatus(c.Request.Context(), id, req.JoinWaitlist)
	if err != nil {
		log.Error("failed to update user waitlist status", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "user telegram_id not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"current_join_waitlist_status": req.JoinWaitlist,
	})
}

func (r *userRoutes) GetLeaderboard(c *gin.Context) {
	log := logger.Logger()

	users, err := r.us.GetLeaderboard(c.Request.Context())
	if err != nil {
		log.Error("failed to get leaderboard", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var response []gin.H
	for _, user := range users {
		response = append(response, gin.H{
			"username":    user.Username,
			"points":      user.Points,
			"profile_img": user.ProfileImage,
			"referrals":   user.Referrals,
		})
	}

	c.JSON(http.StatusOK, response)
}

type userReferral struct {
	TelegramUsername string `json:"telegram_username"`
	ReferralCount    int    `json:"referral_count"`
	Points           int    `json:"points"`
}

func (r *userRoutes) GetUserReferrals(c *gin.Context) {
	log := logger.Logger()

	telegramID := c.Param("telegram_id")
	id, err := strconv.ParseInt(telegramID, 10, 64)
	if err != nil {
		log.Error("failed to parse telegram_id", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid telegram_id"})
		return
	}

	referrals, err := r.us.GetUserReferrals(c.Request.Context(), id)
	if err != nil {
		log.Error("failed to get user referrals", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user referrals"})
		return
	}

	out := make([]userReferral, len(referrals))
	for i, ref := range referrals {
		out[i] = userReferral{
			TelegramUsername: ref.TelegramUsername,
			ReferralCount:    ref.ReferralCount,
			Points:           ref.Points,
		}
	}

	c.JSON(http.StatusOK, out)
}

func (r *userRoutes) GetUserAvatar(c *gin.Context) {
	log := logger.Logger()

	telegramID := c.Param("telegram_id")
	id, err := strconv.ParseInt(telegramID, 10, 64)
	if err != nil {
		log.Error("failed to parse telegram_id", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid telegram_id"})
		return
	}

	_, err = r.us.GetUserByTelegramID(c.Request.Context(), id)
	if err != nil {
		log.Error("failed to get user", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	avatarFilePath, err := r.getUserAvatarURL(id)
	if err != nil {
		log.Error("failed to get user avatar",
			zap.Error(err),
			zap.Int64("telegram_id", id))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch avatar"})
		return
	}

	if avatarFilePath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "no avatar found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"avatar_file_path": avatarFilePath,
	})
}

func (r *userRoutes) getUserAvatarURL(userID int64) (string, error) {
	bot, err := tgbotapi.NewBotAPI(r.a.GetBotToken())
	if err != nil {
		return "", fmt.Errorf("failed to initialize bot: %w", err)
	}

	photos, err := bot.GetUserProfilePhotos(tgbotapi.UserProfilePhotosConfig{
		UserID: userID,
		Limit:  1,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get user photos: %w", err)
	}

	if len(photos.Photos) == 0 {
		return "", fmt.Errorf("no photo found")
	}

	file, err := bot.GetFile(tgbotapi.FileConfig{
		FileID: photos.Photos[0][0].FileID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get file: %w", err)
	}

	return file.FilePath, nil
}
