package config

import (
	"errors"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramBotToken    string
	DBPath              string
	BackupDirectory     string
	BackupIntervalHours int
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	config := &Config{
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		DBPath:           os.Getenv("DB_PATH"),
		BackupDirectory:  os.Getenv("BACKUP_DIRECTORY"),
	}

	if config.TelegramBotToken == "" {
		return nil, errors.New("TELEGRAM_BOT_TOKEN not found in environment variables")
	}

	if config.DBPath == "" {
		return nil, errors.New("DB_PATH not found in environment variables")
	}

	if config.BackupDirectory == "" {
		return nil, errors.New("BACKUP_DIRECTORY not found in environment variables")
	}

	backupIntervalStr := os.Getenv("BACKUP_INTERVAL_HOURS")
	if backupIntervalStr == "" {
		return nil, errors.New("BACKUP_INTERVAL_HOURS not found in environment variables")
	}

	backupInterval, err := strconv.Atoi(backupIntervalStr)
	if err != nil {
		return nil, errors.New("BACKUP_INTERVAL_HOURS must be a valid integer")
	}
	config.BackupIntervalHours = backupInterval

	return config, nil
}
