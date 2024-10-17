package config

import (
	"errors"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramBotToken string
	DBPath           string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	dbPath := os.Getenv("DB_PATH")

	if token == "" {
		return nil, errors.New("TELEGRAM_BOT_TOKEN not set")
	}

	if dbPath == "" {
		return nil, errors.New("DB_PATH not set")
	}

	return &Config{
		TelegramBotToken: token,
		DBPath:           dbPath,
	}, nil
}
