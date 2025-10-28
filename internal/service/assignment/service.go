package assignment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/CAATHARSIS/courier-bot/internal/models"
	"github.com/CAATHARSIS/courier-bot/internal/repository"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Service struct {
	repo              repository.Repository
	log               *slog.Logger
	botAPI            *tgbotapi.BotAPI
	assignmentTimeout time.Duration
}

func NewService(repo repository.Repository, botAPI *tgbotapi.BotAPI, log *slog.Logger) *Service {
	service := &Service{
		repo:              repo,
		log:               log,
		botAPI:            botAPI,
		assignmentTimeout: 10 * time.Minute,
	}

	return service
}

func (s *Service) ProcessNewOrder(ctx context.Context, orderID int) error {
	s.log.Info("Proccessing new order", "orderID", orderID)

	order, err := s.repo.Order.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order %d: %v", orderID, err)
	}

	if err := s.validateOrderForAssignment(order); err != nil {
		return fmt.Errorf("order validation failed: %v", err)
	}

	result, err := s.findAndAssignCourier(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to assign courier: %v", err)
	}

	if !result.Success {
		s.log.Warn("No courier found for order", "orderID", orderID, "errorMessage", result.ErrorMessage)
	}

	return nil
}

func (s *Service) HandleCourierResponse(ctx context.Context, chatID int64, orderID int, accepted bool) error {
	s.log.Info("Processing courier responsse", "orderID", orderID, "accepted", accepted)

	courier, err := s.repo.Courier.GetByChatID(ctx, chatID)
	if err != nil {
		return fmt.Errorf("failed to get courier: %v", err)
	}

	assignment, err := s.repo.OrderAssignment.GetByOrderID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order assignment: %v", err)
	}

	if assignment.CourierID != courier.ID {
		return errors.New("courier mismatch: assignment belongs to another courier")
	}

	if time.Now().After(assignment.ExpiredAt) {
		s.sendSimpleNotification(chatID, "⏰ Время для принятия заказа истекло")
		return s.repo.OrderAssignment.UpdateStatus(ctx, orderID, models.ResponsseStatusExpired)
	}

	status := models.ResponseStatusRejected
	if accepted {
		status = models.ResponseStatusAccepted
		if err := s.repo.Order.UpdateCourierID(ctx, orderID, courier.ID); err != nil {
			return fmt.Errorf("failed to update order: %v", err)
		}
		s.log.Info("Order ACCEPTED by courier", "orderID", orderID, "courierID", courier.ID)

		s.repo.Courier.UpdateCurrentOrderID(ctx, chatID, orderID)

		go s.sendDeliveryDetails(ctx, chatID, orderID)
	} else {
		s.log.Info("Order REJECTED by courier", "orderID", orderID, "courierID", courier.ID)

		go s.findAndAssignCourier(ctx, orderID)
	}

	var responseMessage string
	if accepted {
		responseMessage = fmt.Sprintf("✅ Заказ #%d принят! Ожидайте детали доставки.", orderID)
	} else {
		responseMessage = fmt.Sprintf("❌ Вы отказались от заказа #%d.", orderID)
	}

	if err := s.sendSimpleNotification(chatID, responseMessage); err != nil {
		s.log.Error("Failed to send response message", "error", err)
	}

	return s.repo.OrderAssignment.UpdateStatus(ctx, orderID, status)
}

func (s *Service) assignOrderToCourier(ctx context.Context, orderID, courierID int) (*AssignmentResult, error) {
	s.log.Info("Assinging order to courier", "orderID", orderID, "courierID", courierID)

	order, err := s.repo.Order.GetByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %v", err)
	}

	if order.CourierID != nil {
		return &AssignmentResult{
			Success:      false,
			ErrorMessage: "Order already assigned to another courier",
		}, nil
	}

	courier, err := s.repo.Courier.GetByID(ctx, courierID)
	if err != nil {
		return nil, fmt.Errorf("failed to get courier: %v", err)
	}

	assignment := &models.OrderAssignment{
		OrderID:               orderID,
		CourierID:             courierID,
		AssignedAt:            time.Now(),
		ExpiredAt:             time.Now().Add(s.assignmentTimeout),
		CourierResponseStatus: models.ResponseStatusWaiting,
	}

	err = s.repo.OrderAssignment.Create(ctx, assignment)
	if err != nil {
		return nil, fmt.Errorf("failed to create assignment: %v", err)
	}

	message := s.formatDeliveryMessage(order)
	message.WriteString("⏰ *У вас 10 минут, чтобы принять решение*\n\n")
	message.WriteString("Примите или отколните заказ:")

	if err := s.sendNotificationWithKeyboard(courier.ChatID, orderID, message.String()); err != nil {
		s.log.Error("Failed to send notification to courier", "courierID", courier.ID, "error", err)
	}

	go s.startAssignmentTimer(ctx, orderID, assignment.ExpiredAt)

	s.log.Info("Order assigned to courier", "orderID", orderID, "courierID", courierID)

	return &AssignmentResult{
		Success:   true,
		CourierID: courierID,
	}, nil
}

func (s *Service) findAndAssignCourier(ctx context.Context, orderID int) (*AssignmentResult, error) {
	s.log.Debug("Searching for available courier for order", "orderID", orderID)

	couriers, err := s.repo.Courier.GetActiveCouriers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active couriers: %v", err)
	}

	if len(couriers) == 0 {
		return &AssignmentResult{
			Success:      false,
			ErrorMessage: "No active couriers available",
		}, nil
	}

	rejectedCouriers, err := s.repo.OrderAssignment.GetRejectedCouriers(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rejected couriers: %v", err)
	}

	rejectedMap := make(map[int]bool)
	for _, id := range rejectedCouriers {
		rejectedMap[id] = true
	}

	for _, courier := range couriers {
		if !rejectedMap[courier.ID] {
			return s.assignOrderToCourier(ctx, orderID, courier.ID)
		}
	}

	s.log.Warn("All acitve couriers rejected order", "orderID", orderID, "courierQuantity", len(couriers))
	return &AssignmentResult{
		Success:      false,
		ErrorMessage: "All available couriers rejected this order",
	}, nil
}

func (s *Service) formatDeliveryMessage(order *models.Order) *strings.Builder {
	var builder strings.Builder

	builder.WriteString("*Новый заказ!*\n\n")
	builder.WriteString(fmt.Sprintf("*Адрес доставки:*\n%s, %s\n", order.Address, order.City))

	hasFlat := order.Flat.Valid && order.Flat.String != ""
	hasEntrance := order.Entrance.Valid && order.Entrance.String != ""

	if hasFlat {
		builder.WriteString(fmt.Sprintf("*Квартира:*\n%s\n", order.Flat.String))
	}

	if hasEntrance {
		builder.WriteString(fmt.Sprintf("*Подъезд:*\n%s\n", order.Entrance.String))
	}

	builder.WriteString(fmt.Sprintf("*Клиент:*\n%s\n", order.Name))
	builder.WriteString(fmt.Sprintf("*Телефон:*\n%s\n", order.PhoneNumber))
	builder.WriteString(fmt.Sprintf("*Сумма заказа:*\n%d\n\n", order.FinalPrice))
	builder.WriteString(fmt.Sprintf("*Стоимость доставки:*\n%d\n\n", order.DeliveryPrice))

	if hasFlat && !hasEntrance {
		builder.WriteString("*Подъезд не указан, для уточнения информации свяжитесь с клиентом\n\n*")
	} else if hasEntrance && !hasFlat {
		builder.WriteString("*Квартира не указана, для уточнения информации свяжитесь с клиентом\n\n*")
	} else {
		builder.WriteString("*Квартира и подъезд не указаны, для уточнения информации свяжитесь с клиентом\n\n*")
	}

	return &builder
}

func (s *Service) startAssignmentTimer(ctx context.Context, orderID int, expiry time.Time) {
	duration := time.Until(expiry)
	if duration <= 0 {
		return
	}

	time.Sleep(duration)

	assignment, err := s.repo.OrderAssignment.GetByOrderID(ctx, orderID)
	if err != nil {
		s.log.Error("Failed to get assignment for timer", "error", err)
		return
	}

	if assignment.CourierResponseStatus == models.ResponseStatusWaiting {
		s.log.Info("Assignment timeout for order", "orderID", orderID)

		if err := s.repo.OrderAssignment.UpdateStatus(ctx, orderID, models.ResponsseStatusExpired); err != nil {
			s.log.Error("failed to update assignment status to expired", "error", err)
			return
		}

		go s.findAndAssignCourier(ctx, orderID)
	}
}

func (s *Service) sendNotificationWithKeyboard(chatID int64, orderID int, message string) error {
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "Markdown"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Принять", fmt.Sprintf("accept_%d", orderID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отклонить", fmt.Sprintf("reject_%d", orderID)),
		),
	)
	msg.ReplyMarkup = keyboard

	_, err := s.botAPI.Send(msg)
	if err != nil {
		s.log.Error("Failed to send message with keyboard", "chatID", chatID, "error", err)
		return err
	}

	s.log.Info("Message with keyboard sent", "chatID", chatID, "orderID", orderID)
	return nil
}

func (s *Service) sendNotificationWithDeliveryKeyboard(chatID int64, message string, orderID int, order *models.Order) error {
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "Markdown"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗺️ Маршрут", fmt.Sprintf("nav_%d_%s", orderID, s.escapeAddress(order.Address))),
			tgbotapi.NewInlineKeyboardButtonData("📞 Позвонить", fmt.Sprintf("call_%d_%s", orderID, order.PhoneNumber)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Доставлено", fmt.Sprintf("complete_%d", orderID)),
			tgbotapi.NewInlineKeyboardButtonData("🚨 Проблема", fmt.Sprintf("problem_%d", orderID)),
		),
	)
	msg.ReplyMarkup = keyboard

	_, err := s.botAPI.Send(msg)
	if err != nil {
		s.log.Error("Failed to send delivery details", "chatID", chatID, "error", err)
		return err
	}

	s.log.Info("Delivery Details sent", "chatID", chatID, "orderID", orderID)
	return nil
}

func (s *Service) sendSimpleNotification(chatID int64, message string) error {
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "Markdown"

	_, err := s.botAPI.Send(msg)
	if err != nil {
		s.log.Error("Failed to send message to courier", "chatID", chatID, "error", err)
		return err
	}

	return nil
}

func (s *Service) sendDeliveryDetails(ctx context.Context, chatID int64, orderID int) error {
	order, err := s.repo.Order.GetByID(ctx, orderID)
	if err != nil {
		s.log.Error("Failed to get order for delivery details", "orderID", orderID, "error", err)
		return err
	}

	message := s.formatDeliveryMessage(order)
	message.WriteString("*Используйте кнопки ниже для управления доставкой*")

	return s.sendNotificationWithDeliveryKeyboard(chatID, message.String(), orderID, order)
}

func (s *Service) validateOrderForAssignment(order *models.Order) error {
	if !order.IsPaid {
		return errors.New("order is not paid")
	}

	if order.IsAssembled.Valid && order.IsAssembled.Bool == false {
		return errors.New("order is not assembled")
	}

	if order.CourierID != nil {
		return fmt.Errorf("order already assigned to courier %d", *order.CourierID)
	}

	return nil
}

func (s *Service) formatDeliveryTime(deliveryTime *time.Time) string {
	if deliveryTime == nil {
		return "не указано"
	}
	return deliveryTime.Format("02.01.2006 в 15:04")
}

func (s *Service) escapeAddress(address string) string {
	if len(address) > 50 {
		address = address[:50]
	}
	return address
}

func (s *Service) UpdateAssignmentTimeout(timeout time.Duration) {
	s.assignmentTimeout = timeout
}

func (s *Service) GetActiveOrdersByCourier(ctx context.Context, chatID int64) ([]models.Order, error) {
	courier, err := s.repo.Courier.GetByChatID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get courier: %v", err)
	}

	activeOrders, err := s.repo.Order.GetActiveOrdersByCourier(ctx, courier.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active orders for courier with id #%d: %v", courier.ID, err)
	}

	return activeOrders, nil
}

func (s *Service) GetAssignmentByOrderID(ctx context.Context, orderID int) (*models.OrderAssignment, error) {
	assignment, err := s.repo.OrderAssignment.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	return assignment, nil
}

func (s *Service) UpdateOrderStatusReceived(ctx context.Context, id int, received bool) error {
	return s.repo.Order.UpdateStatusReceived(ctx, id, received)
}

func (s *Service) GetOrderByID(ctx context.Context, id int) (*models.Order, error) {
	return s.repo.Order.GetByID(ctx, id)
}

func (s *Service) GetCourierByChatID(ctx context.Context, chatID int64) (*models.Courier, error) {
	courier, err := s.repo.Courier.GetByChatID(ctx, chatID)
	if err != nil {
		s.log.Error("Failed to get courier by chat ID", "Error", err)
	}
	return courier, err
}

func (s *Service) CheckCourierByChatID(ctx context.Context, chatID int64) bool {
	return s.repo.Courier.CheckCourierByChatID(ctx, chatID)
}

func (s *Service) CreateCourier(ctx context.Context, courier *models.Courier) error {
	return s.repo.Courier.Create(ctx, courier)
}

func (s *Service) UpdateCourierStatusIsActive(ctx context.Context, chatID int64, currStatus bool) error {
	err := s.repo.Courier.UpdateCourierStatusIsActive(ctx, chatID, currStatus)
	if err != nil {
		s.log.Error("Internal Server Error", "Error", err)
	}
	return err
}
