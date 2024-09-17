package scheduler_test

import (
	"testing"
	"time"

	ical "github.com/arran4/golang-ical"
	"github.com/artem-streltsov/ucl-timetable-bot/scheduler"
)

func TestGetNextSunday(t *testing.T) {
	tests := []struct {
		now         time.Time
		expectedDay time.Weekday
	}{
		{time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC), time.Sunday},
		{time.Date(2024, 9, 22, 0, 0, 0, 0, time.UTC), time.Sunday},
	}

	for _, test := range tests {
		nextSunday := scheduler.GetNextSunday(test.now)
		if nextSunday.Weekday() != test.expectedDay {
			t.Errorf("expected %v, got %v", test.expectedDay, nextSunday.Weekday())
		}
	}
}

func TestShouldSendDailySummary(t *testing.T) {
	now := time.Now()

	tests := []struct {
		lastSentTime time.Time
		shouldSend   bool
	}{
		{now.AddDate(0, 0, -2), true},
		{now.AddDate(0, 0, -1), true},
		{now.AddDate(0, 0, 0), false},
	}

	for _, test := range tests {
		if scheduler.ShouldSendDailySummary(test.lastSentTime, now) != test.shouldSend {
			t.Errorf("expected %v, got %v", test.shouldSend, !test.shouldSend)
		}
	}
}

func TestParseLectureStartTime(t *testing.T) {
	lecture := ical.NewEvent("lecture-123")
	lecture.SetProperty("DTSTART", "20240917T150000Z")

	startTime, err := scheduler.ParseLectureStartTime(lecture)
	if err != nil {
		t.Errorf("Error parsing lecture start time: %v", err)
	}

	expectedTime := time.Date(2024, 9, 17, 15, 0, 0, 0, time.UTC)
	if !startTime.Equal(expectedTime) {
		t.Errorf("Expected %v, got %v", expectedTime, startTime)
	}
}

func TestShouldScheduleReminder(t *testing.T) {
	now := time.Now()

	tests := []struct {
		reminderTime   time.Time
		shouldSchedule bool
	}{
		{now.Add(30 * time.Minute), true},
		{now.Add(-1 * time.Hour), false},
	}

	for _, test := range tests {
		if scheduler.ShouldScheduleReminder(test.reminderTime) != test.shouldSchedule {
			t.Errorf("expected %v, got %v", test.shouldSchedule, !test.shouldSchedule)
		}
	}
}
