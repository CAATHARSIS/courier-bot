package config

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost           string
	DBPort           string
	DBUser           string
	DBPassword       string
	DBName           string
	WebhookSecret    string
	TelegramBotToken string
	HTTPAddr         string
	Env              string
}

func Load() *Config {
	err := godotenv.Load("../../.env")
	if err != nil {
		slog.Warn("Warning: .env file not found")
	}

	return &Config{
		DBHost:           getEnv("DB_HOST", "localhost"),
		DBPort:           getEnv("DB_PORT", "5432"),
		DBUser:           getEnv("DB_USER", "postgres"),
		DBPassword:       getEnv("DB_PASSWORD", "postgres"),
		DBName:           getEnv("DB_NAME", "courier-bot"),
		WebhookSecret:    getEnv("WEBHOOK_SECRET", ""),
		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		HTTPAddr:         getEnv("HTTP_ADDR", ":8080"),
		Env:              getEnv("ENV", "local"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultValue
}
