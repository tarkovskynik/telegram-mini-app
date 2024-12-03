package service

//
//import (
//	"context"
//	"fmt"
//	"log"
//
//	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
//)
//
//type PaymentConfig struct {
//	BotToken string
//	Debug    bool
//}
//
//type PaymentService struct {
//	bot  *tgbotapi.BotAPI
//	repo PaymentRepository
//}
//
//type PaymentRepository interface {
//	SavePayment(ctx context.Context, payment *Payment) error
//	RefundPayment(ctx context.Context, paymentID string) error
//}
//
//type Payment struct {
//	UserID        int64
//	Amount        int
//	Currency      string
//	ChargeID      string
//	PaymentStatus string
//}
//
//func NewPaymentService(config PaymentConfig, repo PaymentRepository) (*PaymentService, error) {
//	bot, err := tgbotapi.NewBotAPI(config.BotToken)
//	if err != nil {
//		return nil, fmt.Errorf("failed to initialize bot: %w", err)
//	}
//
//	bot.Debug = config.Debug
//
//	return &PaymentService{
//		bot:  bot,
//		repo: repo,
//	}, nil
//}
//
//func (s *PaymentService) HandlePreCheckoutQuery(query *tgbotapi.PreCheckoutQuery) error {
//	answer := tgbotapi.PreCheckoutConfig{
//		PreCheckoutQueryID: query.ID,
//		OK:                 true,
//	}
//	_, err := s.bot.Send(answer)
//	return err
//}
//
//func (s *PaymentService) HandleSuccessfulPayment(ctx context.Context, msg *tgbotapi.Message) error {
//	payment := msg.SuccessfulPayment
//
//	// Save payment to database
//	err := s.repo.SavePayment(ctx, &Payment{
//		UserID:        msg.From.ID,
//		Amount:        payment.TotalAmount,
//		Currency:      payment.Currency,
//		ChargeID:      payment.TelegramPaymentChargeID,
//		PaymentStatus: "completed",
//	})
//	if err != nil {
//		return fmt.Errorf("failed to save payment: %w", err)
//	}
//
//	// Send confirmation message
//	confirmation := tgbotapi.NewMessage(msg.Chat.ID, "Thank you for your payment!")
//	_, err = s.bot.Send(confirmation)
//	return err
//}
//
//func (s *PaymentService) StartPaymentListener(ctx context.Context) {
//	updateConfig := tgbotapi.NewUpdate(0)
//	updateConfig.Timeout = 60
//
//	updates := s.bot.GetUpdatesChan(updateConfig)
//
//	for {
//		select {
//		case update := <-updates:
//			switch {
//			case update.PreCheckoutQuery != nil:
//				if err := s.HandlePreCheckoutQuery(update.PreCheckoutQuery); err != nil {
//					log.Printf("Failed to handle pre-checkout query: %v", err)
//				}
//
//			case update.Message.SuccessfulPayment != nil:
//				if err := s.HandleSuccessfulPayment(ctx, update.Message); err != nil {
//					log.Printf("Failed to handle successful payment: %v", err)
//				}
//			}
//
//		case <-ctx.Done():
//			return
//		}
//	}
//}
//
//type RewardType string
//
//const (
//	RewardTypeEnergy RewardType = "ENERGY"
//)
//
//func (s *PaymentService) HandleRewards(ctx context.Context, reward RewardType) error {
//
//	switch rewardType {
//	case RewardTypeEnergy:
//		err := s.userService.AddEnergy(ctx, msg.From.ID, rewardAmount)
//		if err != nil {
//			return fmt.Errorf("failed to add energy: %w", err)
//		}
//
//	case RewardTypePoints:
//		err := s.userService.UpdateUserPoints(ctx, msg.From.ID, rewardAmount)
//		if err != nil {
//			return fmt.Errorf("failed to add points: %w", err)
//		}
//	}
//
//	return nil
//}
