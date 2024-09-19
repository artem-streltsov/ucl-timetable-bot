package scheduler

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
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

	nextDaily, err := GetNextNotificationTime(now, dailyNotificationTime, "")
	if err != nil {
		return fmt.Errorf("error calculating next daily notification time: %w", err)
	}
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

	nextWeekly, err := GetNextNotificationTime(now, "", weeklyNotificationTime)
	if err != nil {
		return fmt.Errorf("error calculating next weekly notification time: %w", err)
	}
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

func GetNextNotificationTime(now time.Time, dailyTime, weeklyTime string) (time.Time, error) {
	if dailyTime != "" {
		return getNextDailyTime(now, dailyTime)
	}
	return getNextWeeklyTime(now, weeklyTime)
}

func getNextDailyTime(now time.Time, dailyTime string) (time.Time, error) {
	timeParts := strings.Split(dailyTime, ":")
	if len(timeParts) != 2 {
		return time.Time{}, fmt.Errorf("invalid time format")
	}

	hour, err := strconv.Atoi(timeParts[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid hour: %v", err)
	}
	minute, err := strconv.Atoi(timeParts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid minute: %v", err)
	}

	nextDaily := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, time.UTC)
	if nextDaily.Before(now) {
		nextDaily = nextDaily.AddDate(0, 0, 1)
	}
	return nextDaily, nil
}

func getNextWeeklyTime(now time.Time, weeklyTime string) (time.Time, error) {
	if weeklyTime == "" {
		return time.Time{}, fmt.Errorf("weekly time is empty")
	}

	parts := strings.Split(weeklyTime, " ")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("invalid weekly time format")
	}

	dayStr, timeStr := parts[0], parts[1]
	timeParts := strings.Split(timeStr, ":")
	if len(timeParts) != 2 {
		return time.Time{}, fmt.Errorf("invalid time format in weekly time")
	}

	hour, err := strconv.Atoi(timeParts[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid hour in weekly time: %v", err)
	}
	minute, err := strconv.Atoi(timeParts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid minute in weekly time: %v", err)
	}

	targetWeekday := getWeekday(dayStr)
	daysUntilTarget := (int(targetWeekday) - int(now.Weekday()) + 7) % 7

	nextWeekly := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, time.UTC)
	nextWeekly = nextWeekly.AddDate(0, 0, daysUntilTarget)

	if nextWeekly.Before(now) {
		nextWeekly = nextWeekly.AddDate(0, 0, 7)
	}

	return nextWeekly, nil
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
    
	if !user.LastWeeklySent.IsZero() {
		nextWeekly, err := GetNextNotificationTime(user.LastWeeklySent, "", weeklyTime)
		if err != nil {
			return fmt.Errorf("error getting next weekly notification time: %w", err)
		}
		if nextWeekly.Before(now) {
			if err := notifications.SendWeeklySummary(bot, db, user.ChatID, user.WebcalURL); err != nil {
				log.Printf("Error sending missed weekly summary for chatID %d: %v", user.ChatID, err)
			} else {
				if err := database.UpdateLastWeeklySent(db, user.ChatID, now); err != nil {
					log.Printf("Error updating lastWeeklySent for chatID %d: %v", user.ChatID, err)
				}
			}
		}
	}

	if !user.LastDailySent.IsZero() {
		nextDaily, err := GetNextNotificationTime(user.LastDailySent, dailyTime, "")
		if err != nil {
			return fmt.Errorf("error getting next daily notification time: %w", err)
		}
		if nextDaily.Before(now) {
			if err := notifications.SendDailySummary(bot, db, user.ChatID, user.WebcalURL); err != nil {
				log.Printf("Error sending missed daily summary for chatID %d: %v", user.ChatID, err)
			} else {
				if err := database.UpdateLastDailySent(db, user.ChatID, now); err != nil {
					log.Printf("Error updating lastDailySent for chatID %d: %v", user.ChatID, err)
				}
			}
		}
	}

	if err := ScheduleDailySummary(bot, db, user.ChatID, user.WebcalURL); err != nil {
		log.Printf("Error scheduling daily summary for chatID %d: %v", user.ChatID, err)
	}
	if err := ScheduleWeeklySummary(bot, db, user.ChatID, user.WebcalURL); err != nil {
		log.Printf("Error scheduling weekly summary for chatID %d: %v", user.ChatID, err)
	}

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
