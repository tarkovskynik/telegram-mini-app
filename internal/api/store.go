package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"UD_telegram_miniapp/pkg/auth"
	"UD_telegram_miniapp/pkg/logger"

	"github.com/gin-gonic/gin"
)

type storeRoutes struct {
	a *auth.TelegramAuth
}

func NewStoreRoutes(handler *gin.RouterGroup, a *auth.TelegramAuth) {
	r := &storeRoutes{a: a}
	h := handler.Group("/store")
	h.Use(a.TelegramAuthMiddleware())

	h.POST("/energy-recharge", r.EnergyRechargeHandler)

}

func (r *storeRoutes) EnergyRechargeHandler(c *gin.Context) {
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

	request := CreateInvoiceLinkRequest{
		Title:         "Energy Recharge",
		Description:   "...",
		Payload:       fmt.Sprintf("%d", user.ID),
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
