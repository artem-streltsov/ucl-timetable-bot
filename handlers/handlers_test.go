package handlers_test

import (
	"database/sql"
	"testing"

	"github.com/artem-streltsov/ucl-timetable-bot/common"
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

func SetupDatabase(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)

	createTableSQL := `
    CREATE TABLE users (
        chatID INTEGER PRIMARY KEY,
        webcalURL TEXT,
        lastDailySent DATETIME,
        lastWeeklySent DATETIME,
        dailyNotificationTime TEXT DEFAULT '18:00',
        weeklyNotificationTime TEXT DEFAULT 'SUN 18:00',
        reminderOffset INTEGER DEFAULT 30
    );
    `
	_, err = db.Exec(createTableSQL)
	assert.NoError(t, err)

	return db
}

func SetupMockBot() *MockBotAPI {
	mockBot := new(MockBotAPI)
	mockBot.On("NewMessage", mock.AnythingOfType("int64"), mock.AnythingOfType("string")).Return(tgbotapi.MessageConfig{})
	mockBot.On("Send", mock.AnythingOfType("tgbotapi.MessageConfig")).Return(tgbotapi.Message{}, nil)
	return mockBot
}

func AssertDatabaseValue(t *testing.T, db *sql.DB, chatID int64, column, expectedValue string) {
	var result string
	err := db.QueryRow("SELECT "+column+" FROM users WHERE chatID = ?", chatID).Scan(&result)
	assert.NoError(t, err)
	assert.Equal(t, expectedValue, result)
}

func TestHandleStartCommand(t *testing.T) {
	mockBot := SetupMockBot()
	mockBot.On("NewMessage", int64(123), "Please provide your WebCal link to subscribe to your lecture timetable.").Return(tgbotapi.MessageConfig{})
	mockBot.On("Send", mock.AnythingOfType("tgbotapi.MessageConfig")).Return(tgbotapi.Message{}, nil)

	handlers.HandleStartCommand(mockBot, 123)

	mockBot.AssertExpectations(t)
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
	db := SetupDatabase(t)
	defer db.Close()

	mockBot := SetupMockBot()
	mockBot.On("NewMessage", int64(123), mock.AnythingOfType("string")).Return(tgbotapi.MessageConfig{})
	mockBot.On("NewMessage", int64(456), mock.AnythingOfType("string")).Return(tgbotapi.MessageConfig{})
	mockBot.On("Send", mock.AnythingOfType("tgbotapi.MessageConfig")).Return(tgbotapi.Message{}, nil)

	handlers.HandleWebCalLink(mockBot, db, 123, "webcal://example.com")

	var savedURL string
	err := db.QueryRow("SELECT webcalURL FROM users WHERE chatID = ?", 123).Scan(&savedURL)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", savedURL)

	handlers.HandleWebCalLink(mockBot, db, 456, "http://example.com")

	err = db.QueryRow("SELECT webcalURL FROM users WHERE chatID = ?", 456).Scan(&savedURL)
	assert.Error(t, err)

	mockBot.AssertExpectations(t)
}

func TestHandleTodayCommand(t *testing.T) {
	db := SetupDatabase(t)
	defer db.Close()

	err := database.InsertUser(db, 123, "https://example.com")
	assert.NoError(t, err)

	mockBot := SetupMockBot()
	mockBot.On("NewMessage", int64(123), mock.AnythingOfType("string")).Return(tgbotapi.MessageConfig{})
	mockBot.On("Send", mock.AnythingOfType("tgbotapi.MessageConfig")).Return(tgbotapi.Message{}, nil)

	handlers.HandleTodayCommand(mockBot, db, 123)

	mockBot.AssertExpectations(t)
}

func TestHandleWeekCommand(t *testing.T) {
	db := SetupDatabase(t)
	defer db.Close()

	err := database.InsertUser(db, 123, "https://example.com")
	assert.NoError(t, err)

	mockBot := SetupMockBot()
	mockBot.On("NewMessage", int64(123), mock.AnythingOfType("string")).Return(tgbotapi.MessageConfig{})
	mockBot.On("Send", mock.AnythingOfType("tgbotapi.MessageConfig")).Return(tgbotapi.Message{}, nil)

	handlers.HandleWeekCommand(mockBot, db, 123)

	mockBot.AssertExpectations(t)
}

func TestHandleSetDailyTime(t *testing.T) {
	db := SetupDatabase(t)
	defer db.Close()

	mockBot := SetupMockBot()
	mockBot.On("NewMessage", int64(123), mock.AnythingOfType("string")).Return(tgbotapi.MessageConfig{})
	mockBot.On("Send", mock.AnythingOfType("tgbotapi.MessageConfig")).Return(tgbotapi.Message{}, nil)

	_, err := db.Exec(`INSERT INTO users (chatID, webcalURL, lastDailySent, lastWeeklySent) VALUES (?, ?, NULL, NULL)`, 123, "http://example.com")
	assert.NoError(t, err)

	handlers.HandleSetDailyTime(mockBot, db, 123, "08:00")
	AssertDatabaseValue(t, db, 123, "dailyNotificationTime", "08:00")
	mockBot.AssertExpectations(t)
}

func TestHandleSetWeeklyTime(t *testing.T) {
	db := SetupDatabase(t)
	defer db.Close()

	mockBot := SetupMockBot()
	mockBot.On("NewMessage", int64(123), mock.AnythingOfType("string")).Return(tgbotapi.MessageConfig{})
	mockBot.On("Send", mock.AnythingOfType("tgbotapi.MessageConfig")).Return(tgbotapi.Message{}, nil)

	_, err := db.Exec(`INSERT INTO users (chatID, webcalURL, lastDailySent, lastWeeklySent) VALUES (?, ?, NULL, NULL)`, 123, "http://example.com")
	assert.NoError(t, err)

	handlers.HandleSetWeeklyTime(mockBot, db, 123, "SUN 18:00")
	AssertDatabaseValue(t, db, 123, "weeklyNotificationTime", "SUN 18:00")
	mockBot.AssertExpectations(t)
}

func TestHandleSetReminderOffset(t *testing.T) {
	db := SetupDatabase(t)
	defer db.Close()

	mockBot := SetupMockBot()
	mockBot.On("NewMessage", int64(123), mock.AnythingOfType("string")).Return(common.MessageConfig{})
	mockBot.On("Send", mock.AnythingOfType("tgbotapi.MessageConfig")).Return(common.Message{}, nil)

	_, err := db.Exec(`INSERT INTO users (chatID, webcalURL, lastDailySent, lastWeeklySent) VALUES (?, ?, NULL, NULL)`, 123, "http://example.com")
	assert.NoError(t, err)

	handlers.HandleSetReminderOffset(mockBot, db, 123, "30")
	AssertDatabaseValue(t, db, 123, "reminderOffset", "30")
	mockBot.AssertExpectations(t)
}
