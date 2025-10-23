package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CAATHARSIS/courier-bot/internal/bot"
	"github.com/CAATHARSIS/courier-bot/internal/config"
	delivery "github.com/CAATHARSIS/courier-bot/internal/delivery/http"
	"github.com/CAATHARSIS/courier-bot/internal/logger"
	"github.com/CAATHARSIS/courier-bot/internal/repository"
	"github.com/CAATHARSIS/courier-bot/internal/service/assignment"
	"github.com/CAATHARSIS/courier-bot/internal/service/notification"
	"github.com/CAATHARSIS/courier-bot/pkg/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	_ "github.com/lib/pq"
)

func main() {
	cfg := config.Load()

	log := logger.NewLogger(cfg.Env)
	if cfg.Env != "prod" {
		log.Info("Debug messages are enable")
	}

	migrationDB, err := database.NewPostgresDB(cfg)
	if err != nil {
		log.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	if err := database.RunMigrations(migrationDB, log); err != nil {
		log.Error("Failed to run migrations", "error", err)
		if err := migrationDB.Close(); err != nil {
			log.Error("Failed to close migration db", "error", err)
		}
		os.Exit(1)
	}

	appDB, err := database.NewPostgresDB(cfg)
	if err != nil {
		log.Error("Failed to connect to database", "error", err)
		if err := appDB.Close(); err != nil {
			log.Error("Failed to close app db", "error", err)
		}
		os.Exit(1)
	}

	repo := repository.NewRepository(appDB)

	telegramBot, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		log.Error("Failed to create Telegram bot", "error", err)
		os.Exit(1)
	}

	telegramBot.Debug = cfg.Env == "dev"
	log.Info("Authorized on account", "username", telegramBot.Self.UserName)

	notificationService := notification.NewService(telegramBot, log)

	assignmentService := assignment.NewService(*repo, notificationService, log)

	assignmentManager := assignment.NewAssignmentManager(assignmentService, log)
	assignmentManager.StartCleanupWorker()

	webhookHandler := delivery.NewWebhookHandler(assignmentService, cfg.WebhookSecret, log)

	keyboardManager := bot.NewkeyboardManager(log)
	handlers := bot.NewHandlers(assignmentService, keyboardManager, log)

	botInstance := bot.NewTelegramBot(telegramBot, handlers, log)

	go botInstance.Start(context.Background())

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/order", func(w http.ResponseWriter, r *http.Request) {
		webhookHandler.HandleNewOrderWebhook(context.Background(), w, r)
	})
	mux.HandleFunc("/heath", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("Starting HTTP server", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Failed to start HTTP server", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", "error", err)
	}

	log.Info("Server exited")
}
