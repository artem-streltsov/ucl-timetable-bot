package handlers

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/artem-streltsov/ucl-timetable-bot/database"
	"github.com/artem-streltsov/ucl-timetable-bot/models"
	"github.com/artem-streltsov/ucl-timetable-bot/scheduler"
	"github.com/artem-streltsov/ucl-timetable-bot/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	defaultDailyTime      = "07:00"
	defaultWeeklyTime     = "SUN 18:00"
	defaultReminderOffset = "15"
	ukLocation, _         = time.LoadLocation("Europe/London")
)

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

func (h *Handler) registerUser(chatID int64, username string) (*models.User, error) {
	user, err := h.db.GetUser(chatID)
	if err != nil {
		h.sendMessage(chatID, fmt.Sprintf("Error: %v", err))
		return nil, err
	}
	if user == nil {
		user = &models.User{
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
		h.sendMessage(chatID, "Welcome! Use /set_calendar to set your Calendar link.")
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
	text = utils.EscapeUnderscores(text)
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
