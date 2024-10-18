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
var defaultReminderOffset = "15"
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

func (h *Handler) registerUser(chatID int64, username string) (*database.User, error) {
	user, err := h.db.GetUser(chatID)
	if err != nil {
		h.sendMessage(chatID, fmt.Sprintf("Error: %v", err))
		return nil, err
	}
	if user == nil {
		user = &database.User{
			ChatID:         chatID,
			Username:       username,
			DailyTime:      defaultDailyTime,
			WeeklyTime:     defaultWeeklyTime,
			ReminderOffset: defaultReminderOffset,
		}
		h.db.SaveUser(user)
	} else if user.Username != username {
		user.Username = username
		h.db.SaveUser(user)
	}
	return user, nil
}

func (h *Handler) HandleCommand(chatID int64, cmd string, username string) {
	user, err := h.registerUser(chatID, username)
	if err != nil {
		h.sendMessage(chatID, "Error.")
	}

	switch cmd {
	case "start":
		h.sendMessage(chatID, "Welcome! Use /set\\_calendar to set your Calendar link.")
	case "today":
		h.today(user)
	case "tomorrow":
		h.tomorrow(user)
	case "week":
		h.week(user)
	case "settings":
		h.settings(user)
	case "add_friend":
		h.updateUserState(chatID, "add_friend")
		h.sendMessage(chatID, "Send your friend's username. Example: @username.")
	case "accept_friend":
		h.updateUserState(chatID, "accept_friend")
		h.handleAcceptFriend(user)
	case "set_daily_time":
		h.updateUserState(chatID, "set_daily_time")
		h.sendMessage(chatID, "Send your daily notification time. Example: 07:00.")
	case "set_weekly_time":
		h.updateUserState(chatID, "set_weekly_time")
		h.sendMessage(chatID, "Send your weekly notification day and time. Example: SUN 18:00.")
	case "set_reminder_offset":
		h.updateUserState(chatID, "set_reminder_offset")
		h.sendMessage(chatID, "Send your lectures reminder offset in minutes. Example: 15")
	case "set_calendar":
		h.updateUserState(chatID, "set_calendar")
		h.sendMessage(chatID, "Send your Calendar link.\nIt can be found in Portico -> My Studies -> Timetable -> Add to Calendar -> Copy Calendar Link.\nIt must start with webcal://")
	default:
		h.sendMessage(chatID, "Unknown command. Use commands from the menu.")
	}
}

func (h *Handler) HandleMessage(chatID int64, text string, username string) {
	user, err := h.registerUser(chatID, username)
	if err != nil {
		h.sendMessage(chatID, "Error.")
	}

	state := h.getUserState(chatID)
	switch state {
	case "add_friend":
		h.handleAddFriend(user, text)
	case "accept_friend":
		h.handleAcceptFriend(user)
	case "set_daily_time":
		h.handleSetDailyTime(user, text)
	case "set_weekly_time":
		h.handleSetWeeklyTime(user, text)
	case "set_reminder_offset":
		h.handleSetReminderOffset(user, text)
	case "set_calendar":
		h.handleSetCalendar(user, text)
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

func (h *Handler) today(user *database.User) {
	h.sendTimetable(user, time.Now().In(ukLocation), time.Now(), "today")
}

func (h *Handler) tomorrow(user *database.User) {
	tomorrow := time.Now().In(ukLocation).AddDate(0, 0, 1)
	h.sendTimetable(user, tomorrow, tomorrow, "tomorrow")
}

func (h *Handler) week(user *database.User) {
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

	h.sendTimetable(user, weekStart, weekEnd, period)
}

func (h *Handler) sendTimetable(user *database.User, startDate, endDate time.Time, period string) {
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

func (h *Handler) settings(user *database.User) {
	h.sendMessage(user.ChatID, fmt.Sprintf("Your settings:\nDaily notification time: %v\nWeekly notification day and time: %v\n Reminder offset: %v minutes", user.DailyTime, user.WeeklyTime, user.ReminderOffset))
	if user.WebCalURL == "" {
		h.sendMessage(user.ChatID, "Your Calendar link is not set. Use /set_calendar to set it.")
	}
}

func (h *Handler) handleAddFriend(user *database.User, text string) {
	if !strings.HasPrefix(text, "@") || len(text) < 2 {
		h.sendMessage(user.ChatID, "Invalid username format. Please provide a valid Telegram username (e.g., @username).")
		return
	}

	friendUsername := strings.TrimPrefix(text, "@")
	friend, err := h.db.GetUserByUsername(friendUsername)
	if err != nil {
		h.sendMessage(user.ChatID, "Error accessing the database. Please try again later.")
		return
	}
	if friend == nil {
		h.sendMessage(user.ChatID, "User not found. Please ensure the user has started the bot and their username is correct.")
		return
	}
	if friend.ChatID == user.ChatID {
		h.sendMessage(user.ChatID, "You cannot add yourself as a friend.")
		return
	}

	areFriends, err := h.db.AreFriends(user.ChatID, friend.ChatID)
	if err != nil {
		h.sendMessage(user.ChatID, "Error checking friendship status.")
		return
	}
	if areFriends {
		h.sendMessage(user.ChatID, "You are already friends with this user.")
		return
	}

	requestExists, err := h.db.FriendRequestExists(user.ChatID, friend.ChatID)
	if err != nil {
		h.sendMessage(user.ChatID, "Error checking existing friend requests.")
		return
	}
	if requestExists {
		h.sendMessage(user.ChatID, "You have already sent a friend request to this user.")
		return
	}

	err = h.db.AddFriendRequest(user.ChatID, friend.ChatID)
	if err != nil {
		h.sendMessage(user.ChatID, "Error sending friend request.")
		return
	}

	h.sendMessage(user.ChatID, "Request sent. Your friend now needs to add you with the /accept\\_friend command.")
	h.clearUserState(user.ChatID)

	h.sendMessage(friend.ChatID, fmt.Sprintf("@%s has sent you a friend request. Use /accept\\_friend to accept.", user.Username))
}

func (h *Handler) handleAcceptFriend(user *database.User) {
	requestorIDs, err := h.db.GetPendingFriendRequests(user.ChatID)
	if err != nil {
		h.sendMessage(user.ChatID, "Error fetching friend requests.")
		return
	}

	if len(requestorIDs) == 0 {
		h.sendMessage(user.ChatID, "You have no pending friend requests.")
		h.clearUserState(user.ChatID)
		return
	}

	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, requestorID := range requestorIDs {
		requestor, err := h.db.GetUser(requestorID)
		if err != nil || requestor == nil {
			continue
		}
		requestorUsername := requestor.Username

		callbackData := fmt.Sprintf("accept_%d", requestorID)
		button := tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("@%s", requestorUsername), callbackData)
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(button))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msg := tgbotapi.NewMessage(user.ChatID, "Pending Friend Requests:")
	msg.ReplyMarkup = keyboard

	if _, err := h.api.Send(msg); err != nil {
		log.Printf("Error sending friend requests: %v", err)
	}

	h.clearUserState(user.ChatID)
}

func (h *Handler) HandleCallbackQuery(callback *tgbotapi.CallbackQuery) {
	data := callback.Data
	chatID := callback.Message.Chat.ID

	ack := tgbotapi.NewCallback(callback.ID, "")
	if _, err := h.api.Request(ack); err != nil {
		log.Printf("Error acknowledging callback: %v", err)
	}

	if strings.HasPrefix(data, "accept_") {
		parts := strings.Split(data, "_")
		if len(parts) != 2 {
			h.sendMessage(chatID, "Invalid callback data.")
			return
		}
		var requestorID int64
		_, err := fmt.Sscanf(parts[1], "%d", &requestorID)
		if err != nil {
			h.sendMessage(chatID, "Invalid requestor ID.")
			return
		}

		requestor, err := h.db.GetUser(requestorID)
		if err != nil || requestor == nil {
			h.sendMessage(chatID, "Requestor user not found.")
			return
		}

		currentUser, err := h.db.GetUser(chatID)
		if err != nil || currentUser == nil {
			h.sendMessage(chatID, "Error fetching your data.")
			return
		}

		err = h.db.AcceptFriendRequest(requestorID, chatID)
		if err != nil {
			h.sendMessage(chatID, "Error accepting friend request.")
			return
		}

		h.sendMessage(currentUser.ChatID, fmt.Sprintf("You are now friends with @%s!", requestor.Username))
		h.sendMessage(requestor.ChatID, fmt.Sprintf("@%s has accepted your friend request!", currentUser.Username))
	}
}

func (h *Handler) handleSetCalendar(user *database.User, text string) {
	if !strings.HasPrefix(strings.ToLower(text), "webcal://") {
		h.sendMessage(user.ChatID, "Calendar link must start with webcal://")
		return
	}
	user.WebCalURL = text
	h.db.SaveUser(user)
	h.scheduler.ScheduleUser(user.ChatID)
	h.sendMessage(user.ChatID, "Calendar link saved.")
	h.clearUserState(user.ChatID)
}

func (h *Handler) handleSetDailyTime(user *database.User, text string) {
	if !utils.IsValidTime(text) {
		h.sendMessage(user.ChatID, "Invalid format. Use HH:MM format.")
		return
	}
	user.DailyTime = text
	h.db.SaveUser(user)
	h.scheduler.ScheduleUser(user.ChatID)
	h.sendMessage(user.ChatID, "Daily notification time saved.")
	h.clearUserState(user.ChatID)
}

func (h *Handler) handleSetWeeklyTime(user *database.User, text string) {
	parts := strings.SplitN(text, " ", 2)
	if len(parts) != 2 || !utils.IsValidDay(parts[0]) || !utils.IsValidTime(parts[1]) {
		h.sendMessage(user.ChatID, "Invalid format. Use DAY HH:MM.")
		return
	}
	user.WeeklyTime = text
	h.db.SaveUser(user)
	h.scheduler.ScheduleUser(user.ChatID)
	h.sendMessage(user.ChatID, "Weekly notification time saved.")
	h.clearUserState(user.ChatID)
}

func (h *Handler) handleSetReminderOffset(user *database.User, text string) {
	if !utils.IsValidOffset(text) {
		h.sendMessage(user.ChatID, "Invalid format. Use MM format.")
		return
	}
	user.ReminderOffset = text
	h.db.SaveUser(user)
	h.scheduler.ScheduleUser(user.ChatID)
	h.sendMessage(user.ChatID, "Reminder offset saved.")
	h.clearUserState(user.ChatID)
}
