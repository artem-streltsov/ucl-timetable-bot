package notifications

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode"

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

	today := time.Now().In(utils.GetUKLocation())
	dayWithSuffix := utils.GetDayWithSuffix(today.Day())
	formattedDate := today.Format("Mon,") + " " + dayWithSuffix + " " + today.Format("Jan")

	message := fmt.Sprintf("**%s - Lectures Today:**\n\n", formattedDate)
	for _, lecture := range lecturesThisDay {
		message += FormatEventDetails(lecture) + "\n"
	}

	msg := bot.NewMessage(chatID, message)
	msg.ParseMode = "Markdown"
	if _, err := bot.Send(msg); err != nil {
		return fmt.Errorf("error sending daily summary: %w", err)
	}
	return nil
}

func getWeekLectures(calendar *ical.Calendar, startDay, endDay time.Time) map[string][]*ical.VEvent {
	lectures := make(map[string][]*ical.VEvent)

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
		if startTime.After(startDay) && startTime.Before(endDay) {
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

	now := utils.CurrentTimeUK()
	var startDay, endDay time.Time

	switch now.Weekday() {
	case time.Monday:
		startDay = now                // Start from today
		endDay = now.AddDate(0, 0, 4) // Upcoming Friday
	case time.Tuesday, time.Wednesday, time.Thursday, time.Friday:
		startDay = now.AddDate(0, 0, int(time.Monday-now.Weekday())) // Previous Monday
		endDay = now.AddDate(0, 0, 4)                                // Upcoming Friday
	case time.Saturday:
		startDay = now.AddDate(0, 0, 2) // Next Monday
		endDay = now.AddDate(0, 0, 7)   // Next Friday
	case time.Sunday:
		startDay = now.AddDate(0, 0, 1) // Next Monday
		endDay = now.AddDate(0, 0, 6)   // Next Friday
	}

	lecturesThisWeek := getWeekLectures(calendar, startDay, endDay)
	if len(lecturesThisWeek) == 0 {
		msg := bot.NewMessage(chatID, "No lectures scheduled for this week.")
		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("error sending no lectures message: %w", err)
		}
		return nil
	}

	formattedWeekRange := utils.FormatWeekRange(startDay, endDay)

	message := fmt.Sprintf("**%s - Lectures:**\n", formattedWeekRange)
	for day, lectures := range lecturesThisWeek {
		message += fmt.Sprintf("\n📅 **%s:**\n", day)
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
	openBracketIndex := strings.Index(summary, "[")
	if openBracketIndex != -1 {
		closeBracketIndex := strings.Index(summary, "]")
		if closeBracketIndex != -1 && closeBracketIndex > openBracketIndex {
			summary = summary[:openBracketIndex] + summary[closeBracketIndex+1:]
		}
	}

	summary = removeLevelPattern(summary)

	return strings.TrimSpace(summary)
}

func removeLevelPattern(summary string) string {
	for {
		levelIndex := strings.Index(summary, "Level")
		if levelIndex == -1 {
			break
		}

		numberStartIndex := levelIndex + len("Level")
		if numberStartIndex >= len(summary) {
			break
		}

		if !unicode.IsSpace(rune(summary[numberStartIndex])) {
			break
		}

		numberEndIndex := numberStartIndex + 1
		for numberEndIndex < len(summary) && unicode.IsDigit(rune(summary[numberEndIndex])) {
			numberEndIndex++
		}

		summary = strings.TrimSpace(summary[:levelIndex] + summary[numberEndIndex:])
	}

	return summary
}

func FormatEventDetails(event *ical.VEvent) string {
	category := "Unknown"
	summary := "Unknown"
	location := "Unknown"
	startTime := "Unknown"
	endTime := "Unknown"

	if categoryProp := event.GetProperty(ical.ComponentPropertyCategories); categoryProp != nil {
		category = categoryProp.Value
	}

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

	return fmt.Sprintf("📚 %s: **%s**\n⏰ Time: %s - %s\n📍Location: %s\n", category, summary, startTime, endTime, location)
}
