package scheduler_test

import (
	"database/sql"
	"testing"
	"time"

	ical "github.com/arran4/golang-ical"
	"github.com/artem-streltsov/ucl-timetable-bot/common"
	"github.com/artem-streltsov/ucl-timetable-bot/scheduler"
	"github.com/artem-streltsov/ucl-timetable-bot/utils"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockBotAPI struct {
	mock.Mock
	common.BotAPI
}

func (m *MockBotAPI) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	args := m.Called(c)
	return args.Get(0).(tgbotapi.Message), args.Error(1)
}

func (m *MockBotAPI) NewMessage(chatID int64, text string) tgbotapi.MessageConfig {
	args := m.Called(chatID, text)
	return args.Get(0).(tgbotapi.MessageConfig)
}

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)

	_, err = db.Exec(`CREATE TABLE users (
		chatID INTEGER PRIMARY KEY,
		webcalURL TEXT,
		lastDailySent INTEGER,
		lastWeeklySent INTEGER,
		dailyNotificationTime TEXT DEFAULT '18:00',
		weeklyNotificationTime TEXT DEFAULT 'SUN 18:00',
		reminderOffset INTEGER DEFAULT 30
	)`)
	assert.NoError(t, err)

	return db
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
			now:       time.Date(2023, 5, 15, 12, 0, 0, 0, utils.GetUKLocation()),
			dailyTime: "18:00",
			expected:  time.Date(2023, 5, 15, 18, 0, 0, 0, utils.GetUKLocation()),
		},
		{
			name:       "Weekly notification",
			now:        time.Date(2023, 5, 15, 12, 0, 0, 0, utils.GetUKLocation()),
			weeklyTime: "SUN 10:00",
			expected:   time.Date(2023, 5, 21, 10, 0, 0, 0, utils.GetUKLocation()),
		},
		{
			name:      "Next day daily notification",
			now:       time.Date(2023, 5, 15, 19, 0, 0, 0, utils.GetUKLocation()),
			dailyTime: "18:00",
			expected:  time.Date(2023, 5, 16, 18, 0, 0, 0, utils.GetUKLocation()),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := scheduler.GetNextNotificationTime(tc.now, tc.dailyTime, tc.weeklyTime)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestScheduleDailySummary(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`INSERT INTO users (chatID, webcalURL, dailyNotificationTime, weeklyNotificationTime, reminderOffset)
		VALUES (?, ?, ?, ?, ?)`, 123456, "https://example.com/calendar", "18:00", "SUN 18:00", 30)
	assert.NoError(t, err)

	mockBot := new(MockBotAPI)

	err = scheduler.ScheduleDailySummary(mockBot, db, 123456)
	assert.NoError(t, err)
}

func TestScheduleWeeklySummary(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`INSERT INTO users (chatID, webcalURL, dailyNotificationTime, weeklyNotificationTime, reminderOffset)
		VALUES (?, ?, ?, ?, ?)`, 123456, "https://example.com/calendar", "18:00", "SUN 18:00", 30)
	assert.NoError(t, err)

	mockBot := new(MockBotAPI)

	err = scheduler.ScheduleWeeklySummary(mockBot, db, 123456)
	assert.NoError(t, err)
}

func TestRescheduleNotificationsOnStartup(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	lastSentTime := time.Date(2023, 5, 1, 12, 0, 0, 0, utils.GetUKLocation())
	lastSentEpoch := utils.TimeToEpoch(lastSentTime)

	_, err := db.Exec(`INSERT INTO users (chatID, webcalURL, lastDailySent, lastWeeklySent)
		VALUES (?, ?, ?, ?)`, 123456, "https://example.com/calendar", lastSentEpoch, lastSentEpoch)
	assert.NoError(t, err)

	mockBot := new(MockBotAPI)

	err = scheduler.RescheduleNotificationsOnStartup(mockBot, db)
	assert.NoError(t, err)
}

func TestScheduleLectureReminders(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`INSERT INTO users (chatID, webcalURL, dailyNotificationTime, weeklyNotificationTime, reminderOffset)
		VALUES (?, ?, ?, ?, ?)`, 123456, "https://example.com/calendar", "18:00", "SUN 18:00", 30)
	assert.NoError(t, err)

	mockBot := new(MockBotAPI)

	lectures := []*ical.VEvent{
		func() *ical.VEvent {
			event := ical.NewEvent("lecture-1")
			event.SetProperty(ical.ComponentPropertySummary, "Test Lecture 1")
			event.SetProperty(ical.ComponentPropertyLocation, "Room 101")
			event.SetProperty(ical.ComponentPropertyDtStart, time.Now().Add(time.Hour).Format("20060102T150405Z"))
			event.SetProperty(ical.ComponentPropertyDtEnd, time.Now().Add(2*time.Hour).Format("20060102T150405Z"))
			return event
		}(),
	}

	scheduler.ScheduleLectureReminders(mockBot, db, 123456, lectures)
	assert.NoError(t, err)
}

func TestStopAndRescheduleNotifications(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`INSERT INTO users (chatID, webcalURL, dailyNotificationTime, weeklyNotificationTime, reminderOffset)
		VALUES (?, ?, ?, ?, ?)`, 123456, "https://example.com/calendar", "18:00", "SUN 18:00", 30)
	assert.NoError(t, err)

	mockBot := new(MockBotAPI)

	err = scheduler.StopAndRescheduleNotifications(mockBot, db, 123456)
	assert.NoError(t, err)
}
