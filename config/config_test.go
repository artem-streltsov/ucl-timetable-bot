package config_test

import (
	"os"
	"testing"

	"github.com/artem-streltsov/ucl-timetable-bot/config"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfigSuccess(t *testing.T) {
	os.Setenv("TELEGRAM_BOT_TOKEN", "mock_token")
	os.Setenv("DB_PATH", "mock_db_path")
	os.Setenv("BACKUP_DIRECTORY", "mock_backup_directory")
	os.Setenv("BACKUP_INTERVAL_HOURS", "24")

	defer os.Unsetenv("TELEGRAM_BOT_TOKEN")
	defer os.Unsetenv("DB_PATH")
	defer os.Unsetenv("BACKUP_DIRECTORY")
	defer os.Unsetenv("BACKUP_INTERVAL_HOURS")

	cfg, err := config.Load()

	assert.NoError(t, err)

	assert.Equal(t, "mock_token", cfg.TelegramBotToken)
	assert.Equal(t, "mock_db_path", cfg.DBPath)
	assert.Equal(t, "mock_backup_directory", cfg.BackupDirectory)
	assert.Equal(t, 24, cfg.BackupIntervalHours)
}

func TestLoadConfigMissingTelegramBotToken(t *testing.T) {
	os.Setenv("DB_PATH", "mock_db_path")
	os.Setenv("BACKUP_DIRECTORY", "mock_backup_directory")
	os.Setenv("BACKUP_INTERVAL_HOURS", "24")

	defer os.Unsetenv("DB_PATH")
	defer os.Unsetenv("BACKUP_DIRECTORY")
	defer os.Unsetenv("BACKUP_INTERVAL_HOURS")

	cfg, err := config.Load()

	assert.Nil(t, cfg)
	assert.EqualError(t, err, "TELEGRAM_BOT_TOKEN not found in environment variables")
}

func TestLoadConfigMissingDBPath(t *testing.T) {
	os.Setenv("TELEGRAM_BOT_TOKEN", "mock_token")
	os.Setenv("BACKUP_DIRECTORY", "mock_backup_directory")
	os.Setenv("BACKUP_INTERVAL_HOURS", "24")

	defer os.Unsetenv("TELEGRAM_BOT_TOKEN")
	defer os.Unsetenv("BACKUP_DIRECTORY")
	defer os.Unsetenv("BACKUP_INTERVAL_HOURS")

	cfg, err := config.Load()

	assert.Nil(t, cfg)
	assert.EqualError(t, err, "DB_PATH not found in environment variables")
}

func TestLoadConfigNoEnv(t *testing.T) {
	os.Setenv("TELEGRAM_BOT_TOKEN", "mock_token")
	os.Setenv("DB_PATH", "mock_db_path")
	os.Setenv("BACKUP_DIRECTORY", "mock_backup_directory")
	os.Setenv("BACKUP_INTERVAL_HOURS", "24")

	defer os.Unsetenv("TELEGRAM_BOT_TOKEN")
	defer os.Unsetenv("DB_PATH")
	defer os.Unsetenv("BACKUP_DIRECTORY")
	defer os.Unsetenv("BACKUP_INTERVAL_HOURS")

	cfg, err := config.Load()

	assert.NoError(t, err)

	assert.Equal(t, "mock_token", cfg.TelegramBotToken)
	assert.Equal(t, "mock_db_path", cfg.DBPath)
	assert.Equal(t, "mock_backup_directory", cfg.BackupDirectory)
	assert.Equal(t, 24, cfg.BackupIntervalHours)
}
