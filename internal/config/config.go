package config

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost           string
	DBPort           string
	DBUserName       string
	DBPassword       string
	DBName           string
	AdminPassword    string
	TelegramBotToken string
	BotPort          string
}

func Load() *Config {
	err := godotenv.Load("../../.env")
	if err != nil {
		slog.Warn("Warning: .env file not found")
	}
	
	// local
	// dbHost := "212.41.6.229"

	// deploy
	dbHost := "db"

	return &Config{
		DBHost:           dbHost,
		DBPort:           getEnv("DB_PORT", "5432"),
		DBUserName:       getEnv("DB_USERNAME", "postgres"),
		DBPassword:       getEnv("DB_PASSWORD", "postgres"),
		DBName:           getEnv("DB_NAME", "courier-bot"),
		AdminPassword:    getEnv("ADMIN_PASSWORD", ""),
		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		BotPort:          getEnv("BOT_PORT", ":8080"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultValue
}
