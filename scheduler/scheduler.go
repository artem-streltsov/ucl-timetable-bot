package scheduler

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	ical "github.com/arran4/golang-ical"
	"github.com/artem-streltsov/ucl-timetable-bot/common"
	"github.com/artem-streltsov/ucl-timetable-bot/database"
	"github.com/artem-streltsov/ucl-timetable-bot/notifications"
	"github.com/artem-streltsov/ucl-timetable-bot/utils"
)

func ScheduleDailySummary(bot common.BotAPI, db *sql.DB, chatID int64, webcalURL string) error {
	lastDailySentTime, err := database.GetLastDailySentTime(db, chatID)
	if err != nil {
		return fmt.Errorf("error fetching lastDailySent: %w", err)
	}

	now := utils.CurrentTimeUTC()
	if !ShouldSendDailySummary(lastDailySentTime, now) {
		log.Printf("Daily summary already sent today for chatID: %d", chatID)
		return nil
	}

	nextDaily := GetNextDailySummaryTime(now)
	durationUntilNextDaily := nextDaily.Sub(now)

	time.AfterFunc(durationUntilNextDaily, func() {
		if err := notifications.SendDailySummary(bot, chatID, webcalURL); err != nil {
			log.Printf("Error sending daily summary: %v", err)
		}
		if err := database.UpdateLastDailySent(db, chatID, utils.CurrentTimeUTC()); err != nil {
			log.Printf("Error updating lastDailySent: %v", err)
		}
		if err := ScheduleDailySummary(bot, db, chatID, webcalURL); err != nil {
			log.Printf("Error rescheduling daily summary: %v", err)
		}
	})

	return nil
}

func ScheduleWeeklySummary(bot common.BotAPI, db *sql.DB, chatID int64, webcalURL string) error {
	lastWeeklySentTime, err := database.GetLastWeeklySentTime(db, chatID)
	if err != nil {
		return fmt.Errorf("error fetching lastWeeklySent: %w", err)
	}

	now := utils.CurrentTimeUTC()
	nextSunday := GetNextSunday(now)
	if !ShouldSendWeeklySummary(lastWeeklySentTime, nextSunday) {
		log.Printf("Weekly summary already sent this week for chatID: %d", chatID)
		return nil
	}

	durationUntilNextSunday := nextSunday.Sub(now)
	time.AfterFunc(durationUntilNextSunday, func() {
		if err := notifications.SendWeeklySummary(bot, chatID, webcalURL); err != nil {
			log.Printf("Error sending weekly summary: %v", err)
		}
		if err := database.UpdateLastWeeklySent(db, chatID, utils.CurrentTimeUTC()); err != nil {
			log.Printf("Error updating lastWeeklySent: %v", err)
		}
		if err := ScheduleWeeklySummary(bot, db, chatID, webcalURL); err != nil {
			log.Printf("Error rescheduling weekly summary: %v", err)
		}
	})

	return nil
}

func RescheduleNotificationsOnStartup(bot common.BotAPI, db *sql.DB) error {
	users, err := database.GetAllUsers(db)
	if err != nil {
		return fmt.Errorf("error fetching users: %w", err)
	}

	for _, user := range users {
		if err := handleReschedule(bot, db, user); err != nil {
			log.Printf("Error rescheduling notifications for chatID %d: %v", user.ChatID, err)
		}
	}

	return nil
}

func handleReschedule(bot common.BotAPI, db *sql.DB, user database.User) error {
	now := utils.CurrentTimeUTC()

	if user.LastWeeklySent.Before(now.AddDate(0, 0, -7).Add(5 * time.Minute)) {
		if err := notifications.SendWeeklySummary(bot, user.ChatID, user.WebcalURL); err != nil {
			return fmt.Errorf("Error sending weekly summary for chatID %d: %v", user.ChatID, err)
		}
		if err := database.UpdateLastDailySent(db, user.ChatID, utils.CurrentTimeUTC()); err != nil {
			return fmt.Errorf("Error updating lastDailySent: %v", err)
		}
	}

	if user.LastDailySent.Before(now.AddDate(0, 0, -1).Add(5 * time.Minute)) {
		if err := notifications.SendDailySummary(bot, user.ChatID, user.WebcalURL); err != nil {
			return fmt.Errorf("Error sending daily summary for chatID %d: %v", user.ChatID, err)
		}
		if err := database.UpdateLastWeeklySent(db, user.ChatID, utils.CurrentTimeUTC()); err != nil {
			return fmt.Errorf("Error updating lastWeeklySent: %v", err)
		}
	}

	if err := ScheduleDailySummary(bot, db, user.ChatID, user.WebcalURL); err != nil {
		return fmt.Errorf("error scheduling daily summary: %w", err)
	}

	if err := ScheduleWeeklySummary(bot, db, user.ChatID, user.WebcalURL); err != nil {
		return fmt.Errorf("error scheduling weekly summary: %w", err)
	}

	log.Printf("Rescheduled notifications for chatID %d", user.ChatID)
	return nil
}

func GetNextDailySummaryTime(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, time.UTC).AddDate(0, 0, 1)
}

func GetNextSunday(now time.Time) time.Time {
	nextSunday := time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, time.UTC)
	for nextSunday.Weekday() != time.Sunday {
		nextSunday = nextSunday.Add(24 * time.Hour)
	}
	return nextSunday
}

func ShouldSendDailySummary(lastSentTime, now time.Time) bool {
	return lastSentTime.Before(now.AddDate(0, 0, -1).Add(5 * time.Minute))
}

func ShouldSendWeeklySummary(lastSentTime, nextSunday time.Time) bool {
	return lastSentTime.Before(nextSunday.AddDate(0, 0, -7).Add(5 * time.Minute))
}

func ParseLectureStartTime(lecture *ical.VEvent) (time.Time, error) {
	return time.Parse("20060102T150405Z", lecture.GetProperty("DTSTART").Value)
}

func ShouldScheduleReminder(reminderTime time.Time) bool {
	return time.Until(reminderTime) > 0
}

func ScheduleLectureReminders(bot common.BotAPI, chatID int64, lectures []*ical.VEvent) {
	for _, lecture := range lectures {
		startTime, err := ParseLectureStartTime(lecture)
		if err != nil {
			log.Printf("Error parsing start time for chatID %d: %v", chatID, err)
			continue
		}

		reminderTime := startTime.Add(-time.Minute * 30)
		if ShouldScheduleReminder(reminderTime) {
			durationUntilReminder := time.Until(reminderTime)
			time.AfterFunc(durationUntilReminder, func() {
				if err := notifications.SendReminder(bot, chatID, lecture); err != nil {
					log.Printf("Error sending reminder for chatID %d: %v", chatID, err)
				}
			})
		}
	}
}
