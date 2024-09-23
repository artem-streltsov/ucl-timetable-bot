package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/artem-streltsov/ucl-timetable-bot/common"
	"github.com/artem-streltsov/ucl-timetable-bot/database"
	"github.com/artem-streltsov/ucl-timetable-bot/notifications"
	"github.com/artem-streltsov/ucl-timetable-bot/scheduler"
)

func HandleStartCommand(bot common.BotAPI, chatID int64) {
	msg := bot.NewMessage(chatID, "Please provide your WebCal link to subscribe to your lecture timetable. The link should start with `webcal://`. It can be found in Portico -> My Studies -> Timetable -> Add to Calendar -> Copy the WebCal link")
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending start message: %v", err)
	}
}

func ValidateWebCalLink(webcalURL string) (string, bool) {
	webcalURL = strings.ToLower(webcalURL)
	if !strings.HasPrefix(webcalURL, "webcal://") {
		return "", false
	}
	return strings.Replace(webcalURL, "webcal://", "https://", 1), true
}

func HandleSetWebCalPrompt(bot common.BotAPI, chatID int64) {
	msg := bot.NewMessage(chatID, "Please provide your WebCal link to subscribe to your lecture timetable. The link should start with `webcal://`. It can be found in Portico -> My Studies -> Timetable -> Add to Calendar -> Copy the WebCal link")
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending WebCal prompt: %v", err)
	}
}

func HandleWebCalLink(bot common.BotAPI, db *sql.DB, chatID int64, webcalURL string) bool {
	validWebCalURL, valid := ValidateWebCalLink(webcalURL)
	if !valid {
		msg := bot.NewMessage(chatID, "Invalid link! Please provide a valid WebCal link that starts with 'webcal://'.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending invalid link message: %v", err)
		}
		return false
	}

	if err := database.InsertUser(db, chatID, validWebCalURL); err != nil {
		log.Printf("Error saving WebCal link: %v", err)
		msg := bot.NewMessage(chatID, "There was an error saving your WebCal link. Please try again.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending error message: %v", err)
		}
		return false
	}

	msg := bot.NewMessage(chatID, "Thank you! You will start receiving daily and weekly updates for your lectures.\nUse /settings to configure your notification preferences.")
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending confirmation message: %v", err)
	}

	if err := SendNotifications(bot, db, chatID, validWebCalURL); err != nil {
		log.Printf("Error sending initial notifications: %v", err)
	}

	if err := ScheduleNotifications(bot, db, chatID); err != nil {
		log.Printf("Error scheduling notifications: %v", err)
	}

	return true
}

func SendNotifications(bot common.BotAPI, db *sql.DB, chatID int64, webcalURL string) error {
	if err := notifications.SendWeeklySummary(bot, db, chatID, webcalURL); err != nil {
		return fmt.Errorf("error sending weekly summary: %w", err)
	}

	currentWeekday := time.Now().UTC().Weekday()
	if currentWeekday != time.Sunday && currentWeekday != time.Saturday {
		if err := notifications.SendDailySummary(bot, db, chatID, webcalURL); err != nil {
			return fmt.Errorf("error sending daily summary: %w", err)
		}
	}

	return nil
}

func ScheduleNotifications(bot common.BotAPI, db *sql.DB, chatID int64) error {
	if err := scheduler.ScheduleDailySummary(bot, db, chatID); err != nil {
		return fmt.Errorf("error scheduling daily summary: %w", err)
	}

	if err := scheduler.ScheduleWeeklySummary(bot, db, chatID); err != nil {
		return fmt.Errorf("error scheduling weekly summary: %w", err)
	}

	return nil
}

func HandleTodayCommand(bot common.BotAPI, db *sql.DB, chatID int64) {
	webcalURL, err := database.GetWebCalURL(db, chatID)
	if err != nil {
		log.Printf("Error fetching WebCal URL: %v", err)
		msg := bot.NewMessage(chatID, "An error occurred while fetching your timetable. Please try again later.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending error message: %v", err)
		}
		return
	}

	if webcalURL == "" {
		msg := bot.NewMessage(chatID, "You haven't provided a WebCal link yet. Please use the /start command to set up your timetable.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending WebCal not found message: %v", err)
		}
		return
	}

	if err := notifications.SendDailySummary(bot, db, chatID, webcalURL); err != nil {
		log.Printf("Error sending daily summary: %v", err)
		msg := bot.NewMessage(chatID, "An error occurred while fetching today's lectures. Please try again later.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending error message: %v", err)
		}
	}
}

func HandleWeekCommand(bot common.BotAPI, db *sql.DB, chatID int64) {
	webcalURL, err := database.GetWebCalURL(db, chatID)
	if err != nil {
		log.Printf("Error fetching WebCal URL: %v", err)
		msg := bot.NewMessage(chatID, "An error occurred while fetching your timetable. Please try again later.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending error message: %v", err)
		}
		return
	}

	if webcalURL == "" {
		msg := bot.NewMessage(chatID, "You haven't provided a WebCal link yet. Please use the /start command to set up your timetable.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending WebCal not found message: %v", err)
		}
		return
	}

	if err := notifications.SendWeeklySummary(bot, db, chatID, webcalURL); err != nil {
		log.Printf("Error sending weekly summary: %v", err)
		msg := bot.NewMessage(chatID, "An error occurred while fetching this week's lectures. Please try again later.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending error message: %v", err)
		}
	}
}

func HandleSettingsCommand(bot common.BotAPI, db *sql.DB, chatID int64) {
	dailyNotificationTime, weeklyNotificationTime, reminderOffset, err := database.GetUserPreferences(db, chatID)
	if err != nil {
		log.Printf("Error fetching user preferences: %v", err)
		msg := bot.NewMessage(chatID, "An error occurred while fetching your settings. Please try again later.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending error message: %v", err)
		}
		return
	}

	settingsMessage := fmt.Sprintf("Current Settings (all times are in UK time):\n"+
		"Daily Notification Time: %s\n"+
		"Weekly Notification Time: %s\n"+
		"Reminder Offset: %d minutes\n\n"+
		"To update your settings, use the following commands:\n"+
		"/set_daily_time\n"+
		"/set_weekly_time\n"+
		"/set_reminder_offset",
		dailyNotificationTime, weeklyNotificationTime, reminderOffset)

	msg := bot.NewMessage(chatID, settingsMessage)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending settings message: %v", err)
	}
}

func HandleSetDailyTimePrompt(bot common.BotAPI, chatID int64) {
	msg := bot.NewMessage(chatID, "Please enter the time for daily notifications in HH:MM format (24-hour). All times are in UK time.")
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending daily time prompt: %v", err)
	}
}

func HandleSetWeeklyTimePrompt(bot common.BotAPI, chatID int64) {
	msg := bot.NewMessage(chatID, "Please enter the day and time for weekly notifications in the format DAY HH:MM (e.g., SUN 18:00). All times are in UK time.")
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending weekly time prompt: %v", err)
	}
}

func HandleSetReminderOffsetPrompt(bot common.BotAPI, chatID int64) {
	msg := bot.NewMessage(chatID, "Please enter the reminder offset in minutes (e.g., 30 for 30 minutes before the lecture).")
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending reminder offset prompt: %v", err)
	}
}

func HandleSetDailyTime(bot common.BotAPI, db *sql.DB, chatID int64, timeStr string) bool {
	if _, err := time.Parse("15:04", timeStr); err != nil {
		msg := bot.NewMessage(chatID, "Invalid time format. Please use HH:MM (24-hour format).")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending invalid time format message: %v", err)
		}
		return false
	}

	if err := UpdateUserPreference(db, chatID, "dailyNotificationTime", timeStr, "", 0); err != nil {
		log.Printf("Error updating daily notification time: %v", err)
		msg := bot.NewMessage(chatID, "An error occurred while updating your settings. Please try again later.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending error message: %v", err)
		}
		return false
	}

	if err := scheduler.StopAndRescheduleNotifications(bot, db, chatID); err != nil {
		log.Printf("Error rescheduling notifications: %v", err)
		msg := bot.NewMessage(chatID, "An error occurred while rescheduling your notifications. Please try again later.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending error message: %v", err)
		}
		return false
	}

	msg := bot.NewMessage(chatID, fmt.Sprintf("Daily notification time updated to %s (UK time)", timeStr))
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending confirmation message: %v", err)
	}

	return true
}

func HandleSetWeeklyTime(bot common.BotAPI, db *sql.DB, chatID int64, dayAndTime string) bool {
	parts := strings.Split(dayAndTime, " ")
	if len(parts) != 2 {
		msg := bot.NewMessage(chatID, "Invalid format. Please use DAY HH:MM (e.g., SUN 18:00).")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending invalid format message: %v", err)
		}
		return false
	}

	day := strings.ToUpper(parts[0])
	timeStr := parts[1]

	if !isValidDay(day) || !isValidTime(timeStr) {
		msg := bot.NewMessage(chatID, "Invalid day or time format. Please use a valid day (MON, TUE, WED, THU, FRI, SAT, SUN) and time in HH:MM format.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending invalid day or time format message: %v", err)
		}
		return false
	}

	if err := UpdateUserPreference(db, chatID, "weeklyNotificationTime", "", fmt.Sprintf("%s %s", day, timeStr), 0); err != nil {
		log.Printf("Error updating weekly notification time: %v", err)
		msg := bot.NewMessage(chatID, "An error occurred while updating your settings. Please try again later.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending error message: %v", err)
		}
		return false
	}

	if err := scheduler.StopAndRescheduleNotifications(bot, db, chatID); err != nil {
		log.Printf("Error rescheduling notifications: %v", err)
		msg := bot.NewMessage(chatID, "An error occurred while rescheduling your notifications. Please try again later.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending error message: %v", err)
		}
		return false
	}

	msg := bot.NewMessage(chatID, fmt.Sprintf("Weekly notification time updated to %s %s (UK time)", day, timeStr))
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending confirmation message: %v", err)
	}

	return true
}

func HandleSetReminderOffset(bot common.BotAPI, db *sql.DB, chatID int64, offsetStr string) bool {
	offset, err := time.ParseDuration(offsetStr + "m")
	if err != nil || offset < 0 {
		msg := bot.NewMessage(chatID, "Invalid offset. Please provide a positive number of minutes.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending invalid offset message: %v", err)
		}
		return false
	}

	if err := UpdateUserPreference(db, chatID, "reminderOffset", "", "", int(offset.Minutes())); err != nil {
		log.Printf("Error updating reminder offset: %v", err)
		msg := bot.NewMessage(chatID, "An error occurred while updating your settings. Please try again later.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending error message: %v", err)
		}
		return false
	}

	if err := scheduler.StopAndRescheduleNotifications(bot, db, chatID); err != nil {
		log.Printf("Error rescheduling notifications: %v", err)
		msg := bot.NewMessage(chatID, "An error occurred while rescheduling your notifications. Please try again later.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending error message: %v", err)
		}
		return false
	}

	msg := bot.NewMessage(chatID, fmt.Sprintf("Reminder offset updated to %d minutes", int(offset.Minutes())))
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending confirmation message: %v", err)
	}

	return true
}

func UpdateUserPreference(db *sql.DB, chatID int64, field string, dailyTime, weeklyTime string, reminderOffset int) error {
	currentDailyTime, currentWeeklyTime, currentReminderOffset, err := database.GetUserPreferences(db, chatID)
	if err != nil {
		return err
	}

	switch field {
	case "dailyNotificationTime":
		currentDailyTime = dailyTime
	case "weeklyNotificationTime":
		currentWeeklyTime = weeklyTime
	case "reminderOffset":
		currentReminderOffset = reminderOffset
	case "all":
		return database.UpdateUserPreferences(db, chatID, dailyTime, weeklyTime, reminderOffset)
	default:
		return fmt.Errorf("invalid preference field: %s", field)
	}

	return database.UpdateUserPreferences(db, chatID, currentDailyTime, currentWeeklyTime, currentReminderOffset)
}

func isValidDay(day string) bool {
	validDays := map[string]bool{
		"MON": true, "TUE": true, "WED": true, "THU": true,
		"FRI": true, "SAT": true, "SUN": true,
	}
	return validDays[day]
}

func isValidTime(timeStr string) bool {
	_, err := time.Parse("15:04", timeStr)
	return err == nil
}
