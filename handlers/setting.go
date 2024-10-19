package handlers

import (
	"fmt"
	"strings"

	"github.com/artem-streltsov/ucl-timetable-bot/models"
	"github.com/artem-streltsov/ucl-timetable-bot/utils"
)

func (h *Handler) settings(user *models.User) {
	h.sendMessage(user.ChatID, fmt.Sprintf("Your settings:\nDaily notification time: %v\nWeekly notification day and time: %v\nReminder offset: %v minutes", user.DailyTime, user.WeeklyTime, user.ReminderOffset))
	if user.WebCalURL == "" {
		h.sendMessage(user.ChatID, "Your Calendar link is not set. Use /set_calendar to set it.")
	}
}

func (h *Handler) handleSetCalendar(user *models.User, text string) {
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

func (h *Handler) handleSetDailyTime(user *models.User, text string) {
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

func (h *Handler) handleSetWeeklyTime(user *models.User, text string) {
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

func (h *Handler) handleSetReminderOffset(user *models.User, text string) {
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
