package scheduler_test

import (
	"database/sql"
	"testing"
	"time"

	ical "github.com/arran4/golang-ical"
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

	_, err = db.Exec(`CREATE TABLE users (
		chatID INTEGER PRIMARY KEY,
		webcalURL TEXT,
		lastDailySent DATETIME,
		lastWeeklySent DATETIME,
		dailyNotificationTime TEXT DEFAULT '18:00',
		weeklyNotificationTime TEXT DEFAULT 'SUN 18:00',
		reminderOffset INTEGER DEFAULT 30
	)`)
	assert.NoError(t, err)

	_, err = db.Exec(`INSERT INTO users (chatID, webcalURL, dailyNotificationTime, weeklyNotificationTime, reminderOffset)
		VALUES (?, ?, ?, ?, ?)`, 123456, "https://example.com/calendar", "18:00", "SUN 18:00", 30)
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

	_, err = db.Exec(`CREATE TABLE users (
		chatID INTEGER PRIMARY KEY,
		webcalURL TEXT,
		lastDailySent DATETIME,
		lastWeeklySent DATETIME,
		dailyNotificationTime TEXT DEFAULT '18:00',
		weeklyNotificationTime TEXT DEFAULT 'SUN 18:00',
		reminderOffset INTEGER DEFAULT 30
	)`)
	assert.NoError(t, err)

	_, err = db.Exec(`INSERT INTO users (chatID, webcalURL, dailyNotificationTime, weeklyNotificationTime, reminderOffset)
		VALUES (?, ?, ?, ?, ?)`, 123456, "https://example.com/calendar", "18:00", "SUN 18:00", 30)
	assert.NoError(t, err)

	mockBot := new(MockBotAPI)
	mockBot.On("NewMessage", int64(123456), mock.AnythingOfType("string")).Return(tgbotapi.MessageConfig{})
	mockBot.On("Send", mock.AnythingOfType("tgbotapi.MessageConfig")).Return(tgbotapi.Message{}, nil)

	err = scheduler.ScheduleWeeklySummary(mockBot, db, 123456, "https://example.com/calendar")
	assert.NoError(t, err)
}

func TestRescheduleNotificationsOnStartup(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE users (
		chatID INTEGER PRIMARY KEY,
		webcalURL TEXT,
		lastDailySent DATETIME,
		lastWeeklySent DATETIME,
		dailyNotificationTime TEXT DEFAULT '18:00',
		weeklyNotificationTime TEXT DEFAULT 'SUN 18:00',
		reminderOffset INTEGER DEFAULT 30
	)`)
	assert.NoError(t, err)

	_, err = db.Exec(`INSERT INTO users (chatID, webcalURL, lastDailySent, lastWeeklySent)
		VALUES (?, ?, ?, ?)`, 123456, "https://example.com/calendar", "2023-05-01T12:00:00Z", "2023-05-01T12:00:00Z")
	assert.NoError(t, err)

	mockBot := new(MockBotAPI)
	mockBot.On("NewMessage", int64(123456), mock.AnythingOfType("string")).Return(tgbotapi.MessageConfig{})
	mockBot.On("Send", mock.AnythingOfType("tgbotapi.MessageConfig")).Return(tgbotapi.Message{}, nil)

	err = scheduler.RescheduleNotificationsOnStartup(mockBot, db)
	assert.NoError(t, err)
}

func TestScheduleLectureReminders(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE users (
		chatID INTEGER PRIMARY KEY,
		webcalURL TEXT,
		lastDailySent DATETIME,
		lastWeeklySent DATETIME,
		dailyNotificationTime TEXT DEFAULT '18:00',
		weeklyNotificationTime TEXT DEFAULT 'SUN 18:00',
		reminderOffset INTEGER DEFAULT 30
	)`)
	assert.NoError(t, err)

	_, err = db.Exec(`INSERT INTO users (chatID, webcalURL, dailyNotificationTime, weeklyNotificationTime, reminderOffset)
		VALUES (?, ?, ?, ?, ?)`, 123456, "https://example.com/calendar", "18:00", "SUN 18:00", 30)
	assert.NoError(t, err)

	mockBot := new(MockBotAPI)
	mockBot.On("NewMessage", int64(123456), mock.AnythingOfType("string")).Return(tgbotapi.MessageConfig{})
	mockBot.On("Send", mock.AnythingOfType("tgbotapi.MessageConfig")).Return(tgbotapi.Message{}, nil)

	lectures := []*ical.VEvent{
		func() *ical.VEvent {
			event := ical.NewEvent("lecture-1")
			event.SetProperty(ical.ComponentPropertySummary, "Test Lecture 1")
			event.SetProperty(ical.ComponentPropertyLocation, "Room 101")
			event.SetProperty(ical.ComponentPropertyDtStart, "20230515T090000Z")
			event.SetProperty(ical.ComponentPropertyDtEnd, "20230515T100000Z")
			return event
		}(),
		func() *ical.VEvent {
			event := ical.NewEvent("lecture-2")
			event.SetProperty(ical.ComponentPropertySummary, "Test Lecture 2")
			event.SetProperty(ical.ComponentPropertyLocation, "Room 102")
			event.SetProperty(ical.ComponentPropertyDtStart, "20230516T130000Z")
			event.SetProperty(ical.ComponentPropertyDtEnd, "20230516T140000Z")
			return event
		}(),
	}

	scheduler.ScheduleLectureReminders(mockBot, db, 123456, lectures)
	assert.NoError(t, err)
}
