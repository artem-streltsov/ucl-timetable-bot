package notifications_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	ical "github.com/arran4/golang-ical"
	"github.com/artem-streltsov/ucl-timetable-bot/notifications"
	"github.com/artem-streltsov/ucl-timetable-bot/utils"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

func TestFormatEventDetails(t *testing.T) {
	tests := []struct {
		name     string
		event    *ical.VEvent
		expected string
	}{
		{
			name: "Winter time (GMT)",
			event: func() *ical.VEvent {
				event := ical.NewEvent("test-event")
				event.SetProperty(ical.ComponentPropertySummary, "Test Lecture")
				event.SetProperty(ical.ComponentPropertyLocation, "Room 101")
				event.SetProperty(ical.ComponentPropertyDtStart, "20230115T090000Z") // January 15, 2023 (winter)
				return event
			}(),
			expected: "- Test Lecture at Room 101, starting at 09:00",
		},
		{
			name: "Summer time (BST)",
			event: func() *ical.VEvent {
				event := ical.NewEvent("test-event")
				event.SetProperty(ical.ComponentPropertySummary, "Test Lecture")
				event.SetProperty(ical.ComponentPropertyLocation, "Room 101")
				event.SetProperty(ical.ComponentPropertyDtStart, "20230615T090000Z") // June 15, 2023 (summer)
				return event
			}(),
			expected: "- Test Lecture at Room 101, starting at 10:00",
		},
		{
			name: "Missing summary",
			event: func() *ical.VEvent {
				event := ical.NewEvent("test-event")
				event.SetProperty(ical.ComponentPropertyLocation, "Room 101")
				event.SetProperty(ical.ComponentPropertyDtStart, "20230515T090000Z")
				return event
			}(),
			expected: "- Unknown at Room 101, starting at 10:00",
		},
		{
			name: "Missing location",
			event: func() *ical.VEvent {
				event := ical.NewEvent("test-event")
				event.SetProperty(ical.ComponentPropertySummary, "Test Lecture")
				event.SetProperty(ical.ComponentPropertyDtStart, "20230515T090000Z")
				return event
			}(),
			expected: "- Test Lecture at Unknown, starting at 10:00",
		},
		{
			name: "Missing start time",
			event: func() *ical.VEvent {
				event := ical.NewEvent("test-event")
				event.SetProperty(ical.ComponentPropertySummary, "Test Lecture")
				event.SetProperty(ical.ComponentPropertyLocation, "Room 101")
				return event
			}(),
			expected: "- Test Lecture at Room 101, starting at Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := notifications.FormatEventDetails(tt.event)
			assert.Equal(t, tt.expected, formatted)
		})
	}
}

func createTestCalendar() *ical.Calendar {
	cal := ical.NewCalendar()
	cal.SetMethod(ical.MethodPublish)
	event := cal.AddEvent("uid1@example.com")
	event.SetCreatedTime(time.Now())
	event.SetDtStampTime(time.Now())
	event.SetModifiedAt(time.Now())
	event.SetStartAt(time.Now().In(utils.GetUKLocation()))
	event.SetEndAt(time.Now().In(utils.GetUKLocation()).Add(time.Hour))
	event.SetSummary("Test Event")
	event.SetLocation("Test Location")
	return cal
}

func createMockCalendarServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/calendar")
		cal := createTestCalendar()
		calString := cal.Serialize()
		w.Write([]byte(calString))
	}))
}

func TestSendDailySummary(t *testing.T) {
	mockBot := new(MockBotAPI)
	chatID := int64(123456)

	server := createMockCalendarServer()
	defer server.Close()

	db, sqlMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlMock.ExpectQuery("SELECT dailyNotificationTime, weeklyNotificationTime, reminderOffset FROM users WHERE chatID = ?").
		WithArgs(chatID).
		WillReturnRows(sqlmock.NewRows([]string{"dailyNotificationTime", "weeklyNotificationTime", "reminderOffset"}).
			AddRow("18:00", "SUN 18:00", 30))

	mockBot.On("NewMessage", chatID, mock.AnythingOfType("string")).Return(tgbotapi.MessageConfig{})
	mockBot.On("Send", mock.AnythingOfType("tgbotapi.MessageConfig")).Return(tgbotapi.Message{}, nil)

	err = notifications.SendDailySummary(mockBot, db, chatID, server.URL)
	assert.NoError(t, err)

	mockBot.AssertExpectations(t)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestSendWeeklySummary(t *testing.T) {
	mockBot := new(MockBotAPI)
	chatID := int64(123456)

	server := createMockCalendarServer()
	defer server.Close()

	db, sqlMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlMock.ExpectQuery("SELECT dailyNotificationTime, weeklyNotificationTime, reminderOffset FROM users WHERE chatID = ?").
		WithArgs(chatID).
		WillReturnRows(sqlmock.NewRows([]string{"dailyNotificationTime", "weeklyNotificationTime", "reminderOffset"}).
			AddRow("18:00", "SUN 18:00", 30))

	mockBot.On("NewMessage", chatID, mock.AnythingOfType("string")).Return(tgbotapi.MessageConfig{})
	mockBot.On("Send", mock.AnythingOfType("tgbotapi.MessageConfig")).Return(tgbotapi.Message{}, nil)

	err = notifications.SendWeeklySummary(mockBot, db, chatID, server.URL)
	assert.NoError(t, err)

	mockBot.AssertExpectations(t)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestSendReminder(t *testing.T) {
	mockBot := new(MockBotAPI)
	chatID := int64(123456)

	event := ical.NewEvent("test-event")
	event.SetProperty(ical.ComponentPropertySummary, "Test Lecture")
	event.SetProperty(ical.ComponentPropertyLocation, "Room 101")
	event.SetProperty(ical.ComponentPropertyDtStart, time.Now().In(utils.GetUKLocation()).Add(30*time.Minute).Format("20060102T150405Z"))

	mockBot.On("NewMessage", chatID, mock.AnythingOfType("string")).Return(tgbotapi.MessageConfig{})
	mockBot.On("Send", mock.AnythingOfType("tgbotapi.MessageConfig")).Return(tgbotapi.Message{}, nil)

	err := notifications.SendReminder(mockBot, chatID, event)
	assert.NoError(t, err)

	mockBot.AssertExpectations(t)
}
