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
	assignmentService *assignment.Service
	keyboardManager   KeyboardManagerInterface
	log               *slog.Logger
}

func NewHandlers(assignmentService *assignment.Service, keyboardManager KeyboardManagerInterface, log *slog.Logger) *Handlers {
	return &Handlers{
		assignmentService: assignmentService,
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
	case "📋 Мои заказы":
		h.HandleMyOrdersCommand(ctx, bot, chatID)
	case "ℹ️ Статус":
		h.HandleStatusCommand(bot, chatID)
	case "⚙️ Настройки":
		h.HandleSettingsCommand(bot, chatID)
	default:
		h.HandleUnknownCommand(bot, chatID)
	}
}

// COMMAND HANDLERS

func (h *Handlers) HandleStartCommand(bot BotInterface, chatID int64, user *tgbotapi.User) {
	message := fmt.Sprintf(
		"Доброго времени суток, %s!\n\n"+
			"Я - бот для курьеров доставки. Буду сопровождать вас в вашей работе.\n\n"+
			"*Основные команды:*\n"+
			"• 📋 Мои заказы - посмотреть активные заказы\n"+
			"• ℹ️ Статус - информация о вашем статусе\n"+
			"• ⚙️ Настройки - настройки уведомлений\n"+
			"• 🆘 Помощь - справка по использованию\n\n"+
			"Ожидайте новые заказы!",
		user.FirstName,
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

	orders, err := h.assignmentService.GetActiveOrdersByCourier(ctx, chatID)
	if err != nil {
		h.log.Error("Failed to get active orders for courier", "chatID", chatID, "Error", err)
		bot.SendMessage(chatID, "❌ Не удалось загрузить список заказов. Попробуйте позже.")
		return
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
	assignment, err := h.assignmentService.GetByOrderID(ctx, order.ID)

	if err != nil || assignment == nil {
		return "⏳ Ожидает подтверждения"
	}

	switch assignment.CourierResponseStatus {
	case "waiting":
		return "⏳ Ожидает ответа"
	case "accepted":
		switch {
		case order.IsReceived:
			return "✅ Доставлен"
		case h.isDeliveryInProgerss(order):
			return "🚗 В доставке"
		default:
			return "✅ Принят в работу"
		}
	case "rejected":
		return "❌ Отклонен"
	case "expired":
		return "⏰ Время истекло"
	default:
		return "📋 В обработке"
	}
}

func (h *Handlers) isDeliveryInProgerss(order models.Order) bool {
	if order.DeliveryDate == nil {
		return false
	}

	now := time.Now()
	deliveryTime := *order.DeliveryDate

	timeUntilDelivery := deliveryTime.Sub(now)
	return timeUntilDelivery <= 2*time.Hour || deliveryTime.Before(now)
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
	var waitingCount, acceptCount, deliveryCount int

	for _, item := range orderItems {
		switch item.Status {
		case "⏳ Ожидает ответа", "⏳ Ожидает подтверждения":
			waitingCount++
		case "✅ Принят в работу":
			acceptCount++
		case "🚗 В доставке":
			deliveryCount++
		}
	}

	total := len(orderItems)

	summary := fmt.Sprintf(
		"📋 *Ваши активные заказы*\n\n"+
			"📊 *Статистика:*\n"+
			"• ⏳ Ожидают подтверждения: %d\n"+
			"• ✅ Приняты в работу: %d\n"+
			"• 🚗 В доставке: %d\n"+
			"• 📈 Всего активных: %d\n\n",
		waitingCount,
		acceptCount,
		deliveryCount,
		total,
	)

	summary += "Выберите заказ для просмотра деталей:"

	return summary
}

// ЭТО ЗАГЛУШКА, ЙОУ
func (h *Handlers) HandleStatusCommand(bot BotInterface, chatID int64) {
	message := "ℹ️ *Ваш статус*\n\n" +
		"• 📱 Статус: *Активен*\n" +
		"• 🚗 Доступен для заказов: *Да*\n" +
		"• 📊 Заказов сегодня: *0*\n" +
		"• ⭐ Рейтинг: *Новый курьер*\n\n" +
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

func (h *Handlers) HandleAcceptOrder(ctx context.Context, bot BotInterface, chatID int64, callbackData string, messageID int64) {
	orderID, err := h.ExtractOrderID(callbackData)
	if err != nil {
		h.log.Error("Failed to extract order ID from callback", "CallbackData", callbackData)
		bot.SendMessage(chatID, "❌ Ошибка обработки заказа")
		return
	}

	h.log.Info("Courier accepting order", "chatID", chatID, "orderID", orderID)

	bot.EditMessageReplyMarkup(chatID, messageID, nil)

	err = h.assignmentService.HandleCourierResponse(ctx, chatID, orderID, true)
	if err != nil {
		h.log.Error("Failed to accept order by courier", "orderID", orderID, "chatID", chatID, "error", err)
		bot.SendMessage(chatID, "❌ Не удалось принять заказ. Попробуйте позже.")
		return
	}
}

func (h *Handlers) HandleRejectOrder(ctx context.Context, bot BotInterface, chatID int64, callbackData string, messageID int64) {
	orderID, err := h.ExtractOrderID(callbackData)
	if err != nil {
		h.log.Error("Failed to extract order ID from callback", "CallbackData", callbackData)
		bot.SendMessage(chatID, "❌ Ошибка обработки заказа")
		return
	}

	h.log.Info("Courier rejecting order", "chatID", chatID, "orderID", orderID)

	bot.EditMessageReplyMarkup(chatID, messageID, nil)

	err = h.assignmentService.HandleCourierResponse(ctx, chatID, orderID, false)
	if err != nil {
		h.log.Error("Failed to reject order by courier", "orderID", orderID, "chatID", chatID, "error", err)
		bot.SendMessage(chatID, "❌ Не удалось отклонить заказ. Попробуйте позже.")
		return
	}
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

	h.assignmentService.UpdateOrderStatusReceived(ctx, orderID, true)
	h.log.Info("Order marked as completed by courier", "orderID", orderID, "chatID", chatID)
}

func (h *Handlers) HandleProblemOrder(bot BotInterface, chatID int64, callbackData string) {
	orderID, err := h.ExtractOrderID(callbackData)
	if err != nil {
		h.log.Error("Failed to extract order ID from callback", "Callback", callbackData)
		bot.SendMessage(chatID, "❌ Ошибка обработки заказа")
		return
	}

	message := fmt.Sprintf(
		"🚨 *Проблема с заказом #%d*\n\n"+
			"Выберите тип проблемы:",
		orderID,
	)

	keyboard := h.keyboardManager.CreateProblemKeyboard(orderID)
	bot.SendMessageWithInlineKeyboard(chatID, message, keyboard)
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

	bot.SendMessage(chatID, message)
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

	bot.SendMessage(chatID, message)
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
