package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/artem-streltsov/ucl-timetable-bot/models"
	"github.com/artem-streltsov/ucl-timetable-bot/timetable"
)

func (h *Handler) today(user *models.User) {
	h.sendTimetable(user, time.Now().In(ukLocation), time.Now(), "today")
}

func (h *Handler) tomorrow(user *models.User) {
	tomorrow := time.Now().In(ukLocation).AddDate(0, 0, 1)
	h.sendTimetable(user, tomorrow, tomorrow, "tomorrow")
}

func (h *Handler) week(user *models.User) {
	now := time.Now().In(ukLocation)
	weekday := now.Weekday()

	var weekStart time.Time
	var weekEnd time.Time
	var period string

	if weekday == time.Saturday || weekday == time.Sunday {
		daysUntilMonday := (time.Monday + 7 - weekday) % 7
		weekStart = now.AddDate(0, 0, int(daysUntilMonday))
		period = "next week"
	} else {
		daysSinceMonday := (weekday - time.Monday + 7) % 7
		weekStart = now.AddDate(0, 0, -int(daysSinceMonday))
		period = "this week"
	}

	weekEnd = weekStart.AddDate(0, 0, 4) // Friday

	h.sendTimetable(user, weekStart, weekEnd, period)
}

func (h *Handler) sendTimetable(user *models.User, startDate, endDate time.Time, period string) {
	if user.WebCalURL == "" {
		h.sendMessage(user.ChatID, "Please set your calendar link using /set_calendar")
		return
	}
	cal, err := timetable.FetchCalendar(user.WebCalURL)
	if err != nil {
		h.sendMessage(user.ChatID, "Error fetching calendar")
		return
	}

	if startDate.Day() == endDate.Day() {
		lectures, err := timetable.GetLectures(cal, startDate)
		if err != nil {
			h.sendMessage(user.ChatID, "Error processing calendar")
			return
		}
		if len(lectures) == 0 {
			h.sendMessage(user.ChatID, fmt.Sprintf("No lectures %s.", period))
			return
		}
		dateStr := startDate.Format("Mon, 02 Jan")
		message := fmt.Sprintf("*%s:*\n\n", dateStr) + timetable.FormatLectures(lectures)
		h.sendMessage(user.ChatID, message)
	} else {
		lecturesMap, err := timetable.GetLecturesInRange(cal, startDate, endDate)
		if err != nil {
			h.sendMessage(user.ChatID, "Error processing calendar: "+err.Error())
			return
		}
		if len(lecturesMap) == 0 {
			h.sendMessage(user.ChatID, fmt.Sprintf("No lectures %s.", period))
			return
		}
		startDateStr := startDate.Format("Mon, 02 Jan")
		endDateStr := endDate.Format("Fri, 02 Jan")
		dateRangeStr := fmt.Sprintf("*%s - %s:*\n\n", startDateStr, endDateStr)

		var sb strings.Builder
		sb.WriteString(dateRangeStr)
		for day := startDate; !day.After(endDate); day = day.AddDate(0, 0, 1) {
			dayKey := day.Format("Monday")
			lectures, ok := lecturesMap[dayKey]
			if ok {
				sb.WriteString("\n" + "*" + dayKey + "*" + "\n")
				message := timetable.FormatLectures(lectures)
				sb.WriteString(message)
			}
		}
		h.sendMessage(user.ChatID, sb.String())
	}
}
