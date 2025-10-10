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
	TelegramBotToken string
	Env              string
}

func Load(log slog.Logger) *Config {
	err := godotenv.Load()
	if err != nil {
		log.Warn("Warning: .env file not found")
	}

	return &Config{
		DBHost:           getEnv("DB_HOST", "localhost"),
		DBPort:           getEnv("DB_PORT", "5432"),
		DBUser:           getEnv("DB_USER", "postgres"),
		DBPassword:       getEnv("DB_PASSWORD", "postgres"),
		DBName:           getEnv("DB_NAME", "courier-bot"),
		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		Env:              getEnv("ENV", "local"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultValue
}
