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
	msg := bot.NewMessage(chatID, "Please provide your WebCal link to subscribe to your lecture timetable.")
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

func HandleWebCalLink(bot common.BotAPI, db *sql.DB, chatID int64, webcalURL string) {
	validWebCalURL, valid := ValidateWebCalLink(webcalURL)
	if !valid {
		msg := bot.NewMessage(chatID, "Invalid link! Please provide a valid WebCal link that starts with 'webcal://'.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending invalid link message: %v", err)
		}
		return
	}

	if err := database.InsertUser(db, chatID, validWebCalURL); err != nil {
		log.Printf("Error saving WebCal link: %v", err)
		msg := bot.NewMessage(chatID, "There was an error saving your WebCal link. Please try again.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending error message: %v", err)
		}
		return
	}

	msg := bot.NewMessage(chatID, "Thank you! You will start receiving daily and weekly updates for your lectures. Use /settings to configure your notification preferences.")
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending confirmation message: %v", err)
	}

	if err := SendNotifications(bot, db, chatID, validWebCalURL); err != nil {
		log.Printf("Error sending initial notifications: %v", err)
	}

	if err := ScheduleNotifications(bot, db, chatID, validWebCalURL); err != nil {
		log.Printf("Error scheduling notifications: %v", err)
	}
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

func ScheduleNotifications(bot common.BotAPI, db *sql.DB, chatID int64, webcalURL string) error {
	if err := scheduler.ScheduleDailySummary(bot, db, chatID, webcalURL); err != nil {
		return fmt.Errorf("error scheduling daily summary: %w", err)
	}

	if err := scheduler.ScheduleWeeklySummary(bot, db, chatID, webcalURL); err != nil {
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

	settingsMessage := fmt.Sprintf("Current Settings:\n"+
		"Daily Notification Time: %s\n"+
		"Weekly Notification Time: %s\n"+
		"Reminder Offset: %d minutes\n\n"+
		"To update your settings, use the following commands:\n"+
		"/set_daily_time HH:MM\n"+
		"/set_weekly_time DAY HH:MM (e.g., SUN 18:00)\n"+
		"/set_reminder_offset MINUTES",
		dailyNotificationTime, weeklyNotificationTime, reminderOffset)

	msg := bot.NewMessage(chatID, settingsMessage)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending settings message: %v", err)
	}
}

func HandleSetDailyTime(bot common.BotAPI, db *sql.DB, chatID int64, timeStr string) {
	if _, err := time.Parse("15:04", timeStr); err != nil {
		msg := bot.NewMessage(chatID, "Invalid time format. Please use HH:MM (24-hour format).")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending invalid time format message: %v", err)
		}
		return
	}

	if err := updateUserPreference(db, chatID, "dailyNotificationTime", timeStr); err != nil {
		log.Printf("Error updating daily notification time: %v", err)
		msg := bot.NewMessage(chatID, "An error occurred while updating your settings. Please try again later.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending error message: %v", err)
		}
		return
	}

	msg := bot.NewMessage(chatID, fmt.Sprintf("Daily notification time updated to %s", timeStr))
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending confirmation message: %v", err)
	}
}

func HandleSetWeeklyTime(bot common.BotAPI, db *sql.DB, chatID int64, dayAndTime string) {
	parts := strings.Split(dayAndTime, " ")
	if len(parts) != 2 {
		msg := bot.NewMessage(chatID, "Invalid format. Please use DAY HH:MM (e.g., SUN 18:00).")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending invalid format message: %v", err)
		}
		return
	}

	day := strings.ToUpper(parts[0])
	timeStr := parts[1]

	if !isValidDay(day) || !isValidTime(timeStr) {
		msg := bot.NewMessage(chatID, "Invalid day or time format. Please use a valid day (MON, TUE, WED, THU, FRI, SAT, SUN) and time in HH:MM format.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending invalid day or time format message: %v", err)
		}
		return
	}

	if err := updateUserPreference(db, chatID, "weeklyNotificationTime", fmt.Sprintf("%s %s", day, timeStr)); err != nil {
		log.Printf("Error updating weekly notification time: %v", err)
		msg := bot.NewMessage(chatID, "An error occurred while updating your settings. Please try again later.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending error message: %v", err)
		}
		return
	}

	msg := bot.NewMessage(chatID, fmt.Sprintf("Weekly notification time updated to %s %s", day, timeStr))
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending confirmation message: %v", err)
	}
}

func HandleSetReminderOffset(bot common.BotAPI, db *sql.DB, chatID int64, offsetStr string) {
	offset, err := time.ParseDuration(offsetStr + "m")
	if err != nil || offset < 0 {
		msg := bot.NewMessage(chatID, "Invalid offset. Please provide a positive number of minutes.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending invalid offset message: %v", err)
		}
		return
	}

	if err := updateUserPreference(db, chatID, "reminderOffset", int(offset.Minutes())); err != nil {
		log.Printf("Error updating reminder offset: %v", err)
		msg := bot.NewMessage(chatID, "An error occurred while updating your settings. Please try again later.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending error message: %v", err)
		}
		return
	}

	msg := bot.NewMessage(chatID, fmt.Sprintf("Reminder offset updated to %d minutes", int(offset.Minutes())))
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending confirmation message: %v", err)
	}
}

func updateUserPreference(db *sql.DB, chatID int64, field string, value interface{}) error {
	dailyTime, weeklyTime, reminderOffset, err := database.GetUserPreferences(db, chatID)
	if err != nil {
		return err
	}

	switch field {
	case "dailyNotificationTime":
		dailyTime = value.(string)
	case "weeklyNotificationTime":
		weeklyTime = value.(string)
	case "reminderOffset":
		reminderOffset = value.(int)
	}

	return database.UpdateUserPreferences(db, chatID, dailyTime, weeklyTime, reminderOffset)
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

func UpdateUserPreference(db *sql.DB, chatID int64, field string, dailyTime, weeklyTime string, reminderOffset int) error {
	if field == "all" {
		return database.UpdateUserPreferences(db, chatID, dailyTime, weeklyTime, reminderOffset)
	}

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
	default:
		return fmt.Errorf("invalid preference field: %s", field)
	}

	return database.UpdateUserPreferences(db, chatID, currentDailyTime, currentWeeklyTime, currentReminderOffset)
}
