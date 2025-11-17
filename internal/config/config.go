package config

import (
	"flag"
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
	Local            bool
}

func Load() *Config {
	local := flag.Bool("local", false, "Using for local development")
	flag.Parse()

	var dbHost string

	if *local {
		if err := godotenv.Load("../../.env"); err != nil {
			slog.Warn("Warning: .env file not found")
		}

		dbHost = "212.41.6.229"
	} else {
		dbHost = "db"
	}

	return &Config{
		DBHost:           dbHost,
		DBPort:           getEnv("DB_PORT", "5432"),
		DBUserName:       getEnv("DB_USERNAME", "postgres"),
		DBPassword:       getEnv("DB_PASSWORD", "postgres"),
		DBName:           getEnv("DB_NAME", "courier-bot"),
		AdminPassword:    getEnv("ADMIN_PASSWORD", ""),
		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		BotPort:          getEnv("BOT_PORT", "8080"),
		Local:            *local,
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultValue
}
