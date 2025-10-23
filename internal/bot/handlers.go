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
		h.HandleAcceptOrder(ctx, bot, chatID, callbackData, int64(callback.Message.MessageID))
	case ActionReject:
		h.HandleRejectOrder(ctx, bot, chatID, callbackData, int64(callback.Message.MessageID))
	case ActionComplete:
		h.HandleCompleteOrder(ctx, bot, chatID, callbackData)
	case ActionProblem:
		h.HandleProblemOrder(bot, chatID, callbackData)
	case ActionNavigate:
		h.HandleNavigation(bot, chatID, callbackData)
	case ActionCall:
		h.HandleCallCustomer(bot, chatID, callbackData)
	case ActionStatus:
		h.HandleStatusUpdate(ctx, bot, chatID, callbackData)
	case ActionSettings:
		h.HandleSettings(bot, chatID, callbackData)
	case ActionConfirm:
		h.HandleConfirmation(bot, chatID, callbackData)
	case ActionRefresh:
		h.HandleRefresh(ctx, bot, chatID, callbackData)
	case ActionMenu:
		h.HandleMenu(bot, chatID, callbackData)
	case ActionOrderDetails:
		h.HandleOrderDetails(ctx, bot, chatID, callbackData)
	case ActionBackToOrder:
		h.HandleBackToOrder(ctx, bot, chatID, callbackData)
	case ActionConfirmDelivery:
		h.HandleDeliveryConfirmation(ctx, bot, chatID, callbackData)
	case ActionCancelDelivery:
		h.HandleDeliveryCancel(ctx, bot, chatID, callbackData)
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

// ЭТО ЗАГЛУШКА, ЙОУ
func (h *Handlers) HandleStatusCommand(bot BotInterface, chatID int64) {
	message := "ℹ️ *Ваш статус*\n\n" +
		"• 📱 Статус: *Активен*\n" +
		"• 🚗 Доступен для заказов: *Да*\n" +
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

func (h *Handlers) HandleStatusUpdate(ctx context.Context, bot BotInterface, chatID int64, callbackData string) {
	h.log.Info("Processing status update from courier", "chatID", chatID, "callbackData", callbackData)

	action, orderID, err := h.parseStatusCallback(callbackData)
	if err != nil {
		h.log.Error("Failed to parse status callback", "chatID", chatID, "callbackData", callbackData, "error", err)
		bot.SendMessage(chatID, "❌ Ошибка обработки команды, попробуйте еще раз.")
		return
	}

	order, err := h.assignmentService.GetOrderByID(ctx, orderID)
	if err != nil {
		h.log.Error("Failed to get order", "orderID", orderID, "error", err)
		bot.SendMessage(chatID, "❌ Не удалось найти заказ.")
		return
	}

	courier, err := h.assignmentService.GetCourierByChatID(ctx, chatID)
	if err != nil {
		h.log.Error("Failed to get courier", "chatID", chatID, "error", err)
		bot.SendMessage(chatID, "❌ Ошибка проверки доступа")
		return
	}

	if order.CourierID == nil || *order.CourierID != courier.ID {
		bot.SendMessage(chatID, "❌ Этот заказ не назначен вам.")
		return
	}

	switch action {
	case "status_picked":
		h.handleOrderPicked(bot, chatID, orderID, order)
	case "status_delivering":
		h.handleOrderDelivering(bot, chatID, orderID, order)
	case "status_arrived":
		h.handleOrderArrived(bot, chatID, orderID, order)
	case "status_delivered":
		h.handleOrderDelivered(ctx, bot, chatID, orderID)
	default:
		bot.SendMessage(chatID, "❌ Неизвестное действие.")
		return
	}
}

func (h *Handlers) HandleSettings(bot BotInterface, chatID int64, callbackData string) {
	switch callbackData {
	case SettingsNotifications:
		bot.SendMessage(chatID, "🔔 Настройки уведомлений...\nУбрать может э")
	case SettingsWorkmode:
		bot.SendMessage(chatID, "Настройки режима работы...\nУбрать может э")
	case SettingsContacts:
		bot.SendMessage(chatID, "Контактная информация...\nУбрать может э")
	default:
		h.HandleSettingsCommand(bot, chatID)
	}
}

func (h *Handlers) HandleConfirmation(bot BotInterface, chatID int64, callbackData string) {
	bot.SendMessage(chatID, "✅ Действие подтверждено")
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

	order, err := h.assignmentService.GetOrderByID(ctx, orderID)
	if err != nil {
		bot.SendMessage(chatID, "❌ Не удалось получить информацию о заказе.")
		return
	}

	message := fmt.Sprintf(
		"📋 *Детали заказа #%d*\n\n"+
			"*Статус:* %s\n"+
			"*Адрес:* %s %s\n"+
			"*Клиент:* %s %s\n"+
			"*Телефон:* %s\n"+
			"*Дата доставки:* %s\n\n"+
			"Используйте кнопки ниже для управления доставкой:",
		orderID,
		h.determineOrderStatus(ctx, *order),
		order.City, order.Address,
		order.Surname, order.Name,
		order.PhoneNumber,
		order.DeliveryDate,
	)

	keyboard := h.keyboardManager.CreateDeliveryKeyboard(orderID, order.City+order.Address, order.PhoneNumber)
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

func (h *Handlers) HandleDeliveryConfirmation(ctx context.Context, bot BotInterface, chatID int64, callbackData string) {
	orderID, err := h.ExtractOrderID(callbackData)
	if err != nil {
		h.log.Error("Failed to extract order ID from delivery confirmation", "callbackData", callbackData)
		bot.SendMessage(chatID, "❌ Ошибка подтверждения заказа.")
		return
	}

	err = h.assignmentService.UpdateOrderStatusReceived(ctx, orderID, true)
	if err != nil {
		h.log.Error("Failed to mark order as delivered", "orderID", orderID, "error", err)
		bot.SendMessage(chatID, "❌ Не удалось обновить статус заказа.")
		return
	}

	message := fmt.Sprintf(
		"🎉 *Заказ #%d доставлен!*\n\n"+
			"✅ Доставка успешно завершена и подтверждена!\n\n"+
			"Спасибо за вашу работу!",
		orderID,
	)

	bot.SendMessage(chatID, message)
	h.log.Info("Order confirmed as delivered by courier", "orderID", orderID, "chatID", chatID)

	h.showNextActions(bot, chatID)
}

func (h *Handlers) HandleDeliveryCancel(ctx context.Context, bot BotInterface, chatID int64, callbackData string) {
	orderID, err := h.ExtractOrderID(callbackData)
	if err != nil {
		h.log.Error("Failed to extract order ID from delivery confirmation", "callbackData", callbackData)
		bot.SendMessage(chatID, "❌ Ошибка подтверждения заказа.")
		return
	}

	message := fmt.Sprintf(
		"ℹ️ *Подтверждение доставки отменено*\n\n"+
			"Заказ #%d остается активным.\n\n"+
			"Вы можете завершить доставку позже или сообщить о проблеме.",
		orderID,
	)

	order, err := h.assignmentService.GetOrderByID(ctx, orderID)
	if err == nil && order != nil {
		keyboard := h.keyboardManager.CreateDeliveryKeyboard(orderID, order.City+order.Address, order.PhoneNumber)
		bot.SendMessageWithInlineKeyboard(chatID, message, keyboard)
	} else {
		bot.SendMessage(chatID, message)
	}

	h.log.Info("Delivery confirmation cancelled for order by courier", "orderID", orderID, "chatID", chatID)
}

// STATUS UPDATE HANDLERS

func (h *Handlers) handleOrderPicked(bot BotInterface, chatID int64, orderID int, order *models.Order) {
	message := fmt.Sprintf(
		"📦 *Заказ #%d забран!*\n\n"+
			"✅ Вы успешно забрали заказ у ресторана.\n\n"+
			"*Информация о заказе:*\n"+
			"• Адрес доставки: %s, %s\n"+
			"• Клиент: %s %s\n"+
			"• Телефон: `%s`\n\n"+
			"🚗 Теперь можете начать доставку к клиенту.",
		orderID,
		order.Address, order.City,
		order.Surname, order.Name,
		order.PhoneNumber,
	)

	keyboard := h.keyboardManager.CreateDeliveryKeyboard(orderID, order.Address, order.PhoneNumber)
	bot.SendMessageWithInlineKeyboard(chatID, message, keyboard)

	h.log.Info("Courier picked up order", "chatID", chatID, "orderID", orderID)
}

func (h *Handlers) handleOrderDelivering(bot BotInterface, chatID int64, orderID int, order *models.Order) {
	message := fmt.Sprintf(
		"🚗 *Заказ #%d в пути!*\n\n"+
			"📍 Вы направляетесь к клиенту.\n\n"+
			"*Рекомендации:*\n"+
			"• 🗺️ Используйте навигацию для оптимального маршрута\n"+
			"• 📞 Свяжитесь с клиентом за 10-15 минут до прибытия\n"+
			"• ⏱️ Учитывайте текущую дорожную ситуацию\n\n"+
			"Ориентировочное время прибытия: *15-20 минут*",
		orderID,
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗺️ Построить маршрут", fmt.Sprintf("nav_%d_%s", orderID, h.keyboardManager.EscapeCallbackData(order.Address))),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📞 Позвонить клиенту", fmt.Sprintf("call_%d_%s", orderID, order.PhoneNumber)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📍 Я на месте", fmt.Sprintf("status_arrived_%d", orderID)),
		),
	)

	bot.SendMessageWithInlineKeyboard(chatID, message, keyboard)
	h.log.Info("Courier started delivering order", "chatID", chatID, "orderID", orderID)
}

func (h *Handlers) handleOrderArrived(bot BotInterface, chatID int64, orderID int, order *models.Order) {
	message := fmt.Sprintf(
		"📍 *Вы на месте!*\n\n"+
			"Заказ #%d готов к передаче клиенту.\n\n"+
			"*Действия:*\n"+
			"1. 📞 Позвоните клиенту для встречи\n"+
			"2. ✅ Передайте заказ\n"+
			"3. 💰 Примите оплату (если необходимо)\n"+
			"4. 🏁 Подтвердите доставку\n\n"+
			"Клиент: %s %s\n"+
			"Телефон: `%s`",
		orderID,
		order.Surname, order.Name,
		order.PhoneNumber,
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📞 Позвонить клиенту", fmt.Sprintf("call_%d_%s", orderID, order.PhoneNumber)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Доставка завершена", fmt.Sprintf("status_delivered_%d", orderID)),
			tgbotapi.NewInlineKeyboardButtonData("Возникли проблемы", fmt.Sprintf("problem_%d", orderID)),
		),
	)

	bot.SendMessageWithInlineKeyboard(chatID, message, keyboard)
	h.log.Info("Courier arrived with order", "chatID", chatID, "orderID", orderID)
}

func (h *Handlers) handleOrderDelivered(ctx context.Context, bot BotInterface, chatID int64, orderID int) {
	h.assignmentService.UpdateOrderStatusReceived(ctx, orderID, true)

	message := fmt.Sprintf(
		"🏁 *Подтверждение доставки*\n\n"+
			"Заказ #%d готов к отметке как доставленный.\n\n"+
			"*Пожалуйста, подтвердите:*\n"+
			"✅ Заказ передан клиенту\n"+
			"✅ Оплата получена (если требуется)\n"+
			"После подтверждения заказ будет завершен.",
		orderID,
	)

	keyboard := h.keyboardManager.CreateConfirmationKeyboard("delivery", orderID)
	bot.SendMessageWithInlineKeyboard(chatID, message, keyboard)
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

func (h *Handlers) showNextActions(bot BotInterface, chatID int64) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Мои заказы", "my_orders"),
			tgbotapi.NewInlineKeyboardButtonData("📊 Статистика", "statistics"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Новый заказ", "refresh_orders"),
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "settings"),
		),
	)

	bot.SendMessageWithInlineKeyboard(chatID, "Что дальше?", keyboard)
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
	assignment, err := h.assignmentService.GetAssignmentByOrderID(ctx, order.ID)

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

func (h *Handlers) parseStatusCallback(callbackData string) (action string, orderID int, err error) {
	parts := strings.Split(callbackData, "_")

	if len(parts) < 3 {
		return "", 0, fmt.Errorf("invalid callback format: %s", callbackData)
	}

	action = strings.Join(parts[:2], "_")

	orderID, err = strconv.Atoi(parts[2])
	if err != nil {
		return "", 0, fmt.Errorf("invalid order id in callback: %s", parts[2])
	}

	return action, orderID, nil
}
