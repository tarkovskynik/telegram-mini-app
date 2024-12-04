package service

import (
	"context"
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type PaymentConfig struct {
	BotToken string
	Debug    bool
}

type PaymentService struct {
	bot *tgbotapi.BotAPI
}

type Payment struct {
	UserID        int64
	Amount        int
	Currency      string
	ChargeID      string
	PaymentStatus string
}

type NotificationWS struct {
	UserID           int64
	NotificationChan chan Message
	//Mu               sync.Mutex
}

type Message struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload,omitempty"`
}

func NewPaymentService(config PaymentConfig) (*PaymentService, error) {
	bot, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize bot: %w", err)
	}

	bot.Debug = config.Debug

	return &PaymentService{
		bot: bot,
	}, nil
}

func (s *PaymentService) HandlePreCheckoutQuery(query *tgbotapi.PreCheckoutQuery) error {
	answer := tgbotapi.PreCheckoutConfig{
		PreCheckoutQueryID: query.ID,
		OK:                 true,
	}

	_, err := s.bot.Send(answer)
	return err
}

func (s *PaymentService) HandleSuccessfulPayment(ctx context.Context, msg *tgbotapi.Message, paymentNotifications *NotificationWS) error {
	payment := msg.SuccessfulPayment

	confirmation := tgbotapi.NewMessage(msg.Chat.ID, "Thank you for your payment!")
	_, err := s.bot.Send(confirmation)
	if err != nil {
		return err
	}

	switch payment.InvoicePayload {
	case "ENERGY_RECHARGE":
		mess := Message{
			Type: "ENERGY_RECHARGE_SUCCESS",
			Payload: map[string]any{
				"user_id": msg.From.ID,
			},
		}

		paymentNotifications.NotificationChan <- mess

		fmt.Printf("SUCCESSFUL_PAYMENT:\n%+v\n", payment)
	}

	return nil
}

func (s *PaymentService) StartPaymentListener(ctx context.Context, paymentNotifications *NotificationWS) {
	defer func() {
		close(paymentNotifications.NotificationChan)
	}()

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := s.bot.GetUpdatesChan(updateConfig)

	for {
		select {
		case update := <-updates:
			switch {
			case update.PreCheckoutQuery != nil:
				if err := s.HandlePreCheckoutQuery(update.PreCheckoutQuery); err != nil {
					log.Printf("Failed to handle pre-checkout query: %v", err)
				}

			case update.Message.SuccessfulPayment != nil:
				if err := s.HandleSuccessfulPayment(ctx, update.Message, paymentNotifications); err != nil {
					log.Printf("Failed to handle successful payment: %v", err)
				}
			}

		case <-ctx.Done():
			return
		}
	}
}
