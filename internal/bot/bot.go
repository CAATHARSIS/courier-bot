package bot

import (
	"context"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramBot struct {
	api      *tgbotapi.BotAPI
	handlers *Handlers
	log      *slog.Logger
}

func NewTelegramBot(api *tgbotapi.BotAPI, handlers *Handlers, log *slog.Logger) *TelegramBot {
	return &TelegramBot{
		api:      api,
		handlers: handlers,
		log:      log,
	}
}

func (b *TelegramBot) Start(ctx context.Context) {
	b.log.Info("Starting Telegram bot")

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			b.log.Info("Stopping Telegram bot")
			return
		case update := <-updates:
			go b.handleUpdate(ctx, update)
		}
	}
}

func (b *TelegramBot) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	if update.Message != nil {
		b.handlers.HandleMessage(ctx, b, update)
	} else if update.CallbackQuery != nil {
		b.handlers.HandleCallback(ctx, b, update)
	}
}

func (b *TelegramBot) SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	_, err := b.api.Send(msg)
	return err
}

func (b *TelegramBot) SendMessageWithKeyboard(chatID int64, text string, keyboard tgbotapi.ReplyKeyboardMarkup) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	_, err := b.api.Send(msg)
	return err
}

func (b *TelegramBot) SendMessageWithInlineKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	_, err := b.api.Send(msg)
	return err
}

func (b *TelegramBot) EditMessageText(chatID int64, messageID int64, text string) error {
	editMsg := tgbotapi.NewEditMessageText(chatID, int(messageID), text)
	editMsg.ParseMode = "Markdown"
	_, err := b.api.Send(editMsg)
	return err
}

func (b *TelegramBot) EditMessageReplyMarkup(chatID int64, messageID int64, replyMarkup interface{}) error {
	editMsg := tgbotapi.NewEditMessageReplyMarkup(chatID, int(messageID), replyMarkup.(tgbotapi.InlineKeyboardMarkup))
	_, err := b.api.Send(editMsg)
	return err
}

func (b *TelegramBot) AnswerCallbackQuery(callbackQueryID string) error {
	callback := tgbotapi.NewCallback(callbackQueryID, "")
	_, err := b.api.Request(callback)
	return err
}

func (b *TelegramBot) AnswerCallbackQueryWithText(callbackQueryID, text string) error {
	callback := tgbotapi.NewCallback(callbackQueryID, text)
	_, err := b.api.Request(callback)
	return err
}

func (b *TelegramBot) GetMe() (*tgbotapi.User, error) {
	user, err := b.api.GetMe()
	return &user, err
}

func (b *TelegramBot) TestConnection() error {
	_, err := b.api.GetMe()
	return err
}

func (b *TelegramBot) SetDefaultCommands() error {
	commands := []tgbotapi.BotCommand{
		{Command: "start", Description: "Запустить бота"},
		{Command: "help", Description: "🆘 Помощь"},
		{Command: "orders", Description: "📋 Мои заказы"},
		{Command: "status", Description: "ℹ️ Статус"},
		{Command: "settings", Description: "⚙️ Настройки"},
	}

	config := tgbotapi.NewSetMyCommands(commands...)
	_, err := b.api.Request(config)
	return err
}
