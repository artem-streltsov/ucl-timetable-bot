package notifications

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
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

	message := fmt.Sprintf("*%s:*\n\n", formattedDate)
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

		if !startTime.Before(startDay) && !startTime.After(endDay) {
			weekday := startTime.Weekday().String()
			lectures[weekday] = append(lectures[weekday], event)
		}
	}

	for _, events := range lectures {
		sort.Slice(events, func(i, j int) bool {
			ti, _ := time.Parse("20060102T150405Z", events[i].GetProperty(ical.ComponentPropertyDtStart).Value)
			tj, _ := time.Parse("20060102T150405Z", events[j].GetProperty(ical.ComponentPropertyDtStart).Value)
			return ti.Before(tj)
		})
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

	weekday := now.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		daysUntilMonday := (int(time.Monday) - int(weekday) + 7) % 7
		startDay = now.AddDate(0, 0, daysUntilMonday)
	} else {
		daysSinceMonday := (int(weekday) - int(time.Monday) + 7) % 7
		startDay = now.AddDate(0, 0, -daysSinceMonday)
	}
	endDay = startDay.AddDate(0, 0, 4)

	location := utils.GetUKLocation()
	startDay = time.Date(startDay.Year(), startDay.Month(), startDay.Day(), 0, 0, 0, 0, location)
	endDay = time.Date(endDay.Year(), endDay.Month(), endDay.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), location)

	lecturesThisWeek := getWeekLectures(calendar, startDay, endDay)
	if len(lecturesThisWeek) == 0 {
		msg := bot.NewMessage(chatID, "No lectures scheduled for this week.")
		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("error sending no lectures message: %w", err)
		}
		return nil
	}

	daysOfWeek := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"}
	formattedWeekRange := utils.FormatWeekRange(startDay, endDay)

	message := fmt.Sprintf("*%s:*\n", formattedWeekRange)
	for _, day := range daysOfWeek {
		lectures := lecturesThisWeek[day]
		if len(lectures) == 0 {
			continue
		}
		message += fmt.Sprintf("\n*%s:*\n", day)
		for _, lecture := range lectures {
			message += FormatEventDetails(lecture) + "\n"
		}
	}

	msg := bot.NewMessage(chatID, message)
	msg.ParseMode = "Markdown"
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

	return fmt.Sprintf("📚 *%s*\n⏰ %s - %s\n📍%s\n", summary, startTime, endTime, location)
}
