package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/CAATHARSIS/courier-bot/internal/models"
	"github.com/CAATHARSIS/courier-bot/internal/service/assignment"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handlers struct {
	assignmentManager *assignment.Manager
	keyboardManager   KeyboardManagerInterface
	log               *slog.Logger
}

func NewHandlers(assignmentManager *assignment.Manager, keyboardManager KeyboardManagerInterface, log *slog.Logger) *Handlers {
	return &Handlers{
		assignmentManager: assignmentManager,
		keyboardManager:   keyboardManager,
		log:               log,
	}
}

func (h *Handlers) HandleMessage(ctx context.Context, bot BotInterface, update tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	text := update.Message.Text

	h.log.Info("Received message", "From", chatID, "Message", text)

	switch text {
	case "/start":
		h.HandleStartCommand(bot, chatID, update.Message.From)
	case "/help", "🆘 Помощь":
		h.HandleHelpCommand(bot, chatID)
	case "/orders", "📋 Мои заказы":
		h.HandleMyOrdersCommand(ctx, bot, chatID)
	case "/status", "ℹ️ Статус":
		h.HandleStatusCommand(bot, chatID)
	case "/settings", "⚙️ Настройки":
		h.HandleSettingsCommand(bot, chatID)
	default:
		h.HandleUnknownCommand(bot, chatID)
	}
}

func (h *Handlers) HandleCallback(ctx context.Context, bot BotInterface, update tgbotapi.Update) {
	if update.CallbackQuery == nil {
		return
	}

	callback := update.CallbackQuery

	var chatID int64
	if callback.Message != nil {
		chatID = callback.Message.Chat.ID
	} else {
		chatID = callback.From.ID
		h.log.Warn("Callback without message, usting user ID as chatID", "userID", callback.From.ID, "callbackData", callback.Data)
	}

	callbackData := callback.Data

	h.log.Info("Received callback", "chatID", chatID, "callbackData", callbackData, "messageID", callback.Message.MessageID)

	if bot == nil {
		h.log.Error("Bot interface is nil in callback handler")
		return
	}

	if err := bot.AnswerCallbackQuery(callback.ID); err != nil {
		h.log.Error("Failed to answer callback query", "error", err)
	}

	action := h.keyboardManager.GetActionFromCallback(callbackData)

	switch action {
	case ActionAccept:
		h.HandleAcceptOrder(ctx, bot, chatID, callbackData, callback.Message.MessageID)
	case ActionReject:
		h.HandleRejectOrder(ctx, bot, chatID, callbackData, callback.Message.MessageID)
	case ActionComplete:
		h.HandleCompleteOrder(ctx, bot, chatID, callbackData)
	case ActionNavigate:
		h.HandleNavigation(bot, chatID, callbackData)
	case ActionCall:
		h.HandleCallCustomer(bot, chatID, callbackData)
	case ActionSettings:
		h.HandleSettings(ctx, bot, chatID, callbackData)
	case ActionRefresh:
		h.HandleRefresh(ctx, bot, chatID, callbackData)
	case ActionMenu:
		h.HandleMenu(bot, chatID, callbackData)
	case ActionOrderDetails:
		h.HandleOrderDetails(ctx, bot, chatID, callbackData)
	case ActionBackToOrder:
		h.HandleBackToOrder(ctx, bot, chatID, callbackData)
	case ActionChangeWorkmode:
		h.HandleChangeWorkmode(ctx, bot, chatID, callbackData)
	default:
		h.HandleUnknownCommand(bot, chatID)
	}
}

// COMMAND HANDLERS

func (h *Handlers) HandleStartCommand(bot BotInterface, chatID int64, user *tgbotapi.User) {
	var message string

	if !h.assignmentManager.CheckCourierByChatID(context.Background(), chatID) {
		newCourier := &models.Courier{
			TelegramID: user.ID,
			ChatID:     chatID,
			Name:       user.FirstName + " " + user.LastName,
			Phone:      "",
			IsActive:   true,
		}

		h.assignmentManager.CreateCourier(context.Background(), newCourier)

		message = fmt.Sprintf(
			"Добро пожаловать, %s!\n\n",
			user.FirstName,
		)
	} else {
		message = fmt.Sprintf(
			"С возвращением, %s!\n\n",
			user.FirstName,
		)
	}

	message += fmt.Sprint(
		"Я - бот для курьеров доставки. Буду сопровождать вас в вашей работе.\n\n" +
			"*Основные команды:*\n" +
			"• 📋 Мои заказы - посмотреть активные заказы\n" +
			"• ℹ️ Статус - информация о вашем статусе\n" +
			"• ⚙️ Настройки - настройки уведомлений\n" +
			"• 🆘 Помощь - справка по использованию\n\n" +
			"Ожидайте новые заказы!",
	)

	keyboard := h.keyboardManager.CreateMainMenuKeyboard()
	bot.SendMessageWithKeyboard(chatID, message, keyboard)
}

func (h *Handlers) HandleHelpCommand(bot BotInterface, chatID int64) {
	message := "🆘 *Помощь по боту*\n\n" +
		"*Как работает бот:*\n" +
		"• 📦 Вы получаете уведомления о новых заказах\n" +
		"• ✅ Можете принять или отклонить заказ\n" +
		"• 🗺️ Использовать навигацию к адресу доставки\n" +
		"• 📞 Связаться с клиентом\n" +
		"• 🏁 Отмечать статусы доставки\n\n" +
		"*Основные кнопки:*\n" +
		"• ✅ Принять - взять заказ в работу\n" +
		"• ❌ Отклонить - отказаться от заказа\n" +
		"• 🗺️ Построить маршрут - открыть навигацию\n" +
		"• 📞 Позвонить - связаться с клиентом\n" +
		"• 🏁 Доставка завершена - отметить выполнение\n\n" +
		"Если возникли проблемы, обратитесь к администратору."

	bot.SendMessage(chatID, message)
}

func (h *Handlers) HandleMyOrdersCommand(ctx context.Context, bot BotInterface, chatID int64) {
	h.log.Info("Fetching active orders for courier", "ChatID", chatID)

	orders, err := h.assignmentManager.GetActiveOrdersByCourier(ctx, chatID)
	if err != nil {
		h.log.Error("Failed to get active orders for courier", "chatID", chatID, "Error", err)
		bot.SendMessage(chatID, "❌ Не удалось загрузить список заказов. Попробуйте позже.")
		return
	}

	courier, err := h.assignmentManager.GetCourierByChatID(ctx, chatID)
	if err != nil {
		h.log.Error("Failed to get courier by chat ID", "chatID", chatID, "Error", err)
		bot.SendMessage(chatID, "❌ Не удалось загрузить список заказов. Попробуйте позже.")
		return
	}

	waitingAssignments, err := h.assignmentManager.GetWaitingAssignmentsByCourierID(ctx, courier.ID)
	if err != nil {
		h.log.Error("Failed to get watiting assignments by courier ID", "chatID", chatID, "Error", err)
		bot.SendMessage(chatID, "❌ Не удалось загрузить список заказов. Попробуйте позже.")
		return
	}

	for _, waitingAssignment := range waitingAssignments {
		orderID := waitingAssignment.OrderID
		order, err := h.assignmentManager.GetOrderByID(ctx, orderID)
		if err != nil {
			h.log.Error("Failed to get order by ID", "chatID", chatID, "Error", err)
			bot.SendMessage(chatID, "❌ Не удалось загрузить список заказов. Попробуйте позже.")
			return
		}

		orders = append(orders, *order)
	}

	if len(orders) == 0 {
		message := "📋 *Ваши активные заказы*\n\n" +
			"На данный момент у вас нет активных заказов.\n\n" +
			"💡 *Совет:* Убедитесь, что ваш статус 'Активен' в настройках.\n" +
			"Новые заказы будут приходить автоматически!"
		bot.SendMessage(chatID, message)
		return
	}

	orderItems := h.convertOrdersToOrderListItem(ctx, orders)
	message := h.formatOrdersSummary(orderItems)
	keyboard := h.keyboardManager.CreateOrderListKeyboard(orderItems)

	bot.SendMessageWithInlineKeyboard(chatID, message, keyboard)
}

// ЭТО ЗАГЛУШКА, ЙОУ
func (h *Handlers) HandleStatusCommand(bot BotInterface, chatID int64) {
	message := "ℹ️ *Ваш статус*\n\n" +
		"• 📱 Статус: *Активен*\n" +
		"• 📊 Заказов сегодня: *0*\n" +
		"• ⭐ Рейтинг: *Ты заглушечка*\n\n" +
		"Вы готовы принимать новые заказы! 🚀"

	bot.SendMessage(chatID, message)
}

func (h *Handlers) HandleSettingsCommand(bot BotInterface, chatID int64) {
	message := "⚙️ *Настройки*\n\n" +
		"Выберите настройку для изменения:"

	keyboard := h.keyboardManager.CreateSettingsKeyboard()
	bot.SendMessageWithInlineKeyboard(chatID, message, keyboard)
}

func (h *Handlers) HandleUnknownCommand(bot BotInterface, chatID int64) {
	message := "❓ Неизвестная команда\n\n" +
		"Используйте кнопки меню или введите /help для справки."

	bot.SendMessage(chatID, message)
}

// CALLBACK HANDLERS

func (h *Handlers) HandleAcceptOrder(ctx context.Context, bot BotInterface, chatID int64, callbackData string, messageID int) {
	orderID, err := h.ExtractOrderID(callbackData)
	if err != nil {
		h.log.Error("Failed to extract order ID from callback", "CallbackData", callbackData)
		bot.SendMessage(chatID, "❌ Ошибка обработки заказа")
		return
	}

	h.log.Info("Courier accepting order", "chatID", chatID, "orderID", orderID)

	bot.AnswerCallbackQueryWithText("", "✅ Принимаем заказ...")

	err = h.assignmentManager.HandleCourierResponse(ctx, chatID, orderID, true)
	if err != nil {
		h.log.Error("Failed to accept order by courier", "orderID", orderID, "chatID", chatID, "error", err)
		if err.Error() != "order assignment" {
			bot.SendMessage(chatID, "❌ Не удалось принять заказ. Попробуйте позже.")
		}
		return
	}

	bot.DeleteMessage(chatID, messageID)
}

func (h *Handlers) HandleRejectOrder(ctx context.Context, bot BotInterface, chatID int64, callbackData string, messageID int) {
	orderID, err := h.ExtractOrderID(callbackData)
	if err != nil {
		h.log.Error("Failed to extract order ID from callback", "CallbackData", callbackData)
		bot.SendMessage(chatID, "❌ Ошибка обработки заказа")
		return
	}

	h.log.Info("Courier rejecting order", "chatID", chatID, "orderID", orderID)

	bot.EditMessageReplyMarkup(chatID, messageID, nil)

	err = h.assignmentManager.HandleCourierResponse(ctx, chatID, orderID, false)
	if err != nil {
		h.log.Error("Failed to reject order by courier", "orderID", orderID, "chatID", chatID, "error", err)
		if err.Error() != "order assignment" {
			bot.SendMessage(chatID, "❌ Не удалось отклонить заказ. Попробуйте позже.")
		}
		return
	}

	bot.DeleteMessage(chatID, messageID)
}

func (h *Handlers) HandleCompleteOrder(ctx context.Context, bot BotInterface, chatID int64, callbackData string) {
	orderID, err := h.ExtractOrderID(callbackData)
	if err != nil {
		h.log.Error("Failed to extract order ID from callback", "CallbackData", callbackData)
		bot.SendMessage(chatID, "❌ Ошибка обработки заказа")
		return
	}

	message := fmt.Sprintf(
		"✅ *Заказ #%d завершен!*\n\n"+
			"Поздравляем с успешной доставкой!",
		orderID,
	)

	bot.SendMessage(chatID, message)

	h.assignmentManager.UpdateOrderStatusReceived(ctx, orderID, true)
	h.log.Info("Order marked as completed by courier", "orderID", orderID, "chatID", chatID)
}

func (h *Handlers) HandleNavigation(bot BotInterface, chatID int64, callbackData string) {
	parts := strings.Split(callbackData, "_")
	if len(parts) < 3 {
		bot.SendMessage(chatID, "❌ Не удалось получить адрес для навигации")
		return
	}

	orderID := parts[1]
	address := strings.Join(parts[2:], " ")

	message := fmt.Sprintf(
		"🗺️ *Навигация для заказа #%s*\n\n"+
			"*Адрес:* %s\n\n"+
			"Откройте приложение навигации для построения маршурута.",
		orderID,
		address,
	)

	orderIDInt, err := strconv.Atoi(orderID)
	if err != nil {
		h.log.Warn("Failed to parse order ID from callback", "calback", callbackData)
		bot.SendMessage(chatID, "❌ Ошибка на стороне сервера")
	}

	keyboard := h.keyboardManager.CreateBackToOrderKeyboard(orderIDInt)

	bot.SendMessageWithInlineKeyboard(chatID, message, keyboard)
}

func (h *Handlers) HandleCallCustomer(bot BotInterface, chatID int64, callbackData string) {
	parts := strings.Split(callbackData, "_")
	if len(parts) < 3 {
		bot.SendMessage(chatID, "❌ Не удалось получить номер телефона")
		return
	}

	orderID := parts[1]
	phone := parts[2]

	message := fmt.Sprintf(
		"📞 *Звонок клиенту заказа #%s*\n\n"+
			"*Телефон:* `%s`\n\n"+
			"Нажмите на номер для звонка.",
		orderID,
		phone,
	)

	orderIDInt, err := strconv.Atoi(orderID)
	if err != nil {
		h.log.Warn("Failed to parse order ID from callback", "calback", callbackData)
		bot.SendMessage(chatID, "❌ Ошибка на стороне сервера")
	}

	keyboard := h.keyboardManager.CreateBackToOrderKeyboard(orderIDInt)

	bot.SendMessageWithInlineKeyboard(chatID, message, keyboard)
}

func (h *Handlers) HandleSettings(ctx context.Context, bot BotInterface, chatID int64, callbackData string) {
	switch callbackData {
	case SettingsWorkmode:
		courier, err := h.assignmentManager.GetCourierByChatID(ctx, chatID)
		if err != nil {
			bot.SendMessage(chatID, "❌ Ошибка доступа")
		}

		isActiveText := "Активен"
		if !courier.IsActive {
			isActiveText = "Не активен"
		}

		msg := fmt.Sprintf("⚙️ *Текущий статус: %s*", isActiveText)

		keyboard := h.keyboardManager.CreateChangeWorkmodeKeyboard(courier.IsActive)
		bot.SendMessageWithInlineKeyboard(chatID, msg, keyboard)
	case SettingsContacts:
		keyboard := h.keyboardManager.CreateBackToSettingsKeyboard()
		bot.SendMessageWithInlineKeyboard(chatID, "Контактная информация...\nУбрать может э", keyboard)
	default:
		h.HandleSettingsCommand(bot, chatID)
	}
}

func (h *Handlers) HandleRefresh(ctx context.Context, bot BotInterface, chatID int64, callbackData string) {
	h.HandleMyOrdersCommand(ctx, bot, chatID)
}

func (h *Handlers) HandleMenu(bot BotInterface, chatID int64, callbackData string) {
	h.HandleStartCommand(bot, chatID, &tgbotapi.User{FirstName: "Курьер"})
}

func (h *Handlers) HandleOrderDetails(ctx context.Context, bot BotInterface, chatID int64, callbackData string) {
	orderID, err := h.ExtractOrderID(callbackData)
	if err != nil {
		h.log.Error("Failed to extract order ID from delivery confirmation", "callbackData", callbackData)
		bot.SendMessage(chatID, "❌ Не удалось получить информацию о заказе.")
		return
	}

	order, err := h.assignmentManager.GetOrderByID(ctx, orderID)
	if err != nil {
		bot.SendMessage(chatID, "❌ Не удалось получить информацию о заказе.")
		return
	}

	message := fmt.Sprintf(
		"📋 *Детали заказа #%d*\n\n"+
			"*Статус:* %s\n"+
			"*Адрес:* %s %s\n"+
			"*Клиент:* %s\n"+
			"*Телефон:* %s\n"+
			"*Дата доставки:* %s\n\n"+
			"Используйте кнопки ниже для управления доставкой:",
		orderID,
		h.determineOrderStatus(ctx, *order),
		order.City, order.Address,
		order.Name,
		order.PhoneNumber,
		order.DeliveryDate,
	)
	
	assignment, err := h.assignmentManager.GetWaitingAssignmentsByOrderID(ctx, order.ID)
	if err != nil {
		h.log.Error("Failed to check order assignment status", "callbackData", callbackData, "error", err)
		bot.SendMessage(chatID, "❌ Не удалось получить информацию о заказе.")
		return
	}

	var keyboard tgbotapi.InlineKeyboardMarkup

	switch assignment.CourierResponseStatus {
	case "waiting":
		keyboard = h.keyboardManager.CreateAssignmentKeyboard(orderID)
	case "accepted":
		keyboard = h.keyboardManager.CreateDeliveryKeyboard(orderID, order.City+order.Address, order.PhoneNumber)
	}

	bot.SendMessageWithInlineKeyboard(chatID, message, keyboard)
}

func (h *Handlers) HandleBackToOrder(ctx context.Context, bot BotInterface, chatID int64, callbackData string) {
	orderID, err := h.ExtractOrderID(callbackData)
	if err != nil {
		bot.SendMessage(chatID, "❌ Не удалось вернуться к заказу.")
		return
	}

	h.HandleOrderDetails(ctx, bot, chatID, fmt.Sprintf("%s_%d", ActionOrderDetails, orderID))
}

func (h *Handlers) HandleChangeWorkmode(ctx context.Context, bot BotInterface, chatID int64, callbackData string) {
	parts := strings.Split(callbackData, "_")
	isActiveStatus, err := strconv.ParseBool(parts[2])
	if err != nil {
		h.log.Warn("Invalid callback data", "Error", err, "CallbakckData", callbackData)
		bot.SendMessage(chatID, "Ошибка на стороне сервера, попробуйте позже ⌛")
		return
	}

	err = h.assignmentManager.UpdateCourierStatusIsActive(ctx, chatID, isActiveStatus)
	if err != nil {
		bot.SendMessage(chatID, "Ошибка на стороне сервера, попробуйте позже ⌛")
		return
	}

	var secondPartMsg string
	if isActiveStatus {
		secondPartMsg = "Поменяйте статус в настройках на *\"активен\"*, когда захотите вернуться к работе"
	} else {
		secondPartMsg = "Теперь ваш статус *\"активен\"*, ждите уведомлений о новых заказах"
	}
	msg := "📝 Ваш статус успешно измен\n\n" + secondPartMsg

	bot.SendMessage(chatID, msg)
}

// UTILITY METHODS

func (h *Handlers) ExtractOrderID(callbackData string) (int, error) {
	parts := strings.Split(callbackData, "_")
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid callback data format: %s", callbackData)
	}

	for i := len(parts) - 1; i >= 0; i-- {
		if id, err := strconv.Atoi(parts[i]); err == nil {
			return id, nil
		}
	}

	return 0, fmt.Errorf("order ID not found in callback data: %s", callbackData)
}

func (h *Handlers) convertOrdersToOrderListItem(ctx context.Context, orders []models.Order) []OrderListItem {
	var items []OrderListItem

	for _, order := range orders {
		status := h.determineOrderStatus(ctx, order)
		item := OrderListItem{
			ID:      order.ID,
			Status:  status,
			Address: fmt.Sprintf("%s, %s", order.Address, order.City),
			Time:    h.formatDeliveryTime(order.DeliveryDate),
			Price:   order.FinalPrice,
		}

		items = append(items, item)
	}

	return items
}

func (h *Handlers) determineOrderStatus(ctx context.Context, order models.Order) string {
	assignment, err := h.assignmentManager.GetAssignmentByOrderID(ctx, order.ID)

	if err != nil || assignment == nil {
		return "⏳ Ожидает подтверждения"
	}

	switch assignment.CourierResponseStatus {
	case "waiting":
		return "⏳ Ожидает подтверждения"
	case "accepted":
		return "✅ Принят в работу"
	default:
		return "📋 В обработке"
	}
}

func (h *Handlers) formatDeliveryTime(deliveryTime *time.Time) string {
	if deliveryTime == nil {
		return "⏰ Время не указано"
	}

	now := time.Now()
	delivery := *deliveryTime

	diff := delivery.Sub(now)

	if diff <= 0 {
		return "🚨 СРОЧНО! Просрочен"
	}

	if diff <= time.Hour {
		minutes := int(diff.Minutes())
		if minutes <= 0 {
			return "🚨 СРОЧНО! Просрочен"
		}
		return fmt.Sprintf("🚨 через %d мин", minutes)
	}

	if delivery.Year() == now.Year() && delivery.Month() == now.Month() && delivery.Day() == now.Day() {
		return fmt.Sprintf("🕐 Сегодня в %s", delivery.Format("15:04"))
	}

	tomorrow := now.Add(24 * time.Hour)
	if delivery.Year() == tomorrow.Year() && delivery.Month() == tomorrow.Month() && delivery.Day() == tomorrow.Day() {
		return fmt.Sprintf("📅 Завтра в %s", delivery.Format("15:04"))
	}

	weekLater := now.Add(7 * 24 * time.Hour)
	if delivery.Before(weekLater) {
		weekday := h.getRussianWeekday(delivery.Weekday())
		return fmt.Sprintf("📅 %s в %s", weekday, delivery.Format("15:04"))
	}

	return fmt.Sprintf("📅 %s", delivery.Format("02.01 в 15:04"))
}

func (h *Handlers) getRussianWeekday(weekday time.Weekday) string {
	days := map[time.Weekday]string{
		time.Monday:    "Пн",
		time.Tuesday:   "Вт",
		time.Wednesday: "Ср",
		time.Thursday:  "Чт",
		time.Friday:    "Пт",
		time.Saturday:  "Сб",
		time.Sunday:    "Вс",
	}

	return days[weekday]
}

func (h *Handlers) formatOrdersSummary(orderItems []OrderListItem) string {
	var waitingCount, acceptCount int

	for _, item := range orderItems {
		switch item.Status {
		case "⏳ Ожидает подтверждения":
			waitingCount++
		case "✅ Принят в работу":
			acceptCount++
		}
	}

	total := len(orderItems)

	summary := fmt.Sprintf(
		"📋 *Ваши активные заказы*\n\n"+
			"📊 *Статистика:*\n"+
			"• ⏳ Ожидают подтверждения: %d\n"+
			"• ✅ Приняты в работу: %d\n"+
			"• 📈 Всего активных: %d\n\n",
		waitingCount,
		acceptCount,
		total,
	)

	summary += "Выберите заказ для просмотра деталей:"

	return summary
}
