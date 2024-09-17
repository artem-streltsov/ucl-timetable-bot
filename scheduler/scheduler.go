package scheduler

import (
	"database/sql"
	"log"
	"time"

	ical "github.com/arran4/golang-ical"
	"github.com/artem-streltsov/ucl-timetable-bot/common"
	"github.com/artem-streltsov/ucl-timetable-bot/notifications"
	"github.com/artem-streltsov/ucl-timetable-bot/utils"
)

func GetLastDailySentTime(db *sql.DB, chatID int64) (time.Time, error) {
	var lastDailySent sql.NullString
	err := db.QueryRow("SELECT lastDailySent FROM users WHERE chatID = ?", chatID).Scan(&lastDailySent)
	if err != nil {
		return time.Time{}, err
	}
	if lastDailySent.Valid {
		return utils.ParseTimeUTC(lastDailySent.String)
	}
	return time.Time{}, nil
}

func GetLastWeeklySentTime(db *sql.DB, chatID int64) (time.Time, error) {
	var lastWeeklySent sql.NullString
	err := db.QueryRow("SELECT lastWeeklySent FROM users WHERE chatID = ?", chatID).Scan(&lastWeeklySent)
	if err != nil {
		return time.Time{}, err
	}
	if lastWeeklySent.Valid {
		return utils.ParseTimeUTC(lastWeeklySent.String)
	}
	return time.Time{}, nil
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

func ScheduleDailySummary(bot common.BotAPI, db *sql.DB, chatID int64, webcalURL string) {
	lastDailySentTime, err := GetLastDailySentTime(db, chatID)
	if err != nil {
		log.Println("Error fetching lastDailySent:", err)
		return
	}

	now := utils.CurrentTimeUTC()
	if !ShouldSendDailySummary(lastDailySentTime, now) {
		log.Printf("Daily summary already sent today for chatID: %d", chatID)
		return
	}

	nextDaily := GetNextDailySummaryTime(now)
	durationUntilNextDaily := nextDaily.Sub(now)

	time.AfterFunc(durationUntilNextDaily, func() {
		notifications.SendDailySummary(bot, chatID, webcalURL)
		_, err := db.Exec("UPDATE users SET lastDailySent = ? WHERE chatID = ?", utils.CurrentTimeUTC().Format(time.RFC3339), chatID)
		if err != nil {
			log.Printf("Error updating lastDailySent: %v", err)
		}
		ScheduleDailySummary(bot, db, chatID, webcalURL)
	})
}

func ScheduleWeeklySummary(bot common.BotAPI, db *sql.DB, chatID int64, webcalURL string) {
	lastWeeklySentTime, err := GetLastWeeklySentTime(db, chatID)
	if err != nil {
		log.Println("Error fetching lastWeeklySent:", err)
		return
	}

	now := utils.CurrentTimeUTC()
	nextSunday := GetNextSunday(now)
	if !ShouldSendWeeklySummary(lastWeeklySentTime, nextSunday) {
		log.Printf("Weekly summary already sent this week for chatID: %d", chatID)
		return
	}

	durationUntilNextSunday := nextSunday.Sub(now)
	time.AfterFunc(durationUntilNextSunday, func() {
		notifications.SendWeeklySummary(bot, chatID, webcalURL)
		_, err := db.Exec("UPDATE users SET lastWeeklySent = ? WHERE chatID = ?", utils.CurrentTimeUTC().Format(time.RFC3339), chatID)
		if err != nil {
			log.Printf("Error updating lastWeeklySent: %v", err)
		}
		ScheduleWeeklySummary(bot, db, chatID, webcalURL)
	})
}

// TODO: only reschedule the last one
func RescheduleNotificationsOnStartup(bot common.BotAPI, db *sql.DB) error {
	rows, err := db.Query("SELECT chatID, webcalURL, lastDailySent, lastWeeklySent FROM users")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var chatID int64
		var webcalURL string
		var lastDailySent sql.NullString
		var lastWeeklySent sql.NullString

		err = rows.Scan(&chatID, &webcalURL, &lastDailySent, &lastWeeklySent)
		if err != nil {
			log.Printf("Error scanning user data: %v", err)
			continue
		}

		handleReschedule(bot, db, chatID, webcalURL, lastDailySent, lastWeeklySent)
	}

	return nil
}

func handleReschedule(bot common.BotAPI, db *sql.DB, chatID int64, webcalURL string, lastDailySent sql.NullString, lastWeeklySent sql.NullString) {
	now := utils.CurrentTimeUTC()

	if lastDailySent.Valid {
		lastDailySentTime, err := utils.ParseTimeUTC(lastDailySent.String)
		if err == nil && lastDailySentTime.Before(now.AddDate(0, 0, -1).Add(5*time.Minute)) {
			notifications.SendDailySummary(bot, chatID, webcalURL)
		}
	}

	if lastWeeklySent.Valid {
		lastWeeklySentTime, err := utils.ParseTimeUTC(lastWeeklySent.String)
		if err == nil && lastWeeklySentTime.Before(now.AddDate(0, 0, -7).Add(5*time.Minute)) {
			notifications.SendWeeklySummary(bot, chatID, webcalURL)
		}
	}

	ScheduleDailySummary(bot, db, chatID, webcalURL)
	ScheduleWeeklySummary(bot, db, chatID, webcalURL)
	log.Printf("Rescheduling chatID %v %v", chatID, webcalURL)
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
			log.Println("Error parsing start time for chatID:", chatID, err)
			continue
		}

		reminderTime := startTime.Add(-time.Minute * 30)
		if ShouldScheduleReminder(reminderTime) {
			durationUntilReminder := time.Until(reminderTime)
			time.AfterFunc(durationUntilReminder, func() {
				notifications.SendReminder(bot, chatID, lecture)
			})
		}
	}
}
