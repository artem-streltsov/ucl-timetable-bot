package handlers

import (
	"database/sql"
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
	bot.Send(msg)
}

func ValidateWebCalLink(webcalURL string) (string, bool) {
	webcalURL = strings.ToLower(webcalURL)

	if !strings.HasPrefix(webcalURL, "webcal://") {
		return "", false
	}

	webcalURL = strings.Replace(webcalURL, "webcal://", "https://", 1)
	return webcalURL, true
}

func SaveWebCalLink(db *sql.DB, chatID int64, webcalURL string) error {
	return database.InsertUser(db, chatID, webcalURL)
}

func SendNotifications(bot common.BotAPI, chatID int64, webcalURL string) {
	notifications.SendWeeklySummary(bot, chatID, webcalURL)

	currentWeekday := time.Now().UTC().Weekday()
	if currentWeekday != time.Sunday && currentWeekday != time.Saturday {
		notifications.SendDailySummary(bot, chatID, webcalURL)
	}
}

func ScheduleNotifications(bot common.BotAPI, db *sql.DB, chatID int64, webcalURL string) {
	scheduler.ScheduleDailySummary(bot, db, chatID, webcalURL)
	scheduler.ScheduleWeeklySummary(bot, db, chatID, webcalURL)
}

func HandleWebCalLink(bot common.BotAPI, db *sql.DB, chatID int64, webcalURL string) {
	validWebCalURL, valid := ValidateWebCalLink(webcalURL)
	if !valid {
		msg := bot.NewMessage(chatID, "Invalid link! Please provide a valid WebCal link that starts with 'webcal://'.")
		bot.Send(msg)
		return
	}

	err := SaveWebCalLink(db, chatID, validWebCalURL)
	if err != nil {
		log.Println("Error saving WebCal link:", err)
		msg := bot.NewMessage(chatID, "There was an error saving your WebCal link. Please try again.")
		bot.Send(msg)
		return
	}

	msg := bot.NewMessage(chatID, "Thank you! You will start receiving daily and weekly updates for your lectures.")
	bot.Send(msg)

	SendNotifications(bot, chatID, validWebCalURL)
	ScheduleNotifications(bot, db, chatID, validWebCalURL)
}

func HandleTodayCommand(bot common.BotAPI, db *sql.DB, chatID int64) {
    webCalURL, err := database.GetWebCalURL(db, chatID)
    if err != nil {

    }

    if webCalURL == "" {
		msg := bot.NewMessage(chatID, "Error finding your WebCal link.")
        bot.Send(msg)
        HandleStartCommand(bot, chatID)
        return
    }

    notifications.SendDailySummary(bot, chatID, webCalURL)
}

func HandleWeekCommand(bot common.BotAPI, db *sql.DB, chatID int64) {
    webCalURL, err := database.GetWebCalURL(db, chatID)
    if err != nil {

    }

    if webCalURL == "" {
		msg := bot.NewMessage(chatID, "Error finding your WebCal link.")
        bot.Send(msg)
        HandleStartCommand(bot, chatID)
        return
    }

    notifications.SendWeeklySummary(bot, chatID, webCalURL)
}
