package scheduler_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/artem-streltsov/ucl-timetable-bot/scheduler"
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

func TestGetNextNotificationTime(t *testing.T) {
	testCases := []struct {
		name       string
		now        time.Time
		dailyTime  string
		weeklyTime string
		expected   time.Time
	}{
		{
			name:      "Daily notification",
			now:       time.Date(2023, 5, 15, 12, 0, 0, 0, time.UTC),
			dailyTime: "18:00",
			expected:  time.Date(2023, 5, 15, 18, 0, 0, 0, time.UTC),
		},
		{
			name:       "Weekly notification",
			now:        time.Date(2023, 5, 15, 12, 0, 0, 0, time.UTC),
			weeklyTime: "SUN 10:00",
			expected:   time.Date(2023, 5, 21, 10, 0, 0, 0, time.UTC),
		},
		{
			name:      "Next day daily notification",
			now:       time.Date(2023, 5, 15, 19, 0, 0, 0, time.UTC),
			dailyTime: "18:00",
			expected:  time.Date(2023, 5, 16, 18, 0, 0, 0, time.UTC),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := scheduler.GetNextNotificationTime(tc.now, tc.dailyTime, tc.weeklyTime)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestScheduleDailySummary(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE users (chatID INTEGER PRIMARY KEY, webcalURL TEXT, lastDailySent DATETIME, lastWeeklySent DATETIME, dailyNotificationTime TEXT, weeklyNotificationTime TEXT, reminderOffset INTEGER)`)
	assert.NoError(t, err)

	_, err = db.Exec(`INSERT INTO users (chatID, webcalURL, dailyNotificationTime) VALUES (?, ?, ?)`, 123456, "https://example.com/calendar", "18:00")
	assert.NoError(t, err)

	mockBot := new(MockBotAPI)
	mockBot.On("NewMessage", int64(123456), mock.AnythingOfType("string")).Return(tgbotapi.MessageConfig{})
	mockBot.On("Send", mock.AnythingOfType("tgbotapi.MessageConfig")).Return(tgbotapi.Message{}, nil)

	err = scheduler.ScheduleDailySummary(mockBot, db, 123456, "https://example.com/calendar")
	assert.NoError(t, err)
}

func TestScheduleWeeklySummary(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE users (chatID INTEGER PRIMARY KEY, webcalURL TEXT, lastDailySent DATETIME, lastWeeklySent DATETIME, dailyNotificationTime TEXT, weeklyNotificationTime TEXT, reminderOffset INTEGER)`)
	assert.NoError(t, err)

	_, err = db.Exec(`INSERT INTO users (chatID, webcalURL, weeklyNotificationTime) VALUES (?, ?, ?)`, 123456, "https://example.com/calendar", "SUN 18:00")
	assert.NoError(t, err)

	mockBot := new(MockBotAPI)
	mockBot.On("NewMessage", int64(123456), mock.AnythingOfType("string")).Return(tgbotapi.MessageConfig{})
	mockBot.On("Send", mock.AnythingOfType("tgbotapi.MessageConfig")).Return(tgbotapi.Message{}, nil)

	err = scheduler.ScheduleWeeklySummary(mockBot, db, 123456, "https://example.com/calendar")
	assert.NoError(t, err)
}
