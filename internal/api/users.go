package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

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

	public := h.Group("")
	public.Use(a.TelegramAuthMiddleware())
	{
		public.POST("", r.RegisterUser)
		public.GET("/me", r.GetOwnProfile)
		public.GET("/leaderboard", r.GetLeaderboard)
		public.GET("/me/referrals", r.GetUserReferrals)
	}

	avatars := h.Group("")
	{
		avatars.GET("/:requesting_user_id/avatar/:target_user_id/:timestamp/:hash",
			VerifyAvatarURLMiddleware(),
			r.GetUserAvatar)
	}

	admin := h.Group("/admin")
	{
		admin.GET("/:telegram_id/waitlist", r.GetUserWaitlistStatus)
		admin.PATCH("/:telegram_id/waitlist", r.UpdateUserWaitlistStatus)
	}

}

type RegisterUserRequest struct {
	Handle   string `json:"handle"`
	Referrer *int64 `json:"referrer"`
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

type UserResponse struct {
	TelegramID       int64     `json:"telegram_id"`
	Handle           string    `json:"handle"`
	Username         string    `json:"username"`
	ReferrerID       *int64    `json:"referrer_id,omitempty"`
	Referrals        int       `json:"referrals"`
	Points           int       `json:"points"`
	AvatarProxyPath  string    `json:"avatar_proxy_path"`
	JoinWaitlist     *bool     `json:"join_waitlist,omitempty"`
	RegistrationDate time.Time `json:"registration_date"`
	AuthDate         time.Time `json:"auth_date"`
}

func (r *userRoutes) GetOwnProfile(c *gin.Context) {
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

	user, err := r.us.GetUserByTelegramID(c.Request.Context(), u.ID)
	if err != nil {
		log.Error("failed to get user", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "no user associated with the provided telegram_id"})
		return
	}
	user.AvatarProxyPath = GenerateAvatarURL(u.ID, u.ID)

	out := UserResponse{
		TelegramID:       user.TelegramID,
		Handle:           user.Handle,
		Username:         user.Username,
		ReferrerID:       user.ReferrerID,
		Referrals:        user.Referrals,
		Points:           user.Points,
		AvatarProxyPath:  user.AvatarProxyPath,
		JoinWaitlist:     user.JoinWaitlist,
		RegistrationDate: user.RegistrationDate,
		AuthDate:         user.AuthDate,
	}

	c.JSON(http.StatusOK, out)
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

type LeaderboardEntry struct {
	Username        string `json:"username"`
	AvatarProxyPath string `json:"avatar_proxy_path"`
	Points          int    `json:"points"`
	Referrals       int    `json:"referrals"`
}

func (r *userRoutes) GetLeaderboard(c *gin.Context) {
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

	users, err := r.us.GetLeaderboard(c.Request.Context())
	if err != nil {
		log.Error("failed to get leaderboard", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	out := make([]LeaderboardEntry, len(users))
	for i, user := range users {
		user.AvatarProxyPath = GenerateAvatarURL(u.ID, user.TelegramID)

		out[i] = LeaderboardEntry{
			Username:        user.Username,
			AvatarProxyPath: user.AvatarProxyPath,
			Points:          user.Points,
			Referrals:       user.Referrals,
		}
	}

	c.JSON(http.StatusOK, out)
}

type userReferral struct {
	TelegramUsername string `json:"telegram_username"`
	ReferralCount    int    `json:"referral_count"`
	Points           int    `json:"points"`
	AvatarProxyPath  string `json:"avatar_proxy_path"`
}

func (r *userRoutes) GetUserReferrals(c *gin.Context) {
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

	referrals, err := r.us.GetUserReferrals(c.Request.Context(), u.ID)
	if err != nil {
		log.Error("failed to get user referrals", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user referrals"})
		return
	}

	out := make([]userReferral, len(referrals))
	for i, ref := range referrals {
		ref.AvatarProxyPath = GenerateAvatarURL(u.ID, ref.TelegramID)

		out[i] = userReferral{
			TelegramUsername: ref.TelegramUsername,
			ReferralCount:    ref.ReferralCount,
			Points:           ref.Points,
			AvatarProxyPath:  ref.AvatarProxyPath,
		}
	}

	c.JSON(http.StatusOK, out)
}

func (r *userRoutes) GetUserAvatar(c *gin.Context) {
	log := logger.Logger()

	targetID := c.MustGet("target_user_id").(int64)

	_, err := r.us.GetUserByTelegramID(c.Request.Context(), targetID)
	if err != nil {
		log.Error("failed to get user", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	avatarLink, err := r.prepareAvatarFileLink(targetID)
	if err != nil {
		log.Error("failed to get user avatar",
			zap.Error(err),
			zap.Int64("telegram_id", targetID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch avatar"})
		return
	}

	if avatarLink == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "no avatar found"})
		return
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(avatarLink)
	if err != nil {
		log.Error("failed to download avatar",
			zap.Error(err),
			zap.String("url", avatarLink))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to download avatar"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error("failed to download avatar: non-200 status code",
			zap.Int("status_code", resp.StatusCode),
			zap.String("url", avatarLink))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to download avatar"})
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=86400")

	_, err = io.Copy(c.Writer, resp.Body)
	if err != nil {
		log.Error("failed to stream avatar data",
			zap.Error(err),
			zap.Int64("telegram_id", targetID))
		return
	}
}

func (r *userRoutes) prepareAvatarFileLink(userID int64) (string, error) {
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

	fullLink := file.Link(r.a.GetBotToken())

	return fullLink, nil
}

func VerifyAvatarURLMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		log := logger.Logger()

		requestingUserID := c.Param("requesting_user_id")
		reqID, err := strconv.ParseInt(requestingUserID, 10, 64)
		if err != nil {
			log.Error("failed to parse requesting_user_id", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid requesting_user_id"})
			return
		}

		targetUserID := c.Param("target_user_id")
		targetID, err := strconv.ParseInt(targetUserID, 10, 64)
		if err != nil {
			log.Error("failed to parse target_user_id", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid target_user_id"})
			return
		}

		timestamp := c.Param("timestamp")
		ts, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil {
			log.Error("failed to parse timestamp", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid timestamp"})
			return
		}

		hash := c.Param("hash")

		if !VerifyAvatarURL(reqID, targetID, ts, hash) {
			log.Error("invalid avatar URL signature",
				zap.Int64("requesting_user", reqID),
				zap.Int64("target_user", targetID))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
			return
		}

		c.Set("requesting_user_id", reqID)
		c.Set("target_user_id", targetID)

		c.Next()
	}
}

var secretKey = "secret"

func VerifyAvatarURL(requestingUserID, targetUserID int64, timestamp int64, receivedHash string) bool {
	dataToHash := fmt.Sprintf("%d:%d:%d", requestingUserID, targetUserID, timestamp)

	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(dataToHash))
	expectedHash := hex.EncodeToString(h.Sum(nil))

	return receivedHash == expectedHash
}

func GenerateAvatarURL(requestingUserID, targetUserID int64) string {
	timestamp := time.Now().Unix()
	dataToHash := fmt.Sprintf("%d:%d:%d", requestingUserID, targetUserID, timestamp)

	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(dataToHash))
	hash := hex.EncodeToString(h.Sum(nil))

	return fmt.Sprintf("/api/v1/users/%d/avatar/%d/%d/%s",
		requestingUserID,
		targetUserID,
		timestamp,
		hash)
}
