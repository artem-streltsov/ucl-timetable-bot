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
				event.SetProperty(ical.ComponentPropertyCategories, "Lecture")
				event.SetProperty(ical.ComponentPropertySummary, "Test Lecture Level 6 [Lecture]")
				event.SetProperty(ical.ComponentPropertyLocation, "Room 101")
				event.SetProperty(ical.ComponentPropertyDtStart, "20230115T090000Z") // January 15, 2023 (winter)
				return event
			}(),
			expected: "📚 Lecture: **Test Lecture**\n⏰ Time: 09:00 - Unknown\n📍Location: Room 101\n",
		},
		{
			name: "Summer time (BST)",
			event: func() *ical.VEvent {
				event := ical.NewEvent("test-event")
				event.SetProperty(ical.ComponentPropertyCategories, "Computer Practical")
				event.SetProperty(ical.ComponentPropertySummary, "Test Lecture [Computer Practical] Level 5")
				event.SetProperty(ical.ComponentPropertyLocation, "Room 101")
				event.SetProperty(ical.ComponentPropertyDtStart, "20230615T090000Z") // June 15, 2023 (summer)
				return event
			}(),
			expected: "📚 Computer Practical: **Test Lecture**\n⏰ Time: 10:00 - Unknown\n📍Location: Room 101\n",
		},
		{
			name: "Missing summary",
			event: func() *ical.VEvent {
				event := ical.NewEvent("test-event")
				event.SetProperty(ical.ComponentPropertyCategories, "Lecture")
				event.SetProperty(ical.ComponentPropertyLocation, "Room 101")
				event.SetProperty(ical.ComponentPropertyDtStart, "20230515T090000Z")
				return event
			}(),
			expected: "📚 Lecture: **Unknown**\n⏰ Time: 10:00 - Unknown\n📍Location: Room 101\n",
		},
		{
			name: "Missing location",
			event: func() *ical.VEvent {
				event := ical.NewEvent("test-event")
				event.SetProperty(ical.ComponentPropertyCategories, "Lecture")
				event.SetProperty(ical.ComponentPropertySummary, "Test Lecture")
				event.SetProperty(ical.ComponentPropertyDtStart, "20230515T090000Z")
				return event
			}(),
			expected: "📚 Lecture: **Test Lecture**\n⏰ Time: 10:00 - Unknown\n📍Location: Unknown\n",
		},
		{
			name: "Missing start time",
			event: func() *ical.VEvent {
				event := ical.NewEvent("test-event")
				event.SetProperty(ical.ComponentPropertyCategories, "Lecture")
				event.SetProperty(ical.ComponentPropertySummary, "Test Lecture")
				event.SetProperty(ical.ComponentPropertyLocation, "Room 101")
				return event
			}(),
			expected: "📚 Lecture: **Test Lecture**\n⏰ Time: Unknown - Unknown\n📍Location: Room 101\n",
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
	assert.NoError(t, err)
	defer db.Close()

	mockBot.On("NewMessage", chatID, mock.AnythingOfType("string")).Return(tgbotapi.MessageConfig{})
	mockBot.On("Send", mock.AnythingOfType("tgbotapi.MessageConfig")).Return(tgbotapi.Message{}, nil)

	err = notifications.SendDailySummary(mockBot, db, chatID, server.URL)
	assert.NoError(t, err)

	assert.NoError(t, sqlMock.ExpectationsWereMet())
	mockBot.AssertExpectations(t)
}

func TestSendWeeklySummary(t *testing.T) {
	mockBot := new(MockBotAPI)
	chatID := int64(123456)

	server := createMockCalendarServer()
	defer server.Close()

	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mockBot.On("NewMessage", chatID, mock.AnythingOfType("string")).Return(tgbotapi.MessageConfig{})
	mockBot.On("Send", mock.AnythingOfType("tgbotapi.MessageConfig")).Return(tgbotapi.Message{}, nil)

	err = notifications.SendWeeklySummary(mockBot, db, chatID, server.URL)
	assert.NoError(t, err)

	assert.NoError(t, sqlMock.ExpectationsWereMet())
	mockBot.AssertExpectations(t)
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
