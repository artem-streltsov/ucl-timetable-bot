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

	msg := bot.NewMessage(chatID, "Thank you! You will start receiving daily and weekly updates for your lectures.")
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending confirmation message: %v", err)
	}

	if err := SendNotifications(bot, chatID, validWebCalURL); err != nil {
		log.Printf("Error sending initial notifications: %v", err)
	}

	if err := ScheduleNotifications(bot, db, chatID, validWebCalURL); err != nil {
		log.Printf("Error scheduling notifications: %v", err)
	}
}

func SendNotifications(bot common.BotAPI, chatID int64, webcalURL string) error {
	if err := notifications.SendWeeklySummary(bot, chatID, webcalURL); err != nil {
		return fmt.Errorf("error sending weekly summary: %w", err)
	}

	currentWeekday := time.Now().UTC().Weekday()
	if currentWeekday != time.Sunday && currentWeekday != time.Saturday {
		if err := notifications.SendDailySummary(bot, chatID, webcalURL); err != nil {
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

	if err := notifications.SendDailySummary(bot, chatID, webcalURL); err != nil {
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

	if err := notifications.SendWeeklySummary(bot, chatID, webcalURL); err != nil {
		log.Printf("Error sending weekly summary: %v", err)
		msg := bot.NewMessage(chatID, "An error occurred while fetching this week's lectures. Please try again later.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending error message: %v", err)
		}
	}
}
