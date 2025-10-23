package notification

import (
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type NotificationType string

const (
	TypeNewAssignment NotificationType = "new_assignment"
	TypeAccepted      NotificationType = "accepted"
	TypeRejected      NotificationType = "rejected"
	TypeReminder      NotificationType = "reminder"
	TypeSystem        NotificationType = "system"
	TypeDeliveryInfo  NotificationType = "delivery_info"
)

type Notification struct {
	Type          NotificationType
	CourierChatID int64
	OrderID       int
	Message       string
	Data          map[string]interface{}
}

type NotificationService struct {
	botAPI *tgbotapi.BotAPI
	log    *slog.Logger
}

func NewService(botAPI *tgbotapi.BotAPI, log *slog.Logger) *NotificationService {
	return &NotificationService{
		botAPI: botAPI,
		log:    log,
	}
}

func (s *NotificationService) SendNotification(notification *Notification) error {
	switch notification.Type {
	case TypeNewAssignment:
		return s.SendAssignmentNotification(notification.CourierChatID, notification.OrderID, notification.Message)
	case TypeAccepted:
		return s.SendAcceptanceConfirmation(notification.CourierChatID, notification.OrderID)
	case TypeRejected:
		return s.SendRejectionConfirmation(notification.CourierChatID, notification.OrderID)
	case TypeReminder:
		return s.SendReminder(notification.CourierChatID, notification.OrderID, notification.Data)
	case TypeDeliveryInfo:
		return s.SendDeliveryInformation(notification.CourierChatID, notification.OrderID, notification.Data)
	default:
		return fmt.Errorf("unknown notification type: %s", notification.Type)
	}
}

func (s *NotificationService) SendAssignmentNotification(chatID int64, orderID int, orderInfo string) error {
	message := fmt.Sprintf(
		"📦 *Новый заказ для доставки*\n\n%s\n\n⏰ *У вас 10 минут, чтобы принять решение*",
		orderInfo,
	)

	keyboard := s.createAssignmentKeyboard(orderID)

	return s.sendMessageWithKeyboard(chatID, message, keyboard)
}

func (s *NotificationService) SendAcceptanceConfirmation(chatID int64, orderID int) error {
	message := fmt.Sprintf(
		"✅ *Заказ %d принят!*\n\n"+
			"Заказ успешно закреплен за вами.\n\n"+
			"📋 Ожидайте подробную информацию о доставке в ближайшее время.",
		orderID,
	)

	return s.sendSimpleMessage(chatID, message)
}

func (s *NotificationService) SendRejectionConfirmation(chatID int64, orderID int) error {
	message := fmt.Sprintf(
		"❌ *Вы отказались от заказа #%d*\n\n"+
			"Заказ будет предложен другому курьеру\n\n"+
			"🔄 Ожидайте новые заказы!",
		orderID,
	)

	return s.sendSimpleMessage(chatID, message)
}

func (s *NotificationService) SendReminder(chatID int64, orderID int, data map[string]interface{}) error {
	minutesLeft := 5
	if min, ok := data["minutes_left"].(int); ok {
		minutesLeft = min
	}

	message := fmt.Sprintf(
		"⏰ *Напоминание*\n\n"+
			"По заказу #%d осталось *%d минут* для принятия решения.\n\n"+
			"Пожалуйста, примите или отклоните заказ в ближайшее время.",
		orderID,
		minutesLeft,
	)

	keyboard := s.createAssignmentKeyboard(orderID)
	return s.sendMessageWithKeyboard(chatID, message, keyboard)
}

func (s *NotificationService) SendSystemNotification(chatID int64, message string) error {
	fullMessage := fmt.Sprintf("🔔 *Системное уведомление*\n\n%s", message)
	return s.sendSimpleMessage(chatID, fullMessage)
}

func (s *NotificationService) SendDeliveryInformation(chatID int64, orderID int, data map[string]interface{}) error {
	address, _ := data["address"].(string)
	customerName, _ := data["customer_name"].(string)
	customerPhone, _ := data["customer_phone"].(string)
	deliveryTime, _ := data["delivery_time"].(string)
	orderPrice, _ := data["order_price"].(int)
	deliveryPrice, _ := data["delivery_price"].(int)

	message := fmt.Sprintf(
		"🚚 *Детали доставки #%d*\n\n"+
			"*Адрес:* %s\n"+
			"*Клиент:* %s\n"+
			"*Телефон:* %s\n"+
			"*Время доставки:* %s\n"+
			"*Стоимость заказа:* %d\n"+
			"*Стоимость доставки:* %d\n"+
			"📍 Используйте навигацию для построения маршрута\n"+
			"📞 Свяжитесь с клиентом за 15 минут до доставки",
		orderID,
		address,
		customerName,
		customerPhone,
		deliveryTime,
		orderPrice,
		deliveryPrice,
	)

	keyboard := s.createDeliveryKeyboard(address, customerPhone)
	return s.sendMessageWithKeyboard(chatID, message, keyboard)

}

func (s *NotificationService) sendMessageWithKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	_, err := s.botAPI.Send(msg)
	if err != nil {
		s.log.Error("Failed to send message with keyboard to chat", "chatID", chatID, "error", err)
		return err
	}

	s.log.Info("Message with keyboard sent to chat", "chatID", chatID)
	return nil
}

func (s *NotificationService) sendSimpleMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	_, err := s.botAPI.Send(msg)
	if err != nil {
		s.log.Error("Failed to send message to chat", "chatID", chatID, "error", err)
		return err
	}
	s.log.Info("Message sent to chat", "chatID", chatID)
	return nil
}

func (s *NotificationService) createAssignmentKeyboard(orderID int) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Принять", fmt.Sprintf("accept_%d", orderID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отклонить", fmt.Sprintf("reject_%d", orderID)),
		),
	)
}

func (s *NotificationService) createDeliveryKeyboard(address, phone string) tgbotapi.InlineKeyboardMarkup {
	rows := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("🗺️ Открыть в Яндекс.Картах", fmt.Sprintf("nav_%s", address)),
		},
	}

	if phone != "" {
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("📞 Позвонить клиенту", fmt.Sprintf("call_%s", phone)),
		})
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func (s *NotificationService) SendMessage(chatID int64, message string) error {
	return s.sendSimpleMessage(chatID, message)
}

func (s *NotificationService) SendMessageWithKeyboard(chatID int64, message string, orderID int) error {
	keyboard := s.createAssignmentKeyboard(orderID)
	return s.sendMessageWithKeyboard(chatID, message, keyboard)
}

func (s *NotificationService) TestConnection() error {
	_, err := s.botAPI.GetMe()
	if err != nil {
		return fmt.Errorf("failed to connect to Telegram API: %v", err)
	}

	s.log.Info("Telegram bot connection test successfull")
	return nil
}

func (s *NotificationService) GetDeliveryInfoMessage(orderID int, address, customerName, customerPhone, deliveryTime string, orderPrice, deliveryPrice int) string {
	return fmt.Sprintf(
		"🚚 *Детали доставки #%d*\n\n"+
			"*Адрес:* %s\n"+
			"*Клиент:* %s\n"+
			"*Телефон:* %s\n"+
			"*Время доставки:* %s\n"+
			"*Стоимость заказа:* %d\n"+
			"*Стоимость доставки:* %d\n"+
			"📍 Используйте навигацию для построения маршрута\n"+
			"📞 Свяжитесь с клиентом за 15 минут до доставки",
		orderID,
		address,
		customerName,
		customerPhone,
		deliveryTime,
		orderPrice,
		deliveryPrice,
	)
}

func (s *NotificationService) GetAssignemntMessage(orderInfo string) string {
	return fmt.Sprintf(
		"📦 *Новый заказ для доставки!*\n\n%s\n\n⏰ *У вас 10 минут чтобы принять решение*",
		orderInfo,
	)
}
