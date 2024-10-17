package handlers

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/artem-streltsov/ucl-timetable-bot/database"
	"github.com/artem-streltsov/ucl-timetable-bot/scheduler"
	"github.com/artem-streltsov/ucl-timetable-bot/timetable"
	"github.com/artem-streltsov/ucl-timetable-bot/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var defaultDailyTime = "07:00"
var defaultWeeklyTime = "SUN 18:00"
var ukLocation, _ = time.LoadLocation("Europe/London")

type Handler struct {
	api        *tgbotapi.BotAPI
	db         *database.DB
	scheduler  *scheduler.Scheduler
	userStates map[int64]string
	mu         sync.RWMutex
}

func NewHandler(api *tgbotapi.BotAPI, db *database.DB, scheduler *scheduler.Scheduler) *Handler {
	return &Handler{
		api:        api,
		db:         db,
		scheduler:  scheduler,
		userStates: make(map[int64]string),
	}
}

func (h *Handler) HandleCommand(chatID int64, cmd string) {
	switch cmd {
	case "start":
		h.sendMessage(chatID, "Welcome! Use /set\\_calendar to set your Calendar link.")
	case "today":
		h.today(chatID)
	case "tomorrow":
		h.tomorrow(chatID)
	case "week":
		h.week(chatID)
	case "settings":
		h.settings(chatID)
	case "set_daily_time":
		h.updateUserState(chatID, "set_daily_time")
		h.sendMessage(chatID, "Send your daily notification time. Example: 07:00.")
	case "set_weekly_time":
		h.updateUserState(chatID, "set_weekly_time")
		h.sendMessage(chatID, "Send your weekly notification day and time. Example: SUN 18:00.")
	case "set_calendar":
		h.updateUserState(chatID, "set_calendar")
		h.sendMessage(chatID, "Send your Calendar link.\nIt can be found in Portico -> My Studies -> Timetable -> Add to Calendar -> Copy Calendar Link.\nIt must start with webcal://")
	default:
		h.sendMessage(chatID, "Unknown command. Use commands from the menu.")
	}
}

func (h *Handler) HandleMessage(chatID int64, text string) {
	state := h.getUserState(chatID)
	switch state {
	case "set_daily_time":
		h.handleSetDailyTime(chatID, text)
	case "set_weekly_time":
		h.handleSetWeeklyTime(chatID, text)
	case "set_calendar":
		h.handleSetCalendar(chatID, text)
	default:
		h.sendMessage(chatID, "Please use commands from the menu to interact with the bot.")
	}
}

func (h *Handler) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	if _, err := h.api.Send(msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func (h *Handler) updateUserState(chatID int64, state string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.userStates[chatID] = state
}

func (h *Handler) getUserState(chatID int64) string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.userStates[chatID]
}

func (h *Handler) clearUserState(chatID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.userStates, chatID)
}

func (h *Handler) today(chatID int64) {
	h.sendTimetable(chatID, time.Now().In(ukLocation), time.Now(), "today")
}

func (h *Handler) tomorrow(chatID int64) {
	tomorrow := time.Now().In(ukLocation).AddDate(0, 0, 1)
	h.sendTimetable(chatID, tomorrow, tomorrow, "tomorrow")
}

func (h *Handler) week(chatID int64) {
	now := time.Now().In(ukLocation)
	weekday := now.Weekday()

	var weekStart time.Time
	var weekEnd time.Time
	var period string

	if weekday == time.Saturday {
		daysUntilMonday := (time.Monday + 7 - weekday) % 7
		weekStart = now.AddDate(0, 0, int(daysUntilMonday))
		period = "next week"
	} else if weekday == time.Sunday {
		daysUntilMonday := (time.Monday + 7 - weekday) % 7
		weekStart = now.AddDate(0, 0, int(daysUntilMonday))
		period = "next week"
	} else {
		daysSinceMonday := (weekday - time.Monday + 7) % 7
		weekStart = now.AddDate(0, 0, -int(daysSinceMonday))
		period = "this week"
	}

	weekEnd = weekStart.AddDate(0, 0, 4) // Friday

	h.sendTimetable(chatID, weekStart, weekEnd, period)
}

func (h *Handler) sendTimetable(chatID int64, startDate, endDate time.Time, period string) {
	user, _ := h.db.GetUser(chatID)
	if user == nil || user.WebCalURL == "" {
		h.sendMessage(chatID, "Please set your calendar link using /set_calendar")
		return
	}
	cal, err := timetable.FetchCalendar(user.WebCalURL)
	if err != nil {
		h.sendMessage(chatID, "Error fetching calendar")
		return
	}

	if startDate.Day() == endDate.Day() {
		lectures, err := timetable.GetLectures(cal, startDate)
		if err != nil {
			h.sendMessage(chatID, "Error processing calendar")
			return
		}
		if len(lectures) == 0 {
			h.sendMessage(chatID, fmt.Sprintf("No lectures %s.", period))
			return
		}
		dateStr := startDate.Format("Mon, 02 Jan")
		message := fmt.Sprintf("*%s:*\n\n", dateStr) + timetable.FormatLectures(lectures)
		h.sendMessage(chatID, message)
	} else {
		lecturesMap, err := timetable.GetLecturesInRange(cal, startDate, endDate)
		if err != nil {
			h.sendMessage(chatID, "Error processing calendar: "+err.Error())
			return
		}
		if len(lecturesMap) == 0 {
			h.sendMessage(chatID, fmt.Sprintf("No lectures %s.", period))
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
		h.sendMessage(chatID, sb.String())
	}
}

func (h *Handler) settings(chatID int64) {
	user, _ := h.db.GetUser(chatID)
	if user == nil {
		user = &database.User{ChatID: chatID, DailyTime: defaultDailyTime, WeeklyTime: defaultWeeklyTime}
	}
	h.sendMessage(chatID, fmt.Sprintf("Your settings:\nDaily notification time: %v\nWeekly notification day and time: %v", user.DailyTime, user.WeeklyTime))
	if user.WebCalURL == "" {
		h.sendMessage(chatID, "Your Calendar link is not set. Use /set_calendar to set it.")
	}
}

func (h *Handler) handleSetCalendar(chatID int64, text string) {
	if !strings.HasPrefix(strings.ToLower(text), "webcal://") {
		h.sendMessage(chatID, "Calendar link must start with webcal://")
		return
	}
	user, _ := h.db.GetUser(chatID)
	if user == nil {
		user = &database.User{ChatID: chatID, DailyTime: defaultDailyTime, WeeklyTime: defaultWeeklyTime}
	}
	user.WebCalURL = text
	h.db.SaveUser(user)
	h.scheduler.ScheduleUser(chatID)
	h.sendMessage(chatID, "Calendar link saved.")
	h.clearUserState(chatID)
}

func (h *Handler) handleSetDailyTime(chatID int64, text string) {
	if !utils.IsValidTime(text) {
		h.sendMessage(chatID, "Invalid format. Use HH:MM format.")
		return
	}
	user, _ := h.db.GetUser(chatID)
	if user == nil {
		user = &database.User{ChatID: chatID, DailyTime: defaultDailyTime, WeeklyTime: defaultWeeklyTime}
	}
	user.DailyTime = text
	h.db.SaveUser(user)
	h.scheduler.ScheduleUser(chatID)
	h.sendMessage(chatID, "Daily notification time saved.")
	h.clearUserState(chatID)
}

func (h *Handler) handleSetWeeklyTime(chatID int64, text string) {
	parts := strings.SplitN(text, " ", 2)
	if len(parts) != 2 || !utils.IsValidDay(parts[0]) || !utils.IsValidTime(parts[1]) {
		h.sendMessage(chatID, "Invalid format. Use DAY HH:MM.")
		return
	}
	user, _ := h.db.GetUser(chatID)
	if user == nil {
		user = &database.User{ChatID: chatID, DailyTime: defaultDailyTime, WeeklyTime: defaultWeeklyTime}
	}
	user.WeeklyTime = text
	h.db.SaveUser(user)
	h.scheduler.ScheduleUser(chatID)
	h.sendMessage(chatID, "Weekly notification time saved.")
	h.clearUserState(chatID)
}
