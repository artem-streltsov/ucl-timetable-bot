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

	defer os.Unsetenv("TELEGRAM_BOT_TOKEN")
	defer os.Unsetenv("DB_PATH")

	cfg, err := config.Load()

	assert.NoError(t, err)

	assert.Equal(t, "mock_token", cfg.TelegramBotToken)
	assert.Equal(t, "mock_db_path", cfg.DBPath)
}

func TestLoadConfigMissingTelegramBotToken(t *testing.T) {
	os.Setenv("DB_PATH", "mock_db_path")

	defer os.Unsetenv("DB_PATH")

	cfg, err := config.Load()

	assert.Nil(t, cfg)
	assert.EqualError(t, err, "TELEGRAM_BOT_TOKEN not found in environment variables")
}

func TestLoadConfigMissingDBPath(t *testing.T) {
	os.Setenv("TELEGRAM_BOT_TOKEN", "mock_token")

	defer os.Unsetenv("TELEGRAM_BOT_TOKEN")

	cfg, err := config.Load()

	assert.Nil(t, cfg)
	assert.EqualError(t, err, "DB_PATH not found in environment variables")
}

func TestLoadConfigNoEnv(t *testing.T) {
	os.Setenv("TELEGRAM_BOT_TOKEN", "mock_token")
	os.Setenv("DB_PATH", "mock_db_path")

	defer os.Unsetenv("TELEGRAM_BOT_TOKEN")
	defer os.Unsetenv("DB_PATH")

	cfg, err := config.Load()

	assert.NoError(t, err)

	assert.Equal(t, "mock_token", cfg.TelegramBotToken)
	assert.Equal(t, "mock_db_path", cfg.DBPath)
}
