package assignment

import (
	"context"
	"courier-bot/internal/models"
	"courier-bot/internal/repository"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

type Service struct {
	repo               repository.Repository
	log                *slog.Logger
	notificationSender NotificationSender
	assignmentTimeout  time.Duration
}

type Notification struct {
	CourierChatID int64
	OrderID       int
	Message       string
	WithButtons   bool
}

func NewService(repo repository.Repository, notificationSender NotificationSender) *Service {
	service := &Service{
		repo:               repo,
		notificationSender: notificationSender,
		assignmentTimeout:  10 * time.Minute,
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

	courier, err := s.repo.Courier.GetByChatID(ctx, int(chatID))
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
		if err := s.repo.Order.UpdateCourierID(ctx, orderID, courier); err != nil {
			return fmt.Errorf("failed to update order: %v", err)
		}
		s.log.Info("Order ACCEPTED by courier", "orderID", orderID, "courierID", courier.ID)
	} else {
		s.log.Info("Order REJECTED by courier", "orderID", orderID, "courierID", courier.ID)

		go s.findAndAssignCourier(ctx, orderID)
	}

	var responseMessage string
	if accepted {
		responseMessage = "✅ Заказ принят! Ожидайте детали доставки."
	} else {
		responseMessage = "❌ Вы отказались от заказа."
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

	message := s.formatAssignmentMessage(order)
	if err := s.sendNotification(courier.ChatID, orderID, message); err != nil {
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
			s.log.Info("Assigning order to courier", "orderID", orderID, "couierID", courier.ID)

			return s.assignOrderToCourier(ctx, orderID, courier.ID)
		}
	}

	s.log.Warn("All acitve couriers rejected order", "orderID", orderID, "courierQuantity", len(couriers))
	return &AssignmentResult{
		Success:      false,
		ErrorMessage: "All available couriers rejected this order",
	}, nil
}

func (s *Service) formatAssignmentMessage(order *models.Order) string {
	res := fmt.Sprintf(
		"*Новый заказ!*\n\n"+
			"*Адрес доставки:*\n%s, %s\n"+
			"*Квартира:*\n%s\n"+
			"*Подъезд:\n%s\n"+
			"*Клиент:*\n%s %s\n"+
			"*Телефон:*\n%s\n"+
			"*Сумма заказа:*\n%d\n\n"+
			"*Стоимость доставки:*\n%d\n\n"+
			"Примите или отколните заказ:",
		order.Address, order.City,
		order.Flat,
		order.Entrance,
		order.Surname, order.Name,
		order.PhoneNumber,
		order.FinalPrice,
		order.DeliveryPrice,
	)
	return res
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

func (s *Service) sendNotification(chatID int64, orderID int, message string) error {
	buttons := []Button{
		{
			Text: "✅ Принять",
			Data: fmt.Sprintf("accepted_%d", orderID),
		},
		{
			Text: "❌ Отклонить",
			Data: fmt.Sprintf("rejected_%d", orderID),
		},
	}

	err := s.notificationSender.SendMessageWithKeyboard(chatID, message, orderID, buttons)
	if err != nil {
		s.log.Error("Failed to send notification to courier for order", "chatID", chatID, "orderID", orderID)
		return err
	}

	s.log.Info("Notification sent to courier for order", "chatID", chatID, "orderID", orderID)
	return nil
}

func (s *Service) sendSimpleNotification(chatID int64, message string) error {
	err := s.notificationSender.SendMessage(chatID, message, 0)
	if err != nil {
		s.log.Error("Failed to send simple notification", "error", err)
		return err
	}

	return nil
}

func (s *Service) validateOrderForAssignment(order *models.Order) error {
	if !order.IsPaid {
		return errors.New("order is not paid")
	}

	if !order.IsAssembled {
		return errors.New("order is not assembled")
	}

	if order.CourierID != nil {
		return fmt.Errorf("order already assigned to courier %d", *order.CourierID)
	}

	if order.DeliverDate == nil {
		return fmt.Errorf("order has no delivery date")
	}

	return nil
}

func (s *Service) UpdateAssignmentTimeout(timeout time.Duration) {
	s.assignmentTimeout = timeout
}
