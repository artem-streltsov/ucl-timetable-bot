package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	ical "github.com/arran4/golang-ical"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

const (
	reminderOffset = 15 * time.Minute
	dbFile         = "user_data.db"
)

func parseTimeUTC(timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, timeStr)
}

func currentTimeUTC() time.Time {
	return time.Now().UTC()
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if telegramToken == "" {
		log.Fatalf("Telegram bot token not found in environment variables")
	}

	db, err := initDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	bot, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	bot.Debug = false
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	rescheduleNotificationsOnStartup(bot, db)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID

		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				handleStartCommand(bot, chatID)
			}
		} else if update.Message.Text != "" {
			handleWebCalLink(bot, db, chatID, update.Message.Text)
		}
	}
}

func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, err
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		chatID INTEGER PRIMARY KEY,
		webcalURL TEXT,
		lastDailySent DATETIME,
		lastWeeklySent DATETIME
	);
	`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func insertUser(db *sql.DB, chatID int64, webcalURL string) error {
	upsertSQL := `
	INSERT INTO users (chatID, webcalURL) VALUES (?, ?)
	ON CONFLICT(chatID) DO UPDATE SET webcalURL=excluded.webcalURL;
	`
	_, err := db.Exec(upsertSQL, chatID, webcalURL)
	return err
}

func handleStartCommand(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Welcome! Please provide your WebCal link to subscribe to your lecture timetable.")
	bot.Send(msg)
}

func handleWebCalLink(bot *tgbotapi.BotAPI, db *sql.DB, chatID int64, webcalURL string) {
	webcalURL = strings.ToLower(webcalURL)

	if !strings.HasPrefix(webcalURL, "webcal://") {
		msg := tgbotapi.NewMessage(chatID, "Invalid link! Please provide a valid WebCal link that starts with 'webcal://'.")
		bot.Send(msg)
		return
	}

	webcalURL = strings.Replace(webcalURL, "webcal://", "https://", 1)

	err := insertUser(db, chatID, webcalURL)
	if err != nil {
		log.Println("Error saving WebCal link:", err)
		msg := tgbotapi.NewMessage(chatID, "There was an error saving your WebCal link. Please try again.")
		bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(chatID, "Thank you! You will start receiving daily and weekly updates for your lectures.")
	bot.Send(msg)

	sendWeeklySummary(bot, chatID, webcalURL)
	if time.Now().UTC().Weekday() != time.Sunday || time.Now().UTC().Weekday() != time.Saturday {
		sendDailySummary(bot, chatID, webcalURL)
	}

	scheduleDailySummary(bot, db, chatID, webcalURL)
	scheduleWeeklySummary(bot, db, chatID, webcalURL)
}

func scheduleDailySummary(bot *tgbotapi.BotAPI, db *sql.DB, chatID int64, webcalURL string) {
	var lastDailySentStr string
	err := db.QueryRow("SELECT lastDailySent FROM users WHERE chatID = ?", chatID).Scan(&lastDailySentStr)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error fetching lastDailySent: %v", err)
		return
	}

	lastDailySent, err := parseTimeUTC(lastDailySentStr)
	if err != nil {
		log.Printf("Error parsing lastDailySent time: %v", err)
		return
	}

	now := currentTimeUTC()

	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return
	}

	nextCheck := time.Date(now.Year(), now.Month(), now.Day(), 7, 0, 0, 0, time.UTC)
	if now.After(nextCheck) {
		nextCheck = nextCheck.Add(24 * time.Hour)
	}

	if lastDailySent.After(nextCheck.AddDate(0, 0, -1)) {
		log.Printf("Daily summary already sent today for chatID: %d", chatID)
		return
	}

	durationUntilNextCheck := nextCheck.Sub(now)
	time.AfterFunc(durationUntilNextCheck, func() {
		sendDailySummary(bot, chatID, webcalURL)
		_, err := db.Exec("UPDATE users SET lastDailySent = ? WHERE chatID = ?", currentTimeUTC().Format(time.RFC3339), chatID)
		if err != nil {
			log.Printf("Error updating lastDailySent: %v", err)
		}

		scheduleDailySummary(bot, db, chatID, webcalURL)
	})
}

func scheduleWeeklySummary(bot *tgbotapi.BotAPI, db *sql.DB, chatID int64, webcalURL string) {
	var lastWeeklySentStr string
	err := db.QueryRow("SELECT lastWeeklySent FROM users WHERE chatID = ?", chatID).Scan(&lastWeeklySentStr)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error fetching lastWeeklySent: %v", err)
		return
	}

	lastWeeklySent, err := parseTimeUTC(lastWeeklySentStr)
	if err != nil {
		log.Printf("Error parsing lastWeeklySent time: %v", err)
		return
	}

	now := currentTimeUTC()
	nextSunday := time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, time.UTC)
	for nextSunday.Weekday() != time.Sunday {
		nextSunday = nextSunday.Add(24 * time.Hour)
	}

	if lastWeeklySent.After(nextSunday.AddDate(0, 0, -7)) {
		log.Printf("Weekly summary already sent this week for chatID: %d", chatID)
		return
	}

	durationUntilNextSunday := nextSunday.Sub(now)
	time.AfterFunc(durationUntilNextSunday, func() {
		sendWeeklySummary(bot, chatID, webcalURL)
		_, err := db.Exec("UPDATE users SET lastWeeklySent = ? WHERE chatID = ?", currentTimeUTC().Format(time.RFC3339), chatID)
		if err != nil {
			log.Printf("Error updating lastWeeklySent: %v", err)
		}

		scheduleWeeklySummary(bot, db, chatID, webcalURL)
	})
}

func rescheduleNotificationsOnStartup(bot *tgbotapi.BotAPI, db *sql.DB) {
	rows, err := db.Query("SELECT chatID, webcalURL, lastDailySent, lastWeeklySent FROM users")
	if err != nil {
		log.Fatalf("Error fetching users from database: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var chatID int64
		var webcalURL string
		var lastDailySentStr, lastWeeklySentStr string
		var lastDailySent, lastWeeklySent time.Time

		err = rows.Scan(&chatID, &webcalURL, &lastDailySentStr, &lastWeeklySentStr)
		if err != nil {
			log.Printf("Error scanning user data: %v", err)
			continue
		}

		lastDailySent, err = parseTimeUTC(lastDailySentStr)
		if err != nil {
			log.Printf("Error parsing lastDailySent: %v", err)
			continue
		}

		lastWeeklySent, err = parseTimeUTC(lastWeeklySentStr)
		if err != nil {
			log.Printf("Error parsing lastWeeklySent: %v", err)
			continue
		}

		now := currentTimeUTC()

		if lastDailySent.IsZero() || lastDailySent.Before(now.AddDate(0, 0, -1)) {
			sendDailySummary(bot, chatID, webcalURL)
		}

		if lastWeeklySent.IsZero() || lastWeeklySent.Before(now.AddDate(0, 0, -7)) {
			sendWeeklySummary(bot, chatID, webcalURL)
		}

		scheduleDailySummary(bot, db, chatID, webcalURL)
		scheduleWeeklySummary(bot, db, chatID, webcalURL)
		log.Printf("Rescheduling chatID %v %v", chatID, webcalURL)
	}
}

func getDayLectures(calendar *ical.Calendar) []*ical.VEvent {
	today := currentTimeUTC().Format("20060102")
	lectures := []*ical.VEvent{}
	for _, event := range calendar.Events() {
		start := event.GetProperty("DTSTART").Value
		if strings.HasPrefix(start, today) {
			lectures = append(lectures, event)
		}
	}
	return lectures
}

func sendDailySummary(bot *tgbotapi.BotAPI, chatID int64, webcalURL string) {
	response, err := http.Get(webcalURL)
	if err != nil {
		log.Println("Error fetching calendar for chatID:", chatID, err)
		return
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println("Error reading calendar for chatID:", chatID, err)
		return
	}

	calendar, err := ical.ParseCalendar(strings.NewReader(string(body)))
	if err != nil {
		log.Println("Error parsing calendar for chatID:", chatID, err)
		return
	}

	lecturesThisDay := getDayLectures(calendar)
	if len(lecturesThisDay) == 0 {
		msg := tgbotapi.NewMessage(chatID, "No lectures scheduled for today.")
		bot.Send(msg)
		return
	}

	message := "Today's Lectures:\n"
	for _, lecture := range lecturesThisDay {
		message += formatEventDetails(lecture) + "\n"
	}

	msg := tgbotapi.NewMessage(chatID, message)
	bot.Send(msg)
}

func getWeekLectures(calendar *ical.Calendar) map[string][]*ical.VEvent {
	lectures := make(map[string][]*ical.VEvent)
	now := currentTimeUTC()
	offset := int(time.Monday - now.Weekday())
	if offset > 0 {
		offset = -6 // Go back to previous Monday if today is Sunday
	}

	monday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, offset)
	friday := monday.AddDate(0, 0, 4).Add(24 * time.Hour)

	daysOfWeek := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"}

	for _, event := range calendar.Events() {
		startTimeStr := event.GetProperty("DTSTART").Value
		startTime, err := time.Parse("20060102T150405Z", startTimeStr)
		if err != nil {
			continue
		}

		if startTime.After(monday) && startTime.Before(friday) {
			weekday := daysOfWeek[int(startTime.Weekday())-1]
			lectures[weekday] = append(lectures[weekday], event)
		}
	}

	return lectures
}

func sendWeeklySummary(bot *tgbotapi.BotAPI, chatID int64, webcalURL string) {
	response, err := http.Get(webcalURL)
	if err != nil {
		log.Println("Error fetching calendar for chatID:", chatID, err)
		return
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println("Error reading calendar for chatID:", chatID, err)
		return
	}

	calendar, err := ical.ParseCalendar(strings.NewReader(string(body)))
	if err != nil {
		log.Println("Error parsing calendar for chatID:", chatID, err)
		return
	}

	lecturesThisWeek := getWeekLectures(calendar)
	if len(lecturesThisWeek) == 0 {
		msg := tgbotapi.NewMessage(chatID, "No lectures scheduled for this week.")
		bot.Send(msg)
		return
	}

	message := "This Week's Lectures:\n"
	for day, lectures := range lecturesThisWeek {
		message += fmt.Sprintf("\n%s:\n", day)
		for _, lecture := range lectures {
			message += formatEventDetails(lecture) + "\n"
		}
	}

	msg := tgbotapi.NewMessage(chatID, message)
	bot.Send(msg)
}

func scheduleLectureReminders(bot *tgbotapi.BotAPI, chatID int64, lectures []*ical.VEvent) {
	for _, lecture := range lectures {
		startTime, err := time.Parse("20060102T150405Z", lecture.GetProperty("DTSTART").Value)
		if err != nil {
			log.Println("Error parsing start time for chatID:", chatID, err)
			continue
		}

		reminderTime := startTime.Add(-reminderOffset)
		durationUntilReminder := time.Until(reminderTime)

		if durationUntilReminder > 0 {
			time.AfterFunc(durationUntilReminder, func() {
				sendReminder(bot, chatID, lecture)
			})
		}
	}
}

func sendReminder(bot *tgbotapi.BotAPI, chatID int64, lecture *ical.VEvent) {
	message := fmt.Sprintf("Reminder: Your lecture is starting in %v minutes!\n", reminderOffset)
	message += formatEventDetails(lecture)
	msg := tgbotapi.NewMessage(chatID, message)
	bot.Send(msg)
}

func formatEventDetails(event *ical.VEvent) string {
	summary := event.GetProperty("SUMMARY").Value
	location := event.GetProperty("LOCATION").Value
	startTime := event.GetProperty("DTSTART").Value
	start, _ := time.Parse("20060102T150405Z", startTime)
	return fmt.Sprintf("- %s at %s, starting at %s", summary, location, start.Format("15:04"))
}
