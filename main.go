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
)

const dbFile = "user_data.db"

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
		log.Panic(err)
	}

	bot.Debug = false
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

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
		webcalURL TEXT
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

func getUserWebCalURL(db *sql.DB, chatID int64) (string, error) {
	var webcalURL string
	query := "SELECT webcalURL FROM users WHERE chatID = ?"
	err := db.QueryRow(query, chatID).Scan(&webcalURL)
	if err != nil {
		return "", err
	}
	return webcalURL, nil
}

func handleStartCommand(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Welcome! Please provide the link to your WebCal link to subscribe to your lecture timetable.")
	bot.Send(msg)
}

func handleWebCalLink(bot *tgbotapi.BotAPI, db *sql.DB, chatID int64, webcalURL string) {
	webcalURL = strings.ToLower(webcalURL)

	if !strings.HasPrefix(webcalURL, "webcal://") {
		msg := tgbotapi.NewMessage(chatID, "Invalid link! Please provide a valid WebCal link that starts with 'webcal://' or 'Webcal://'.")
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

	msg := tgbotapi.NewMessage(chatID, "Thank you! You will start receiving daily updates for your lectures.")
	bot.Send(msg)

	sendWeeklySummary(bot, chatID, webcalURL)     // Send initial weekly summary
	sendDailySummary(bot, chatID, webcalURL)      // Send initial daily summary
	scheduleDailySummary(bot, chatID, webcalURL)  // Schedule daily updates
	scheduleWeeklySummary(bot, chatID, webcalURL) // Schedule weekly summary for Sundays
}

func scheduleDailySummary(bot *tgbotapi.BotAPI, chatID int64, webcalURL string) {
	now := time.Now()
	nextCheck := time.Date(now.Year(), now.Month(), now.Day(), 7, 0, 0, 0, now.Location())
	if now.After(nextCheck) {
		nextCheck = nextCheck.Add(24 * time.Hour)
	}

	durationUntilNextCheck := nextCheck.Sub(now)
	time.AfterFunc(durationUntilNextCheck, func() {
		if time.Now().Weekday() != time.Saturday || time.Now().Weekday() != time.Sunday {
			sendDailySummary(bot, chatID, webcalURL)
		}
		scheduleDailySummary(bot, chatID, webcalURL)
	})
}

func scheduleWeeklySummary(bot *tgbotapi.BotAPI, chatID int64, webcalURL string) {
	now := time.Now()
	nextSunday := time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, now.Location())
	for nextSunday.Weekday() != time.Sunday {
		nextSunday = nextSunday.Add(24 * time.Hour)
	}

	durationUntilNextSunday := nextSunday.Sub(now)
	time.AfterFunc(durationUntilNextSunday, func() {
		sendWeeklySummary(bot, chatID, webcalURL)
		scheduleWeeklySummary(bot, chatID, webcalURL)
	})
}

func getDayLectures(calendar *ical.Calendar) []*ical.VEvent {
	today := time.Now().Add(1000 * time.Hour).Format("20060102")
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
	now := time.Now().Add(1000 * time.Hour)

	daysOfWeek := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}

	for _, event := range calendar.Events() {
		startTimeStr := event.GetProperty("DTSTART").Value
		startTime, err := time.Parse("20060102T150405Z", startTimeStr)
		if err != nil {
			continue
		}

		if startTime.After(now) && startTime.Before(now.Add(7*24*time.Hour)) {
			weekday := daysOfWeek[int(startTime.Weekday())]
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
	message := "Reminder: Your lecture is starting in 15 minutes!\n"
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
