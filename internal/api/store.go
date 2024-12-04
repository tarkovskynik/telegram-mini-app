package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"UD_telegram_miniapp/internal/repository"
	"UD_telegram_miniapp/internal/service"
	"UD_telegram_miniapp/pkg/auth"
	"UD_telegram_miniapp/pkg/logger"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type storeRoutes struct {
	a     *auth.TelegramAuth
	store *service.PaymentService
	user  *repository.Repository
}

func NewStoreRoutes(handler *gin.RouterGroup, a *auth.TelegramAuth, store *service.PaymentService, user *repository.Repository) {
	r := &storeRoutes{
		a:     a,
		store: store,
		user:  user,
	}

	h := handler.Group("/store")
	h.Use(a.TelegramAuthMiddleware())

	h.POST("/energy-recharge", r.EnergyRechargeHandler)
	h.GET("/ws", r.handleWebSocket)
}

func (r *storeRoutes) EnergyRechargeHandler(c *gin.Context) {
	log := logger.Logger()

	userData, exists := c.Get("telegram_user")
	if !exists {
		log.Error("telegram user data not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	_, ok := userData.(*auth.TelegramUserData)
	if !ok {
		log.Error("invalid type assertion for telegram user data")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	request := CreateInvoiceLinkRequest{
		Title:         "Energy Recharge",
		Description:   "...",
		Payload:       "ENERGY_RECHARGE",
		ProviderToken: "",
		Currency:      "XTR",
		Prices: []LabeledPrice{
			{
				Label:  "Product",
				Amount: 1,
			},
		},
	}

	invoiceLink, err := createInvoiceLink(r.a.GetBotToken(), request)
	if err != nil {
		fmt.Printf("Error creating invoice link: %v\n", err)
		return
	}

	out := struct {
		Link string `json:"link"`
	}{
		invoiceLink,
	}

	c.JSON(http.StatusOK, out)

	fmt.Printf("Invoice link created: %s\n", invoiceLink)
}

func createInvoiceLink(botToken string, request CreateInvoiceLinkRequest) (string, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/createInvoiceLink", botToken)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	var result struct {
		Ok          bool   `json:"ok"`
		Result      string `json:"result"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("error parsing response: %w", err)
	}

	if !result.Ok {
		return "", fmt.Errorf("telegram API error: %s", result.Description)
	}

	return result.Result, nil
}

type LabeledPrice struct {
	Label  string `json:"label"`
	Amount int    `json:"amount"`
}

type CreateInvoiceLinkRequest struct {
	Title         string         `json:"title"`
	Description   string         `json:"description"`
	Payload       string         `json:"payload"`
	ProviderToken string         `json:"provider_token"`
	Currency      string         `json:"currency"`
	Prices        []LabeledPrice `json:"prices"`
	PhotoURL      string         `json:"photo_url,omitempty"`
	PhotoSize     int            `json:"photo_size,omitempty"`
	PhotoWidth    int            `json:"photo_width,omitempty"`
	PhotoHeight   int            `json:"photo_height,omitempty"`
	NeedName      bool           `json:"need_name,omitempty"`
	NeedEmail     bool           `json:"need_email,omitempty"`
	NeedPhone     bool           `json:"need_phone_number,omitempty"`
	NeedAddress   bool           `json:"need_shipping_address,omitempty"`
	IsFlexible    bool           `json:"is_flexible,omitempty"`
}

func (r *storeRoutes) handleWebSocket(c *gin.Context) {
	log := logger.Logger()

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

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("websocket upgrade failed", zap.Error(err))
		return
	}

	paymentNotifications := &service.NotificationWS{
		UserID:           user.ID,
		NotificationChan: make(chan service.Message),
	}

	go r.store.StartPaymentListener(context.TODO(), paymentNotifications)
	go r.PaymentNotificationsLoop(conn, paymentNotifications)
}

func (r *storeRoutes) PaymentNotificationsLoop(conn *websocket.Conn, paymentNotifications *service.NotificationWS) {
	defer func() {
		conn.Close()
	}()

	for message := range paymentNotifications.NotificationChan {
		if message.Type == "ENERGY_RECHARGE_SUCCESS" {
			err := r.user.ResetEnergy(context.TODO(), paymentNotifications.UserID)
			if err != nil {
				fmt.Printf("Error resetting energy recharge: %v\n", err)
			}

			out, err := json.MarshalIndent(message, "", "	")
			if err != nil {
				fmt.Println(fmt.Errorf("error marshaling response: %w", err))
			}

			err = conn.WriteMessage(websocket.TextMessage, out)
			if err != nil {
				fmt.Println(fmt.Errorf("error sending response: %w", err))
			}
		}
	}
}
