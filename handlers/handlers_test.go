package handlers_test

import (
	"database/sql"
	"testing"

	"github.com/artem-streltsov/ucl-timetable-bot/database"
	"github.com/artem-streltsov/ucl-timetable-bot/handlers"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockBotAPI struct {
	mock.Mock
}

func (m *MockBotAPI) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	args := m.Called(c)
	return args.Get(0).(tgbotapi.Message), args.Error(1)
}

func (m *MockBotAPI) GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	args := m.Called(config)
	return args.Get(0).(tgbotapi.UpdatesChannel)
}

func (m *MockBotAPI) NewMessage(chatID int64, text string) tgbotapi.MessageConfig {
	args := m.Called(chatID, text)
	return args.Get(0).(tgbotapi.MessageConfig)
}

func TestValidateWebCalLink(t *testing.T) {
	cases := []struct {
		input    string
		expected string
		valid    bool
	}{
		{"webcal://example.com", "https://example.com", true},
		{"WEBCAL://example.com", "https://example.com", true},
		{"http://example.com", "", false},
		{"example.com", "", false},
	}

	for _, c := range cases {
		result, valid := handlers.ValidateWebCalLink(c.input)
		assert.Equal(t, c.expected, result)
		assert.Equal(t, c.valid, valid)
	}
}

func TestHandleWebCalLink(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE users (chatID INTEGER PRIMARY KEY, webcalURL TEXT, lastDailySent DATETIME, lastWeeklySent DATETIME)`)
	assert.NoError(t, err)

	mockBot := new(MockBotAPI)
	mockBot.On("NewMessage", int64(123), mock.AnythingOfType("string")).Return(tgbotapi.MessageConfig{})
	mockBot.On("NewMessage", int64(456), mock.AnythingOfType("string")).Return(tgbotapi.MessageConfig{})
	mockBot.On("Send", mock.AnythingOfType("tgbotapi.MessageConfig")).Return(tgbotapi.Message{}, nil)

	handlers.HandleWebCalLink(mockBot, db, 123, "webcal://example.com")

	var savedURL string
	err = db.QueryRow("SELECT webcalURL FROM users WHERE chatID = ?", 123).Scan(&savedURL)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", savedURL)

	handlers.HandleWebCalLink(mockBot, db, 456, "http://example.com")

	err = db.QueryRow("SELECT webcalURL FROM users WHERE chatID = ?", 456).Scan(&savedURL)
	assert.Error(t, err)

	mockBot.AssertExpectations(t)
}

func TestHandleTodayCommand(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE users (chatID INTEGER PRIMARY KEY, webcalURL TEXT, lastDailySent DATETIME, lastWeeklySent DATETIME)`)
	assert.NoError(t, err)

	err = database.InsertUser(db, 123, "https://example.com")
	assert.NoError(t, err)

	mockBot := new(MockBotAPI)
	mockBot.On("NewMessage", int64(123), mock.AnythingOfType("string")).Return(tgbotapi.MessageConfig{})
	mockBot.On("Send", mock.AnythingOfType("tgbotapi.MessageConfig")).Return(tgbotapi.Message{}, nil)

	handlers.HandleTodayCommand(mockBot, db, 123)

	mockBot.AssertExpectations(t)
}

func TestHandleWeekCommand(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE users (chatID INTEGER PRIMARY KEY, webcalURL TEXT, lastDailySent DATETIME, lastWeeklySent DATETIME)`)
	assert.NoError(t, err)

	err = database.InsertUser(db, 123, "https://example.com")
	assert.NoError(t, err)

	mockBot := new(MockBotAPI)
	mockBot.On("NewMessage", int64(123), mock.AnythingOfType("string")).Return(tgbotapi.MessageConfig{})
	mockBot.On("Send", mock.AnythingOfType("tgbotapi.MessageConfig")).Return(tgbotapi.Message{}, nil)

	handlers.HandleWeekCommand(mockBot, db, 123)

	mockBot.AssertExpectations(t)
}
