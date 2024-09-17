package notifications_test

import (
	"testing"
	"time"

	ical "github.com/arran4/golang-ical"
	"github.com/artem-streltsov/ucl-timetable-bot/notifications"
	"github.com/stretchr/testify/assert"
)

func TestFormatEventDetails(t *testing.T) {
	fixedTime := "20230915T090000Z"
	startTime, err := time.Parse("20060102T150405Z", fixedTime)
	if err != nil {
		t.Fatalf("Error parsing fixed time: %v", err)
	}
	formattedTime := startTime.Format("15:04")

	event := ical.NewEvent("test")
	event.SetProperty("SUMMARY", "Lecture on Go")
	event.SetProperty("LOCATION", "Room 101")
	event.SetProperty("DTSTART", fixedTime)

	result := notifications.FormatEventDetails(event)
	expected := "- Lecture on Go at Room 101, starting at " + formattedTime
	assert.Equal(t, expected, result)
}
