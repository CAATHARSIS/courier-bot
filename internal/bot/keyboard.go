package bot

import (
	"fmt"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type KeyboardManager struct {
	log *slog.Logger
}

func NewkeyboardManager(log *slog.Logger) *KeyboardManager {
	return &KeyboardManager{
		log: log,
	}
}

func (km *KeyboardManager) CreateAssignmentKeyboard(orderID int) tgbotapi.InlineKeyboardMarkup {
	km.log.Debug("Creating assignment keyboard for order", "orderID", orderID)

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Принять заказ", fmt.Sprintf("%s_%d", ActionAccept, orderID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отклонить заказ", fmt.Sprintf("%s_%d", ActionReject, orderID)),
		),
	)
}

func (km *KeyboardManager) CreateDeliveryKeyboard(orderID int, address, phone string) tgbotapi.InlineKeyboardMarkup {
	km.log.Debug("Creating delivery keyboard for order", "orderID", orderID)

	rows := [][]tgbotapi.InlineKeyboardButton{}

	if address != "" {
		navigationRow := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗺️ Построить маршрут", fmt.Sprintf("nav_%d_%s", orderID, km.escapeCallbackData(address))),
		)
		rows = append(rows, navigationRow)
	}

	if phone != "" {
		callRow := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📞 Позвонить клиенту", fmt.Sprintf("call_%d_%s", orderID, phone)),
		)
		rows = append(rows, callRow)
	}

	completionRow := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🏁 Доставка завершена", fmt.Sprintf("%s_%d", ActionComplete, orderID)),
		tgbotapi.NewInlineKeyboardButtonData("🚨 Проблема с доставкой", fmt.Sprintf("%s_%d", ActionProblem, orderID)),
	)
	rows = append(rows, completionRow)

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func (km *KeyboardManager) CreateStatusKeyboard(orderID int) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚗 Забрал заказ", fmt.Sprintf("status_picked_%d", orderID)),
			tgbotapi.NewInlineKeyboardButtonData("🚚 В пути", fmt.Sprintf("status_delivery_%d", orderID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📍 На месте", fmt.Sprintf("status_arrived_%d", orderID)),
			tgbotapi.NewInlineKeyboardButtonData("✅ Доставлено", fmt.Sprintf("status_delivered_%d", orderID)),
		),
	)
}

func (km *KeyboardManager) CreateMainMenuKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📋 Мои заказы"),
			tgbotapi.NewKeyboardButton("ℹ️ Статус"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("⚙️ Настройки"),
			tgbotapi.NewKeyboardButton("🆘 Помощь"),
		),
	)
}

func (km *KeyboardManager) CreateSettingsKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔔 Уведомления", "settings_notifications"),
			tgbotapi.NewInlineKeyboardButtonData("Режим работы", "settings_workmode"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Контакты", "seetings_contacts"),
			tgbotapi.NewInlineKeyboardButtonData("↩️ Назад", "menu_main"),
		),
	)
}

func (km *KeyboardManager) CreateConfirmationKeyboard(action string, data interface{}) tgbotapi.InlineKeyboardMarkup {
	callbackData := fmt.Sprintf("%s_confirm_%v", action, data)

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Подтвердить", callbackData),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cancel_action"),
		),
	)
}

func (km *KeyboardManager) CreateOrderListKeyboard(orders []OrderListItem) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	for _, order := range orders {
		row := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("📦 Заказ #%d - %s", order.ID, order.Status),
				fmt.Sprintf("order_details_%d", order.ID),
			),
		)
		rows = append(rows, row)
	}

	refreshRow := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить", "refresh_orders"),
		tgbotapi.NewInlineKeyboardButtonData("↩️ Назад", "menu_main"),
	)
	rows = append(rows, refreshRow)

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func (km *KeyboardManager) CreateProblemKeyboard(orderID int) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📞 Клиент не отвечает", fmt.Sprintf("problem_noanswer_%d", orderID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Неверный адрес", fmt.Sprintf("problem_wrongaddress_%d", orderID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💳 Проблема с оплатой", fmt.Sprintf("problem_payment_%d", orderID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚗 Технические проблемы", fmt.Sprintf("problem_technical_%d", orderID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❔ Другое", fmt.Sprintf("problem_other_%d", orderID)),
			tgbotapi.NewInlineKeyboardButtonData("↩️ Назад", fmt.Sprintf("back_to_order_%d", orderID)),
		),
	)
}

func (km *KeyboardManager) CreateYesNoKeyboard(action string, id int) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Да", fmt.Sprintf("%s_yes_%d", action, id)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Нет", fmt.Sprintf("%s_no_%d", action, id)),
		),
	)
}

func (km *KeyboardManager) RemoveKeyboard() tgbotapi.ReplyKeyboardRemove {
	return tgbotapi.NewRemoveKeyboard(true)
}

func (km *KeyboardManager) ParseCallbackData(callbackData string) (action string, id int, err error) {
	parts := strings.Split(callbackData, "_")
	if len(parts) < 2 {
		return "", 0, fmt.Errorf("invalid callback data format: %s", callbackData)
	}

	action = parts[0]

	if len(parts) > 1 {
		var extractedID int
		_, err := fmt.Sscanf(parts[len(parts)-1], "%d", &extractedID)
		if err == nil {
			id = extractedID
		}
	}

	return action, id, nil
}

// escapeCallbackData экранирует данные для callback_data
// В Telegram callback_data не может превышать 64 байта и содержать некоторые символы
func (km *KeyboardManager) escapeCallbackData(data string) string {
	if len(data) > 50 {
		data = data[:50]
	}

	replacements := map[string]string{
		" ":  "_",
		"\n": "",
		"\t": "",
		",":  "",
		".":  "",
		":":  "",
		";":  "",
	}

	for old, new := range replacements {
		data = strings.ReplaceAll(data, old, new)
	}

	return data
}

func (km *KeyboardManager) GetActionFromCallback(callback string) string {
	prefixes := []string{
		ActionAccept,
		ActionReject,
		ActionComplete,
		ActionProblem,
		"nav",
		"call",
		"status",
		"settings",
		"confirm",
		"cancel",
		"refresh",
		"menu",
		"order_details",
		"back_to_order",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(callback, prefix) {
			return prefix
		}
	}
	return "unknown"
}
