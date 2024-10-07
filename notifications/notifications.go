package notifications

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	ical "github.com/arran4/golang-ical"
	"github.com/artem-streltsov/ucl-timetable-bot/common"
	"github.com/artem-streltsov/ucl-timetable-bot/utils"
)

func getDayLectures(calendar *ical.Calendar) []*ical.VEvent {
	today := utils.CurrentTimeUK().Format("20060102")
	lectures := []*ical.VEvent{}
	for _, event := range calendar.Events() {
		start := event.GetProperty(ical.ComponentPropertyDtStart)
		if start != nil && strings.HasPrefix(start.Value, today) {
			lectures = append(lectures, event)
		}
	}
	return lectures
}

func SendDailySummary(bot common.BotAPI, db *sql.DB, chatID int64, webcalURL string) error {
	response, err := http.Get(webcalURL)
	if err != nil {
		return fmt.Errorf("error fetching calendar: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("error reading calendar: %w", err)
	}

	calendar, err := ical.ParseCalendar(strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("error parsing calendar: %w", err)
	}

	lecturesThisDay := getDayLectures(calendar)
	if len(lecturesThisDay) == 0 {
		msg := bot.NewMessage(chatID, "No lectures scheduled for today.")
		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("error sending no lectures message: %w", err)
		}
		return nil
	}

	message := fmt.Sprintf("Today's Lectures (all times are in UK time):\n")
	for _, lecture := range lecturesThisDay {
		message += FormatEventDetails(lecture) + "\n"
	}

	msg := bot.NewMessage(chatID, message)
	if _, err := bot.Send(msg); err != nil {
		return fmt.Errorf("error sending daily summary: %w", err)
	}
	return nil
}

func getWeekLectures(calendar *ical.Calendar) map[string][]*ical.VEvent {
	lectures := make(map[string][]*ical.VEvent)
	now := utils.CurrentTimeUK()
	offset := int(time.Monday - now.Weekday())
	if offset > 0 {
		offset = -6 // Go back to previous Monday if today is Sunday
	}

	monday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, utils.GetUKLocation()).AddDate(0, 0, offset)
	friday := monday.AddDate(0, 0, 4).Add(24 * time.Hour)

	daysOfWeek := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"}

	for _, event := range calendar.Events() {
		startProp := event.GetProperty(ical.ComponentPropertyDtStart)
		if startProp == nil {
			continue
		}
		startTimeStr := startProp.Value
		startTime, err := time.Parse("20060102T150405Z", startTimeStr)
		if err != nil {
			log.Printf("Error parsing start time: %v", err)
			continue
		}

		startTime = startTime.In(utils.GetUKLocation())

		if startTime.After(monday) && startTime.Before(friday) {
			weekday := daysOfWeek[int(startTime.Weekday())-1]
			lectures[weekday] = append(lectures[weekday], event)
		}
	}

	return lectures
}

func SendWeeklySummary(bot common.BotAPI, db *sql.DB, chatID int64, webcalURL string) error {
	response, err := http.Get(webcalURL)
	if err != nil {
		return fmt.Errorf("error fetching calendar: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("error reading calendar: %w", err)
	}

	calendar, err := ical.ParseCalendar(strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("error parsing calendar: %w", err)
	}

	lecturesThisWeek := getWeekLectures(calendar)
	if len(lecturesThisWeek) == 0 {
		msg := bot.NewMessage(chatID, "No lectures scheduled for this week.")
		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("error sending no lectures message: %w", err)
		}
		return nil
	}

	message := fmt.Sprintf("This Week's Lectures (all times are in UK time):\n")
	for day, lectures := range lecturesThisWeek {
		message += fmt.Sprintf("\n%s:\n", day)
		for _, lecture := range lectures {
			message += FormatEventDetails(lecture) + "\n"
		}
	}

	msg := bot.NewMessage(chatID, message)
	if _, err := bot.Send(msg); err != nil {
		return fmt.Errorf("error sending weekly summary: %w", err)
	}
	return nil
}

func SendReminder(bot common.BotAPI, chatID int64, lecture *ical.VEvent) error {
	message := fmt.Sprintf("Reminder: Your lecture is starting soon!\n(All times are in UK time)\n\n")
	message += FormatEventDetails(lecture)
	msg := bot.NewMessage(chatID, message)
	if _, err := bot.Send(msg); err != nil {
		return fmt.Errorf("error sending reminder: %w", err)
	}
	return nil
}

func cleanSummary(summary string) string {
	re := regexp.MustCompile(`\s*$begin:math:display$.*?$end:math:display$`)
	return re.ReplaceAllString(summary, "")
}

func FormatEventDetails(event *ical.VEvent) string {
	summary := "Unknown"
	location := "Unknown"
	startTime := "Unknown"
	endTime := "Unknown"

	if summaryProp := event.GetProperty(ical.ComponentPropertySummary); summaryProp != nil {
		rawSummary := summaryProp.Value
		summary = cleanSummary(rawSummary)
		summary = strings.TrimSpace(summary)
	}

	if locationProp := event.GetProperty(ical.ComponentPropertyLocation); locationProp != nil {
		location = locationProp.Value
	}

	if startProp := event.GetProperty(ical.ComponentPropertyDtStart); startProp != nil {
		if start, err := time.Parse("20060102T150405Z", startProp.Value); err == nil {
			ukTime := start.In(utils.GetUKLocation())
			startTime = ukTime.Format("15:04")
		}
	}

	if endProp := event.GetProperty(ical.ComponentPropertyDtEnd); endProp != nil {
		if end, err := time.Parse("20060102T150405Z", endProp.Value); err == nil {
			ukEndTime := end.In(utils.GetUKLocation())
			endTime = ukEndTime.Format("15:04")
		}
	}

	return fmt.Sprintf("📚 Lecture: %s\n⏰ Time: %s - %s\n📍Location: %s\n", summary, startTime, endTime, location)
}
