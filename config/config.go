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
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	config := &Config{
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		DBPath:           os.Getenv("DB_PATH"),
	}

	if config.TelegramBotToken == "" {
		return nil, errors.New("TELEGRAM_BOT_TOKEN not found in environment variables")
	}

	if config.DBPath == "" {
		return nil, errors.New("DB_PATH not found in environment variables")
	}

	return config, nil
}
