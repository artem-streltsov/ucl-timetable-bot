package scheduler

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	ical "github.com/arran4/golang-ical"
	"github.com/artem-streltsov/ucl-timetable-bot/common"
	"github.com/artem-streltsov/ucl-timetable-bot/database"
	"github.com/artem-streltsov/ucl-timetable-bot/notifications"
	"github.com/artem-streltsov/ucl-timetable-bot/utils"
)

func ScheduleDailySummary(bot common.BotAPI, db *sql.DB, chatID int64, webcalURL string) error {
	now := utils.CurrentTimeUTC()
	dailyNotificationTime, _, _, err := database.GetUserPreferences(db, chatID)
	if err != nil {
		return fmt.Errorf("error getting user preferences: %w", err)
	}

	nextDaily := GetNextNotificationTime(now, dailyNotificationTime, "")
	durationUntilNextDaily := nextDaily.Sub(now)

	time.AfterFunc(durationUntilNextDaily, func() {
		if err := notifications.SendDailySummary(bot, db, chatID, webcalURL); err != nil {
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
	now := utils.CurrentTimeUTC()
	_, weeklyNotificationTime, _, err := database.GetUserPreferences(db, chatID)
	if err != nil {
		return fmt.Errorf("error getting user preferences: %w", err)
	}

	nextWeekly := GetNextNotificationTime(now, "", weeklyNotificationTime)
	durationUntilNextWeekly := nextWeekly.Sub(now)

	time.AfterFunc(durationUntilNextWeekly, func() {
		if err := notifications.SendWeeklySummary(bot, db, chatID, webcalURL); err != nil {
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

func GetNextNotificationTime(now time.Time, dailyTime, weeklyTime string) time.Time {
	if dailyTime != "" {
		return getNextDailyTime(now, dailyTime)
	}
	return getNextWeeklyTime(now, weeklyTime)
}

func getNextDailyTime(now time.Time, dailyTime string) time.Time {
	timeParts := strings.Split(dailyTime, ":")
	hour, _ := time.Parse("15", timeParts[0])
	minute, _ := time.Parse("04", timeParts[1])

	nextDaily := time.Date(now.Year(), now.Month(), now.Day(), hour.Hour(), minute.Minute(), 0, 0, time.UTC)
	if nextDaily.Before(now) {
		nextDaily = nextDaily.AddDate(0, 0, 1)
	}
	return nextDaily
}

func getNextWeeklyTime(now time.Time, weeklyTime string) time.Time {
	parts := strings.Split(weeklyTime, " ")
	dayStr, timeStr := parts[0], parts[1]
	timeParts := strings.Split(timeStr, ":")
	hour, _ := time.Parse("15", timeParts[0])
	minute, _ := time.Parse("04", timeParts[1])

	targetWeekday := getWeekday(dayStr)
	daysUntilTarget := (int(targetWeekday) - int(now.Weekday()) + 7) % 7

	nextWeekly := time.Date(now.Year(), now.Month(), now.Day(), hour.Hour(), minute.Minute(), 0, 0, time.UTC)
	nextWeekly = nextWeekly.AddDate(0, 0, daysUntilTarget)

	if nextWeekly.Before(now) {
		nextWeekly = nextWeekly.AddDate(0, 0, 7)
	}

	return nextWeekly
}

func getWeekday(day string) time.Weekday {
	switch day {
	case "MON":
		return time.Monday
	case "TUE":
		return time.Tuesday
	case "WED":
		return time.Wednesday
	case "THU":
		return time.Thursday
	case "FRI":
		return time.Friday
	case "SAT":
		return time.Saturday
	case "SUN":
		return time.Sunday
	default:
		return time.Sunday
	}
}

func RescheduleNotificationsOnStartup(bot common.BotAPI, db *sql.DB) error {
	users, err := database.GetAllUsers(db)
	if err != nil {
		return fmt.Errorf("error fetching users: %w", err)
	}

	now := utils.CurrentTimeUTC()

	for _, user := range users {
		if err := handleReschedule(bot, db, user, now); err != nil {
			log.Printf("Error rescheduling notifications for chatID %d: %v", user.ChatID, err)
		}
	}

	return nil
}

func handleReschedule(bot common.BotAPI, db *sql.DB, user database.User, now time.Time) error {
	dailyTime, weeklyTime, _, err := database.GetUserPreferences(db, user.ChatID)
	if err != nil {
		return fmt.Errorf("error getting user preferences: %w", err)
	}

	nextWeekly := GetNextNotificationTime(user.LastWeeklySent, "", weeklyTime)
	if nextWeekly.Before(now) {
		if err := notifications.SendWeeklySummary(bot, db, user.ChatID, user.WebcalURL); err != nil {
			log.Printf("Error sending missed weekly summary for chatID %d: %v", user.ChatID, err)
		} else {
			if err := database.UpdateLastWeeklySent(db, user.ChatID, now); err != nil {
				log.Printf("Error updating lastWeeklySent for chatID %d: %v", user.ChatID, err)
			}
		}
	}

	nextDaily := GetNextNotificationTime(user.LastDailySent, dailyTime, "")
	if nextDaily.Before(now) {
		if err := notifications.SendDailySummary(bot, db, user.ChatID, user.WebcalURL); err != nil {
			log.Printf("Error sending missed daily summary for chatID %d: %v", user.ChatID, err)
		} else {
			if err := database.UpdateLastDailySent(db, user.ChatID, now); err != nil {
				log.Printf("Error updating lastDailySent for chatID %d: %v", user.ChatID, err)
			}
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

func ParseLectureStartTime(lecture *ical.VEvent) (time.Time, error) {
	return time.Parse("20060102T150405Z", lecture.GetProperty("DTSTART").Value)
}

func ShouldScheduleReminder(reminderTime time.Time) bool {
	return time.Until(reminderTime) > 0
}

func ScheduleLectureReminders(bot common.BotAPI, db *sql.DB, chatID int64, lectures []*ical.VEvent) {
	_, _, reminderOffset, err := database.GetUserPreferences(db, chatID)
	if err != nil {
		log.Printf("Error getting user preferences for chatID %d: %v", chatID, err)
		return
	}

	for _, lecture := range lectures {
		startTime, err := ParseLectureStartTime(lecture)
		if err != nil {
			log.Printf("Error parsing start time for chatID %d: %v", chatID, err)
			continue
		}

		reminderTime := startTime.Add(-time.Duration(reminderOffset) * time.Minute)
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
