package notifications

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	ical "github.com/arran4/golang-ical"
	"github.com/artem-streltsov/ucl-timetable-bot/common"
)

const reminderOffset = time.Minute * 30

func getDayLectures(calendar *ical.Calendar) []*ical.VEvent {
	today := time.Now().UTC().Format("20060102")
	lectures := []*ical.VEvent{}
	for _, event := range calendar.Events() {
		start := event.GetProperty("DTSTART").Value
		if strings.HasPrefix(start, today) {
			lectures = append(lectures, event)
		}
	}
	return lectures
}

func SendDailySummary(bot common.BotAPI, chatID int64, webcalURL string) {
	response, err := http.Get(webcalURL)
	if err != nil {
		log.Println("Error fetching calendar for chatID:", chatID, err)
		return
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println("Error reading calendar for chatID:", chatID, err)
		return
	}

	calendar, err := ical.ParseCalendar(strings.NewReader(string(body)))
	if err != nil {
		log.Println("Error parsing calendar for chatID:", chatID, err)
		return
	}

	lecturesThisDay := getDayLectures(calendar)
	if len(lecturesThisDay) == 0 {
		msg := bot.NewMessage(chatID, "No lectures scheduled for today.")
		bot.Send(msg)
		return
	}

	message := "Today's Lectures:\n"
	for _, lecture := range lecturesThisDay {
		message += FormatEventDetails(lecture) + "\n"
	}

	msg := bot.NewMessage(chatID, message)
	bot.Send(msg)
}

func getWeekLectures(calendar *ical.Calendar) map[string][]*ical.VEvent {
	lectures := make(map[string][]*ical.VEvent)
	now := time.Now().UTC()
	offset := int(time.Monday - now.Weekday())
	if offset > 0 {
		offset = -6 // Go back to previous Monday if today is Sunday
	}

	monday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, offset)
	friday := monday.AddDate(0, 0, 4).Add(24 * time.Hour)

	daysOfWeek := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"}

	for _, event := range calendar.Events() {
		startTimeStr := event.GetProperty("DTSTART").Value
		startTime, err := time.Parse("20060102T150405Z", startTimeStr)
		if err != nil {
			continue
		}

		if startTime.After(monday) && startTime.Before(friday) {
			weekday := daysOfWeek[int(startTime.Weekday())-1]
			lectures[weekday] = append(lectures[weekday], event)
		}
	}

	return lectures
}

func SendWeeklySummary(bot common.BotAPI, chatID int64, webcalURL string) {
	response, err := http.Get(webcalURL)
	if err != nil {
		log.Println("Error fetching calendar for chatID:", chatID, err)
		return
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println("Error reading calendar for chatID:", chatID, err)
		return
	}

	calendar, err := ical.ParseCalendar(strings.NewReader(string(body)))
	if err != nil {
		log.Println("Error parsing calendar for chatID:", chatID, err)
		return
	}

	lecturesThisWeek := getWeekLectures(calendar)
	if len(lecturesThisWeek) == 0 {
		msg := bot.NewMessage(chatID, "No lectures scheduled for this week.")
		bot.Send(msg)
		return
	}

	message := "This Week's Lectures:\n"
	for day, lectures := range lecturesThisWeek {
		message += fmt.Sprintf("\n%s:\n", day)
		for _, lecture := range lectures {
			message += FormatEventDetails(lecture) + "\n"
		}
	}

	msg := bot.NewMessage(chatID, message)
	bot.Send(msg)
}

func SendReminder(bot common.BotAPI, chatID int64, lecture *ical.VEvent) {
	message := fmt.Sprintf("Reminder: Your lecture is starting in %v minutes!\n", reminderOffset)
	message += FormatEventDetails(lecture)
	msg := bot.NewMessage(chatID, message)
	bot.Send(msg)
}

func FormatEventDetails(event *ical.VEvent) string {
	summary := event.GetProperty("SUMMARY").Value
	location := event.GetProperty("LOCATION").Value
	startTime := event.GetProperty("DTSTART").Value
	start, _ := time.Parse("20060102T150405Z", startTime)
	return fmt.Sprintf("- %s at %s, starting at %s", summary, location, start.Format("15:04"))
}
