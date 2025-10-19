package bot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotInterface interface {
	SendMessage(chatID int64, text string) error
	SendMessageWithKeyboard(chatID int64, text string, keyboard tgbotapi.ReplyKeyboardMarkup) error
	SendMessageWithInlineKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error

	EditMessageText(chatID int64, messageID int, text string) error
	EditMessageReplyMarkup(chatID int64, messageID int, replyMarkup interface{}) error

	AnswerCallbackQuery(callbackQueryID string) error
	AnswerCallbackQueryWithText(callbackQueryID, text string) error

	SendPhoto(chatID int64, photoURL string, caption string) error
	SendLocation(chatID int64, latitude, longitude float64) (*tgbotapi.ChatMember, error)

	GetChat(chatID int64) (*tgbotapi.Chat, error)
	GetChatMember(chatID int64, userID int64) (*tgbotapi.ChatMember, error)

	SetMyCommands(commands []tgbotapi.BotCommand) error
	GetMe() (*tgbotapi.User, error)
	TestConnection() error

	SetDefaultCommands() error
}

type KeyboardManagerInterface interface {
	CreateAssignmentKeyboard(orderID int) tgbotapi.InlineKeyboardMarkup
	CreateDeliveryKeyboard(orderID int, address, phone string) tgbotapi.InlineKeyboardMarkup
	CreateStatusKeyboard(orderID int) tgbotapi.InlineKeyboardMarkup
	CreateMainMenuKeyboard() tgbotapi.ReplyKeyboardMarkup
	CreateSettingsKeyboard() tgbotapi.InlineKeyboardMarkup
	CreateConfirmationKeyboard(action string, id int) tgbotapi.InlineKeyboardMarkup
	CreateOrderListKeyboard(orders []OrderListItem) tgbotapi.InlineKeyboardMarkup
	CreateProblemKeyboard(orderID int) tgbotapi.InlineKeyboardMarkup
	CreateYesNoKeyboard(action string, id int) tgbotapi.InlineKeyboardMarkup
	RemoveKeyboard() tgbotapi.ReplyKeyboardRemove

	GetActionFromCallback(callbackData string) string
	GetSubActionFromCallback(callbackData string) string
	ParseCallbackData(callbackData string) (action string, id int, err error)

	EscapeCallbackData(data string) string
}

type HandlersInterface interface {
	HandleMessage(bot BotInterface, update tgbotapi.Update)
	HandleCallback(bot BotInterface, update tgbotapi.Update)

	HandleStartCommand(bot BotInterface, chatID int64, user *tgbotapi.User)
	HandleHelpCommand(bot BotInterface, chatID int64)
	HandleMyOrdersCommand(bot BotInterface, chatID int64)
	HandleStatusCommand(bot BotInterface, chatID int64)
	HandleSettingsCommand(bot BotInterface, chatID int64)
	HandleUnknownCommand(bot BotInterface, chatID int64)

	HandleAcceptOrder(bot BotInterface, chatID int64, callbackData string, messageID int)
	HandleRejectOrder(bot BotInterface, chatID int64, callbackData string, messageID int)
	HandleCompleteOrder(bot BotInterface, chatID int64, callbackData string)
	HandleProblemOrder(bot BotInterface, chatID int64, callbackData string)
	HandleNavigation(bot BotInterface, chatID int64, callbackData string)
	HanldeCallCustomeer(bot BotInterface, chatID int64, callbackData string)
	HandleStatusUpdate(bot BotInterface, chatID int64, callbackData string)
	HanldeSettings(bot BotInterface, chatID int64, callbackData string)
	HandleConfirmation(bot BotInterface, chatID int64, callbackData string)
	HanldeRefresh(bot BotInterface, chatID int64, callbackData string)
	HandleMenu(bot BotInterface, chatID int64, callbackData string)
	HandleOrderDetails(bot BotInterface, chatID int64, callbackData string)
	HandleBackToOrder(bot BotInterface, chatID int64, callbackData string)
	HandleUnknownCallback(bot BotInterface, chatID int64, callbackData string)

	ExtractOrderID(callbackData string) (int, error)
}

const (
	// Main Actions
	ActionAccept   = "accept"
	ActionReject   = "reject"
	ActionComplete = "complete"
	ActionProblem  = "problem"

	// Utility Actions
	ActionNavigate = "nav"
	ActionCall     = "call"
	ActionStatus   = "status"
	ActionSettings = "settings"
	ActionConfirm  = "confirm"
	ActionCancel   = "cancel"
	ActionRefresh  = "refresh"
	ActionMenu     = "menu"

	// Sub-actions
	ActionOrderDetails = "order_details"
	ActionBackToOrder  = "back_to_order"

	// Problem Sub-types
	ProblemNoAnswer     = "problem_noanswer"
	ProblemWrongAddress = "problem_wrongaddress"
	ProblemPayment      = "problem_payment"
	ProblemTechnical    = "problem_technical"
	ProblemOther        = "problem_other"

	// Status Sub-types
	StatusPicked     = "status_picked"
	StatusDelivering = "status_delivering"
	StatusArrived    = "status_arrived"
	StatusDelivered  = "status_delivered"

	// Settings Sub-types
	SettingsNotifications = "settings_notifications"
	SettingsWorkmode      = "settings_workmode"
	SettingsContacts      = "settings_contacts"

	// Menu Sub-types
	MenuMain = "menu_main"
)

type OrderListItem struct {
	ID     int
	Status string
}

type BotConfig struct {
	Token   string
	Debug   bool
	Timeout int // in seconds
}
