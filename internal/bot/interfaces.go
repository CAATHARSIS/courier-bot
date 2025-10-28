package bot

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotInterface interface {
	SendMessage(chatID int64, text string) error
	SendMessageWithKeyboard(chatID int64, text string, keyboard tgbotapi.ReplyKeyboardMarkup) error
	SendMessageWithInlineKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error

	EditMessageText(chatID int64, messageID int, text string) error
	EditMessageReplyMarkup(chatID int64, messageID int, replyMarkup interface{}) error

	DeleteMessage(chatID int64, messageID int)

	AnswerCallbackQuery(callbackQueryID string) error
	AnswerCallbackQueryWithText(callbackQueryID, text string) error

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
	CreateConfirmationKeyboard(action string, data interface{}) tgbotapi.InlineKeyboardMarkup
	CreateOrderListKeyboard(orders []OrderListItem) tgbotapi.InlineKeyboardMarkup
	CreateProblemKeyboard(orderID int) tgbotapi.InlineKeyboardMarkup
	CreateYesNoKeyboard(action string, id int) tgbotapi.InlineKeyboardMarkup
	CreateChangeWorkmodeKeyboard(isActive bool) tgbotapi.InlineKeyboardMarkup
	RemoveKeyboard() tgbotapi.ReplyKeyboardRemove

	GetActionFromCallback(callbackData string) string
	ParseCallbackData(callbackData string) (action string, id int, err error)

	EscapeCallbackData(data string) string
}

type HandlersInterface interface {
	HandleMessage(ctx context.Context, bot BotInterface, update tgbotapi.Update)
	HandleCallback(ctx context.Context, bot BotInterface, update tgbotapi.Update)

	HandleStartCommand(bot BotInterface, chatID int64, user *tgbotapi.User)
	HandleHelpCommand(bot BotInterface, chatID int64)
	HandleMyOrdersCommand(ctx context.Context, bot BotInterface, chatID int64)
	HandleStatusCommand(bot BotInterface, chatID int64)
	HandleSettingsCommand(bot BotInterface, chatID int64)
	HandleUnknownCommand(bot BotInterface, chatID int64)

	HandleAcceptOrder(ctx context.Context, bot BotInterface, chatID int64, callbackData string, messageID int64)
	HandleRejectOrder(ctx context.Context, bot BotInterface, chatID int64, callbackData string, messageID int64)
	HandleCompleteOrder(ctx context.Context, bot BotInterface, chatID int64, callbackData string)
	HandleProblemOrder(bot BotInterface, chatID int64, callbackData string)
	HandleNavigation(bot BotInterface, chatID int64, callbackData string)
	HanldeCallCustomeer(bot BotInterface, chatID int64, callbackData string)
	HandleChangeWorkmode(ctx context.Context, bot BotInterface, chatID int64, callbackData string)

	HandleStatusUpdate(ctx context.Context, bot BotInterface, chatID int64, callbackData string)
	HanldeSettings(ctx context.Context, ot BotInterface, chatID int64, callbackData string)
	HandleConfirmation(bot BotInterface, chatID int64, callbackData string)
	HanldeRefresh(ctx context.Context, bot BotInterface, chatID int64, callbackData string)
	HandleMenu(bot BotInterface, chatID int64, callbackData string)
	HandleOrderDetails(ctx context.Context, bot BotInterface, chatID int64, callbackData string)
	HandleBackToOrder(ctx context.Context, bot BotInterface, chatID int64, callbackData string)
	HandleDeliveryConfirmation(ctx context.Context, bot BotInterface, chatID int64, callbackData string)
	HandleDeliveryCancel(ctx context.Context, bot BotInterface, chatID int64, callbackData string)
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
	ActionNavigate        = "nav"
	ActionCall            = "call"
	ActionStatus          = "status"
	ActionSettings        = "settings"
	ActionConfirm         = "confirm"
	ActionCancel          = "cancel"
	ActionRefresh         = "refresh"
	ActionMenu            = "menu"
	ActionConfirmDelivery = "confirm_delivery"
	ActionCancelDelivery  = "cancel_delivery"
	ActionChangeWorkmode  = "change_workmode"

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
	ID      int
	Status  string
	Address string
	Time    string
	Price   int
}

type BotConfig struct {
	Token   string
	Debug   bool
	Timeout int // in seconds
}
