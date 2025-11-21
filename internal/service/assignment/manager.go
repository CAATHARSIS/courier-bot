package assignment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/CAATHARSIS/courier-bot/internal/models"
	"github.com/CAATHARSIS/courier-bot/internal/repository"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Manager struct {
	repo              *repository.Repository
	botAPI            *tgbotapi.BotAPI
	log               *slog.Logger
	assignmentTimeout time.Duration
	delay             time.Duration
	maxRetries        int

	activeTimers    map[int]*time.Timer
	assignmentLocks map[int]bool
	waitingOrders   map[int]*WaitingOrder
	mu              sync.RWMutex
	wg              sync.WaitGroup
	StopChan        chan struct{}
}

type AssignmentManagerConfig struct {
	AssignmentTimeout time.Duration
	RetryDelay        time.Duration
	MaxRetries        int
}

type AssignmentResult struct {
	Success      bool
	CourierID    int
	ErrorMessage string
}

type WaitingOrder struct {
	OrderID    int
	AssignedAt time.Time
	ExpiredAt  time.Time
	CourierID  int
	Status     models.CourierResponseStatus
	RetryCount int
	LastError  string
}

func NewManager(repo *repository.Repository, botAPI *tgbotapi.BotAPI, log *slog.Logger, config AssignmentManagerConfig) *Manager {
	return &Manager{
		repo:              repo,
		botAPI:            botAPI,
		log:               log,
		assignmentTimeout: config.AssignmentTimeout,
		delay:             config.RetryDelay,
		maxRetries:        config.MaxRetries,
		activeTimers:      make(map[int]*time.Timer),
		assignmentLocks:   make(map[int]bool),
		waitingOrders:     make(map[int]*WaitingOrder),
		StopChan:          make(chan struct{}),
	}
}

func (m *Manager) ProcessNewOrder(ctx context.Context, orderID int) error {
	m.log.Info("AssignmentManager: processing new order", "orderID", orderID)

	defer func(start time.Time) {
		m.log.Info("Order processing completed", "duration", time.Since(start))
	}(time.Now())

	if m.isOrderLocked(orderID) {
		m.log.Debug("This order is processing (skip)", "orderID", orderID)
		return nil
	}

	m.lockOrder(orderID)
	defer m.unlockOrder(orderID)

	m.registerWaitingOrder(orderID)

	order, err := m.GetOrderByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order %d: %v", orderID, err)
	}

	if err := m.validateOrderForAssignment(order); err != nil {
		return fmt.Errorf("order validation failed: %v", err)
	}

	result, err := m.findAndAssignCourier(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to assign courier: %v", err)
	}

	if !result.Success {
		m.log.Warn("No courier found for order", "orderID", orderID, "errorMessage", result.ErrorMessage)
	}

	return nil
}

func (m *Manager) findAndAssignCourier(ctx context.Context, orderID int) (*AssignmentResult, error) {
	m.log.Debug("Searching for available courier for order", "orderID", orderID)

	couriers, err := m.repo.Courier.GetActiveCouriers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active couriers: %v", err)
	}

	if len(couriers) == 0 {
		return &AssignmentResult{
			Success:      false,
			ErrorMessage: "No active couriers available",
		}, nil
	}

	rejectedCouriers, err := m.repo.OrderAssignment.GetRejectedCouriers(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rejected couriers: %v", err)
	}

	rejectedMap := make(map[int]bool)
	for _, id := range rejectedCouriers {
		rejectedMap[id] = true
	}

	for _, courier := range couriers {
		if !rejectedMap[courier.ID] {
			return m.assignOrderToCourier(ctx, orderID, courier.ID)
		}
	}

	m.log.Warn("All acitve couriers rejected order", "orderID", orderID, "courierQuantity", len(couriers))
	return &AssignmentResult{
		Success:      false,
		ErrorMessage: "All available couriers rejected this order",
	}, nil
}

func (m *Manager) assignOrderToCourier(ctx context.Context, orderID, courierID int) (*AssignmentResult, error) {
	m.log.Info("Assinging order to courier", "orderID", orderID, "courierID", courierID)

	order, err := m.repo.Order.GetByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %v", err)
	}

	if order.CourierID != nil {
		return &AssignmentResult{
			Success:      false,
			ErrorMessage: "Order already assigned to another courier",
		}, nil
	}

	courier, err := m.repo.Courier.GetByID(ctx, courierID)
	if err != nil {
		return nil, fmt.Errorf("failed to get courier: %v", err)
	}

	assignment := &models.OrderAssignment{
		OrderID:               orderID,
		CourierID:             courierID,
		AssignedAt:            time.Now(),
		ExpiredAt:             time.Now().Add(m.assignmentTimeout),
		CourierResponseStatus: models.ResponseStatusWaiting,
	}

	err = m.repo.OrderAssignment.Create(ctx, assignment)
	if err != nil {
		return nil, fmt.Errorf("failed to create assignment: %v", err)
	}

	message := m.formatDeliveryMessage(order)
	message.WriteString(fmt.Sprintf("⏰ *У вас %.0f минут, чтобы принять решение*\n\n", m.assignmentTimeout.Minutes()))
	message.WriteString("Примите или отколните заказ:")

	if err := m.sendNotificationWithKeyboard(courier.ChatID, orderID, message.String()); err != nil {
		m.log.Error("Failed to send notification to courier", "courierID", courier.ID, "error", err)
	}

	m.StartAssignmentTimer(ctx, orderID, assignment.ExpiredAt)

	m.log.Info("Order assigned to courier", "orderID", orderID, "courierID", courierID)

	return &AssignmentResult{
		Success:   true,
		CourierID: courierID,
	}, nil
}

func (m *Manager) HandleAssignmentTimeout(ctx context.Context, orderID int) {
	m.log.Info("AssignmentManager: timeout for order", "orderID", orderID)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.handleTimeoutLocked(ctx, orderID)
}

func (m *Manager) HandleCourierResponse(ctx context.Context, chatID int64, orderID int, accepted bool) error {
	m.log.Info("AssignmentManager: handling courier response for order", "orderID", orderID, "accepted", accepted)

	m.mu.RLock()
	waiting, exists := m.waitingOrders[orderID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("assignment not found for order %d", orderID)
	}

	if time.Now().After(waiting.ExpiredAt) {
		m.sendSimpleNotification(chatID, "⏰ Время для принятия заказа истекло")
		m.updateWaitingOrderStatus(orderID, false)
		statusExpired := models.ResponseStatusExpired
		m.UpdateOrderAssignmentStatust(ctx, orderID, statusExpired)
		return fmt.Errorf("assignment timeout for order %d", orderID)
	}

	m.cancelTimer(orderID)

	m.updateWaitingOrderStatus(orderID, accepted)

	courier, err := m.GetCourierByChatID(ctx, chatID)
	if err != nil {
		return fmt.Errorf("failed to get courier: %v", err)
	}

	assignment, err := m.GetAssignmentByOrderID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order assignment: %v", err)
	}

	if assignment.CourierID != courier.ID {
		return errors.New("courier mismatch: assignment belongs to another courier")
	}

	status := models.ResponseStatusRejected
	if accepted {
		status = models.ResponseStatusAccepted
		if err := m.repo.Order.UpdateCourierID(ctx, orderID, courier.ID); err != nil {
			return fmt.Errorf("failed to update order: %v", err)
		}

		go m.sendDeliveryDetails(ctx, chatID, orderID)
	} else {
		go m.scheduleRetry(ctx, orderID)
	}

	if err := m.repo.OrderAssignment.UpdateStatus(ctx, orderID, status); err != nil {
		return err
	}

	var responseMessage string
	if accepted {
		responseMessage = fmt.Sprintf("✅ Заказ #%d принят! Ожидайте детали доставки.", orderID)
	} else {
		responseMessage = fmt.Sprintf("❌ Вы отказались от заказа #%d.", orderID)
	}

	return m.sendSimpleNotification(chatID, responseMessage)
}

func (m *Manager) scheduleRetry(ctx context.Context, orderID int) {
	m.mu.Lock()

	waiting, exists := m.waitingOrders[orderID]
	if !exists {
		m.mu.Unlock()
		return
	}

	retryCount := waiting.RetryCount
	if retryCount > m.maxRetries {
		m.log.Warn("Max retry count reached, canceling assignment", "orderID", orderID)
		m.cancelAssignmentLocked(orderID)
		m.mu.Unlock()
		return
	}

	waiting.RetryCount++
	waiting.Status = models.ResponseStatusWaiting
	waiting.LastError = ""

	m.mu.Unlock()

	m.log.Info("Scheduling retry with delay", "orderID", orderID, "retryCount", retryCount+1, "delay", m.delay)

	time.AfterFunc(m.delay, func() {
		m.log.Info("Executing delayd retry", "orderID", orderID, "retryCount", retryCount+1)

		m.mu.RLock()
		_, stillExists := m.waitingOrders[orderID]
		m.mu.RUnlock()

		if !stillExists {
			m.log.Info("Assignment was canceled during retry delay", "orderID", orderID)
			return
		}

		if err := m.ProcessNewOrder(ctx, orderID); err != nil {
			m.log.Error("Retry failed for order", "orderID", orderID, "error", err)
		}
	})
}

func (m *Manager) StartAssignmentTimer(ctx context.Context, orderID int, expiry time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if oldTimer, exists := m.activeTimers[orderID]; exists {
		oldTimer.Stop()
		delete(m.activeTimers, orderID)
	}

	duration := time.Until(expiry)
	if duration <= 0 {
		go m.handleTimeoutLocked(ctx, orderID)
		return
	}

	m.wg.Add(1)
	timer := time.AfterFunc(duration, func() {
		defer m.wg.Done()

		select {
		case <-m.StopChan:
			return
		default:
			m.mu.Lock()
			m.handleTimeoutLocked(ctx, orderID)
			m.mu.Unlock()
		}
	})

	m.activeTimers[orderID] = timer
	if waiting, exitsts := m.waitingOrders[orderID]; exitsts {
		waiting.ExpiredAt = expiry
	}
}

func (m *Manager) handleTimeoutLocked(ctx context.Context, orderID int) {
	waiting, exists := m.waitingOrders[orderID]
	if !exists || waiting.Status != models.ResponseStatusWaiting {
		return
	}

	waiting.Status = models.ResponseStatusExpired
	waiting.LastError = "Assignment timeout"

	go func() {
		if err := m.repo.OrderAssignment.UpdateStatus(ctx, orderID, models.ResponseStatusExpired); err != nil {
			m.log.Error("Failed to update assignment status to expired", "orderID", orderID, "error", err)
		}
	}()

	delete(m.activeTimers, orderID)

	go m.scheduleRetry(ctx, orderID)
}

func (m *Manager) registerWaitingOrder(orderID int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.waitingOrders[orderID] = &WaitingOrder{
		OrderID:    orderID,
		AssignedAt: time.Now(),
		Status:     models.ResponseStatusWaiting,
	}
}

func (m *Manager) GetActiveAssignments(orderID int) map[int]*WaitingOrder {
	m.mu.RLock()
	defer m.mu.RUnlock()

	active := make(map[int]*WaitingOrder)
	for orderID, waiting := range m.waitingOrders {
		if waiting.Status == models.ResponseStatusWaiting {
			active[orderID] = waiting
		}
	}

	return active
}

func (m *Manager) GetAssignmentsStats() *AssignmentStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &AssignmentStats{
		ActiveTimers: len(m.activeTimers),
		LockedOrders: len(m.assignmentLocks),
		ByStatus:     make(map[models.CourierResponseStatus]int),
		RetryStats:   make(map[int]int),
	}

	for _, waiting := range m.waitingOrders {
		stats.TotalWaiting++
		stats.ByStatus[waiting.Status]++

		stats.RetryStats[waiting.RetryCount]++

		if waiting.RetryCount > stats.MaxRetries {
			stats.MaxRetries = waiting.RetryCount
		}
	}

	return stats
}

type AssignmentStats struct {
	TotalWaiting int
	ActiveTimers int
	LockedOrders int
	MaxRetries   int
	ByStatus     map[models.CourierResponseStatus]int
	RetryStats   map[int]int
}

func (m *Manager) CancelAssignment(orderID int) error {
	m.log.Info("AssignmentManager: canceling assignment for order", "orderID", orderID)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.cancelAssignmentLocked(orderID)
	return nil
}

func (m *Manager) cancelAssignmentLocked(orderID int) {
	if timer, exists := m.activeTimers[orderID]; exists {
		timer.Stop()
		delete(m.activeTimers, orderID)
	}

	delete(m.waitingOrders, orderID)
	delete(m.assignmentLocks, orderID)
}

func (m *Manager) isOrderLocked(orderID int) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.assignmentLocks[orderID]
}

func (m *Manager) lockOrder(orderID int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.assignmentLocks[orderID] = true
}

func (m *Manager) unlockOrder(orderID int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.assignmentLocks, orderID)
}

func (m *Manager) cancelTimer(orderID int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if timer, exists := m.activeTimers[orderID]; exists {
		timer.Stop()
		delete(m.activeTimers, orderID)
	}
}

func (m *Manager) validateOrderForAssignment(order *models.Order) error {
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

func (m *Manager) updateWaitingOrderStatus(orderID int, accepted bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if waiting, exists := m.waitingOrders[orderID]; exists {
		if accepted {
			waiting.Status = models.ResponseStatusAccepted
		} else {
			waiting.Status = models.ResponseStatusRejected
		}

		waiting.LastError = ""
	}
}

func (m *Manager) CleanUpAssignments() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	lifeCycle := 24 * time.Hour

	for orderID, waiting := range m.waitingOrders {
		if now.Sub(waiting.AssignedAt) > lifeCycle {
			m.log.Info("Delete dead assignments", "orderID", orderID)

			m.cancelAssignmentLocked(orderID)
		}
	}
}

func (m *Manager) GetWaitingOrderInfo(orderID int) *WaitingOrder {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.waitingOrders[orderID]
}

func (m *Manager) StartCleanupWorker() {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()

		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				m.CleanUpAssignments()
			case <-m.StopChan:
				return
			}
		}
	}()
}

func (m *Manager) Stop() {
	close(m.StopChan)

	m.mu.Lock()

	for orderID, timer := range m.activeTimers {
		timer.Stop()
		delete(m.activeTimers, orderID)
	}
	m.mu.Unlock()

	m.wg.Wait()

	m.log.Info("AssignmentManager stopped gracefully")
}

func (m *Manager) formatDeliveryMessage(order *models.Order) *strings.Builder {
	var builder strings.Builder

	builder.WriteString("*Новый заказ!*\n\n")
	builder.WriteString(fmt.Sprintf("*Адрес доставки:* %s, %s\n", order.Address, order.City))
	builder.WriteString(fmt.Sprintf("*Дата доставки:* %s", m.formatDeliveryTime(order.DeliveryDate)))

	hasFlat := order.Flat.Valid && order.Flat.String != ""
	hasEntrance := order.Entrance.Valid && order.Entrance.String != ""

	if hasFlat {
		builder.WriteString(fmt.Sprintf("*Квартира:* %s\n", order.Flat.String))
	}

	if hasEntrance {
		builder.WriteString(fmt.Sprintf("*Подъезд:* %s\n", order.Entrance.String))
	}

	builder.WriteString(fmt.Sprintf("*Клиент:* %s\n", order.Name))
	builder.WriteString(fmt.Sprintf("*Телефон:* %s\n", order.PhoneNumber))
	builder.WriteString(fmt.Sprintf("*Сумма заказа:* %d\n\n", order.FinalPrice))
	builder.WriteString(fmt.Sprintf("*Стоимость доставки:* %d\n\n", order.DeliveryPrice))

	if hasFlat && !hasEntrance {
		builder.WriteString("*Подъезд не указан, для уточнения информации свяжитесь с клиентом\n\n*")
	} else if hasEntrance && !hasFlat {
		builder.WriteString("*Квартира не указана, для уточнения информации свяжитесь с клиентом\n\n*")
	} else {
		builder.WriteString("*Квартира и подъезд не указаны, для уточнения информации свяжитесь с клиентом\n\n*")
	}

	return &builder
}

func (m *Manager) sendNotificationWithKeyboard(chatID int64, orderID int, message string) error {
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "Markdown"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Принять", fmt.Sprintf("accept_%d", orderID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отклонить", fmt.Sprintf("reject_%d", orderID)),
		),
	)
	msg.ReplyMarkup = keyboard

	_, err := m.botAPI.Send(msg)
	if err != nil {
		m.log.Error("Failed to send message with keyboard", "chatID", chatID, "error", err)
		return err
	}

	m.log.Info("Message with keyboard sent", "chatID", chatID, "orderID", orderID)
	return nil
}

func (m *Manager) sendNotificationWithDeliveryKeyboard(chatID int64, message string, orderID int, order *models.Order) error {
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "Markdown"

	rows := [][]tgbotapi.InlineKeyboardButton{}

	if order.Address != "" {
		navigationRow := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗺️ Построить маршрут", fmt.Sprintf("nav_%d_%s", orderID, m.EscapeCallbackData(order.Address + "_" + order.City))),
		)
		rows = append(rows, navigationRow)
	}

	if order.PhoneNumber != "" {
		callRow := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📞 Позвонить клиенту", fmt.Sprintf("call_%d_%s", orderID, order.PhoneNumber)),
		)
		rows = append(rows, callRow)
	}

	completionRow := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🏁 Доставка завершена", fmt.Sprintf("%s_%d", "complete", orderID)),
	)
	rows = append(rows, completionRow)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	msg.ReplyMarkup = keyboard

	_, err := m.botAPI.Send(msg)
	if err != nil {
		m.log.Error("Failed to send delivery details", "chatID", chatID, "error", err)
		return err
	}

	m.log.Info("Delivery Details sent", "chatID", chatID, "orderID", orderID)
	return nil
}

func (m *Manager) sendSimpleNotification(chatID int64, message string) error {
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "Markdown"

	_, err := m.botAPI.Send(msg)
	if err != nil {
		m.log.Error("Failed to send message to courier", "chatID", chatID, "error", err)
		return err
	}

	return nil
}

func (m *Manager) sendDeliveryDetails(ctx context.Context, chatID int64, orderID int) error {
	order, err := m.repo.Order.GetByID(ctx, orderID)
	if err != nil {
		m.log.Error("Failed to get order for delivery details", "orderID", orderID, "error", err)
		return err
	}

	message := m.formatDeliveryMessage(order)
	message.WriteString("*Используйте кнопки ниже для управления доставкой*")

	return m.sendNotificationWithDeliveryKeyboard(chatID, message.String(), orderID, order)
}

func (m *Manager) formatDeliveryTime(deliveryTime *time.Time) string {
	if deliveryTime == nil {
		return "не указано"
	}
	return deliveryTime.Format("02.01.2006 в 15:04")
}

func (m *Manager) UpdateAssignmentTimeout(timeout time.Duration) {
	m.assignmentTimeout = timeout
}

// escapeCallbackData экранирует данные для callback_data
// В Telegram callback_data не может превышать 64 байта и содержать некоторые символы
func (m *Manager) EscapeCallbackData(data string) string {
	if len(data) > 50 {
		data = data[:50]
	}

	replacements := map[string]string{
		"\n": "",
		"\t": "",
	}

	for old, new := range replacements {
		data = strings.ReplaceAll(data, old, new)
	}

	return data
}

// REPO

func (m *Manager) CheckCourierByChatID(ctx context.Context, chatID int64) bool {
	return m.repo.Courier.CheckCourierByChatID(ctx, chatID)
}

func (m *Manager) CreateCourier(ctx context.Context, courier *models.Courier) error {
	return m.repo.Courier.Create(ctx, courier)
}

func (m *Manager) GetActiveOrdersByCourier(ctx context.Context, chatID int64) ([]models.Order, error) {
	courier, err := m.repo.Courier.GetByChatID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get courier: %v", err)
	}

	return m.repo.Order.GetActiveOrdersByCourier(ctx, courier.ID)
}

func (m *Manager) GetOrderByID(ctx context.Context, id int) (*models.Order, error) {
	return m.repo.Order.GetByID(ctx, id)
}

func (m *Manager) GetCourierByChatID(ctx context.Context, chatID int64) (*models.Courier, error) {
	courier, err := m.repo.Courier.GetByChatID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("get courier by chat ID %d: %w", chatID, err)
	}
	return courier, err
}

func (m *Manager) UpdateCourierStatusIsActive(ctx context.Context, chatID int64, currStatus bool) error {
	err := m.repo.Courier.UpdateCourierStatusIsActive(ctx, chatID, currStatus)
	if err != nil {
		m.log.Error("Internal Server Error", "Error", err)
	}
	return err
}

func (m *Manager) GetAssignmentByOrderID(ctx context.Context, orderID int) (*models.OrderAssignment, error) {
	assignment, err := m.repo.OrderAssignment.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	return assignment, nil
}

func (m *Manager) GetWaitingAssignmentsByCourierID(ctx context.Context, coureirID int) ([]models.OrderAssignment, error) {
	return m.repo.OrderAssignment.GetWaitingByCourierID(ctx, coureirID)
}

func (m *Manager) GetWaitingAssignmentsByOrderID(ctx context.Context, orderID int) (*models.OrderAssignment, error) {
	return m.repo.OrderAssignment.GetWiatingByOrderID(ctx, orderID)
}

func (m *Manager) UpdateOrderAssignmentStatust(ctx context.Context, id int, status models.CourierResponseStatus) error {
	return m.repo.OrderAssignment.UpdateStatus(ctx, id, status)
}

func (m *Manager) UpdateCourierLocation(ctx context.Context, chatID int64, location models.CourierLocation) error {
	return m.repo.Courier.UpdateLocation(ctx, chatID, location)
}

func (m *Manager) UpdateCourierTrackingMode(ctx context.Context, chatID int64, trackingMode bool) error {
	return m.repo.Courier.UpdateTrackingMode(ctx, chatID, trackingMode)
}

func (m *Manager) GetCourierIsActiveStatus(ctx context.Context, chatID int64) (bool, error) {
	return m.repo.Courier.GetIsActiveStatus(ctx, chatID)
}
