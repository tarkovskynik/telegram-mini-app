package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"UD_telegram_miniapp/internal/model"
	"UD_telegram_miniapp/internal/repository"
	"UD_telegram_miniapp/internal/service"
	"UD_telegram_miniapp/pkg/auth"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type socialQuestRoutes struct {
	qs service.SocialQuestServiceI
	a  *auth.TelegramAuth
}

func NewSocialQuestRoutes(handler *gin.RouterGroup, qs *service.SocialQuestService, a *auth.TelegramAuth) {
	h := &socialQuestRoutes{qs: qs, a: a}

	quests := handler.Group("/socialquests")
	{
		public := quests.Group("/")
		public.Use(a.TelegramAuthMiddleware())
		{
			public.GET("/:telegram_id", h.GetUserQuests)
			public.GET("/:telegram_id/:social_quest_id", h.GetQuestByID)
			public.POST("/:telegram_id/:social_quest_id", h.ClaimQuest)
		}

		admin := quests.Group("/admin")
		admin.Use(a.TelegramAuthMiddleware())
		{
			admin.POST("/new", h.CreateSocialQuest)
			admin.POST("/validations/new", h.CreateValidationKind)
			admin.GET("/validations", h.ListValidationKinds)
			admin.POST("/validations/:quest_id/:validation_id", h.AddQuestValidation)
			admin.DELETE("/validations/:quest_id/:validation_id", h.RemoveQuestValidation)
		}
	}
}

type questResponse struct {
	QuestID     string `json:"quest_id"`
	QuestType   string `json:"quest_type"`
	ActionType  string `json:"action_type"`
	Image       string `json:"img"`
	Title       string `json:"title"`
	Description string `json:"description"`
	PointReward int    `json:"point_reward"`
	Link        string `json:"link"`
	ChatID      int64  `json:"chat_id"`

	Completed  bool   `json:"completed"`
	StartedAt  *int64 `json:"started_at"`
	FinishedAt *int64 `json:"finished_at"`

	ValidationsRequired []QuestValidation `json:"validations_required"`
	ValidationsComplete []QuestValidation `json:"validations_complete"`
}

type QuestValidation struct {
	ValidationID   int    `json:"validation_id"`
	ValidationName string `json:"validation_name"`
}

func (h *socialQuestRoutes) GetUserQuests(c *gin.Context) {
	telegramID := c.Param("telegram_id")
	id, err := strconv.ParseInt(telegramID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid telegram_id"})
		return
	}

	quests, userQuests, validationStatus, err := h.qs.GetUserQuests(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "telegram_id not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	userQuestMap := make(map[uuid.UUID]*model.UserSocialQuest)
	for _, uq := range userQuests {
		userQuestMap[uq.QuestID] = uq
	}

	response := make([]questResponse, len(quests))
	for i, quest := range quests {
		userQuest := userQuestMap[quest.QuestID]

		validationsRequired := make([]QuestValidation, len(quest.Validations))
		for j, v := range quest.Validations {
			validationsRequired[j] = QuestValidation{
				ValidationID:   v.ValidationID,
				ValidationName: v.ValidationName,
			}
		}

		completedValidations := make([]QuestValidation, 0)
		for _, v := range quest.Validations {
			if _, ok := validationStatus[v]; ok {
				completedValidations = append(completedValidations, QuestValidation{
					ValidationID:   v.ValidationID,
					ValidationName: v.ValidationName,
				})
			}
		}

		var started, finished *int64
		if userQuest.StartedAt != nil {
			unix := userQuest.StartedAt.Unix()
			started = &unix
		}
		if userQuest.FinishedAt != nil {
			unix := userQuest.FinishedAt.Unix()
			finished = &unix
		}

		response[i] = questResponse{
			QuestID:             quest.QuestID.String(),
			QuestType:           quest.QuestType.Name,
			ActionType:          quest.ActionType.Name,
			Image:               quest.Image,
			Title:               quest.Title,
			Description:         quest.Description,
			PointReward:         quest.PointReward,
			Completed:           userQuest.Completed,
			StartedAt:           started,
			FinishedAt:          finished,
			Link:                quest.Link,
			ChatID:              quest.ChatID,
			ValidationsRequired: validationsRequired,
			ValidationsComplete: completedValidations,
		}
	}

	c.JSON(http.StatusOK, response)
}

func (h *socialQuestRoutes) GetQuestByID(c *gin.Context) {
	telegramID := c.Param("telegram_id")
	questIDStr := c.Param("social_quest_id")

	id, err := strconv.ParseInt(telegramID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid telegram_id"})
		return
	}

	questID, err := uuid.Parse(questIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid social_quest_id"})
		return
	}

	quest, userQuest, validationStatus, err := h.qs.GetQuestByID(c.Request.Context(), id, questID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrQuestNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "social_quest_id not found"})
		case errors.Is(err, service.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "telegram_id not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	validationsRequired := make([]QuestValidation, len(quest.Validations))
	for i, v := range quest.Validations {
		validationsRequired[i] = QuestValidation{
			ValidationID:   v.ValidationID,
			ValidationName: v.ValidationName,
		}
	}

	completedValidations := make([]QuestValidation, 0)
	for _, v := range quest.Validations {
		if _, ok := validationStatus[v]; ok {
			completedValidations = append(completedValidations, QuestValidation{
				ValidationID:   v.ValidationID,
				ValidationName: v.ValidationName,
			})
		}
	}

	var started, finished *int64
	if userQuest.StartedAt != nil {
		unix := userQuest.StartedAt.Unix()
		started = &unix
	}
	if userQuest.FinishedAt != nil {
		unix := userQuest.FinishedAt.Unix()
		finished = &unix
	}

	response := questResponse{
		QuestID:             quest.QuestID.String(),
		QuestType:           quest.QuestType.Name,
		ActionType:          quest.ActionType.Name,
		Image:               quest.Image,
		Title:               quest.Title,
		Description:         quest.Description,
		PointReward:         quest.PointReward,
		Completed:           userQuest.Completed,
		StartedAt:           started,
		FinishedAt:          finished,
		Link:                quest.Link,
		ChatID:              quest.ChatID,
		ValidationsRequired: validationsRequired,
		ValidationsComplete: completedValidations,
	}

	c.JSON(http.StatusOK, response)
}

func (h *socialQuestRoutes) ClaimQuest(c *gin.Context) {
	telegramID := c.Param("telegram_id")
	questIDStr := c.Param("social_quest_id")

	id, err := strconv.ParseInt(telegramID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid telegram_id"})
		return
	}

	questID, err := uuid.Parse(questIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid social_quest_id"})
		return
	}

	err = h.qs.ClaimQuest(c.Request.Context(), id, questID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrQuestNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "social_quest_id not found"})
		case errors.Is(err, service.ErrQuestNotStarted):
			c.JSON(http.StatusForbidden, gin.H{"error": "Validation condition not met"})
		case errors.Is(err, service.ErrQuestAlreadyClaimed):
			c.JSON(http.StatusForbidden, gin.H{"error": "quest already claimed"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.Status(http.StatusOK)
}

type CreateSocialQuestRequest struct {
	Image         string `json:"img" binding:"required"`
	Title         string `json:"title" binding:"required"`
	Description   string `json:"description"`
	PointReward   int    `json:"point_reward" binding:"required,min=1"`
	ValidationIDs []int  `json:"validation_id_list"`
	QuestTypeID   int    `json:"quest_type_id" binding:"required"`
	ActionTypeID  int    `json:"action_type_id" binding:"required"`
	Link          string `json:"link"`
	ChatID        int64  `json:"chat_id"`
}

func (h *socialQuestRoutes) CreateSocialQuest(c *gin.Context) {
	var req CreateSocialQuestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	validations := make([]model.QuestValidation, 0)
	for _, id := range req.ValidationIDs {
		qv := model.QuestValidation{ValidationID: id}
		validations = append(validations, qv)
	}

	quest := &model.SocialQuest{
		QuestID:     uuid.New(),
		Image:       req.Image,
		Title:       req.Title,
		Description: req.Description,
		PointReward: req.PointReward,
		Validations: validations,
		QuestType: model.QuestType{
			ID: req.QuestTypeID,
		},
		ActionType: model.ActionType{
			ID: req.ActionTypeID,
		},
		Link:   req.Link,
		ChatID: req.ChatID,
	}

	questID, err := h.qs.CreateSocialQuest(c.Request.Context(), quest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"social_quest_id": questID,
	})
}

type CreateValidationKindRequest struct {
	ValidationName string `json:"validation_name" binding:"required"`
}

type ValidationKindResponse struct {
	ValidationID   int    `json:"validation_id"`
	ValidationName string `json:"validation_name"`
}

func (h *socialQuestRoutes) CreateValidationKind(c *gin.Context) {
	var req CreateValidationKindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	validationKind := &model.QuestValidationKind{
		ValidationName: strings.ToUpper(req.ValidationName),
	}

	err := h.qs.CreateValidationKind(c.Request.Context(), validationKind)
	if err != nil {
		if errors.Is(err, service.ErrValidationNameExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "validation name already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{})
}

func (h *socialQuestRoutes) ListValidationKinds(c *gin.Context) {
	validations, err := h.qs.ListValidationKinds(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := make([]ValidationKindResponse, len(validations))
	for i, v := range validations {
		response[i] = ValidationKindResponse{
			ValidationID:   v.ValidationID,
			ValidationName: v.ValidationName,
		}
	}

	c.JSON(http.StatusOK, response)
}

type AddValidationRequest struct {
	ValidationID int `json:"validation_id" binding:"required"`
}

func (h *socialQuestRoutes) AddQuestValidation(c *gin.Context) {
	questID := c.Param("quest_id")
	qID, err := uuid.Parse(questID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quest_id"})
		return
	}

	validationIDStr := c.Param("validation_id")
	validationID, err := strconv.Atoi(validationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid validation_id"})
		return
	}

	err = h.qs.AddQuestValidation(c.Request.Context(), qID, validationID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrQuestNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "quest not found"})
		case errors.Is(err, repository.ErrValidationNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "validation kind not found"})
		case errors.Is(err, repository.ErrValidationAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{"error": "validation already assigned to quest"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add validation"})
		}
		return
	}

	c.Status(http.StatusCreated)
}

func (h *socialQuestRoutes) RemoveQuestValidation(c *gin.Context) {
	questID := c.Param("quest_id")
	qID, err := uuid.Parse(questID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quest_id"})
		return
	}

	validationID := c.Param("validation_id")
	vID, err := strconv.Atoi(validationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid validation_id"})
		return
	}

	err = h.qs.RemoveQuestValidation(c.Request.Context(), qID, vID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrQuestNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "quest not found"})
		case errors.Is(err, service.ErrValidationNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "validation not found for this quest"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove validation"})
		}
		return
	}

	c.Status(http.StatusOK)
}
