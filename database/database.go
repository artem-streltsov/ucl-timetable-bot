package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	ChatID                 int64
	WebcalURL              string
	LastDailySent          time.Time
	LastWeeklySent         time.Time
	DailyNotificationTime  string
	WeeklyNotificationTime string
	ReminderOffset         int
}

func InitDB(dbPath string) (*sql.DB, error) {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("error creating directory for database: %w", err)
		}
		file, err := os.Create(dbPath)
		if err != nil {
			return nil, fmt.Errorf("error creating database file: %w", err)
		}
		file.Close()
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("error opening database file: %w", err)
	}

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("error running migrations: %w", err)
	}

	return db, nil
}

func runMigrations(db *sql.DB) error {
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("could not create driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "sqlite3", driver)
	if err != nil {
		return fmt.Errorf("could not create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("could not run migrations: %w", err)
	}

	return nil
}

func InsertUser(db *sql.DB, chatID int64, webcalURL string) error {
	insertSQL := `INSERT OR REPLACE INTO users (chatID, webcalURL, lastDailySent, lastWeeklySent) 
				  VALUES (?, ?, NULL, NULL)`
	_, err := db.Exec(insertSQL, chatID, webcalURL)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}
	return nil
}

func GetWebCalURL(db *sql.DB, chatID int64) (string, error) {
	var webcalURL string
	err := db.QueryRow("SELECT webcalURL FROM users WHERE chatID = ?", chatID).Scan(&webcalURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("failed to query user WebCal link: %w", err)
	}
	return webcalURL, nil
}

func GetLastDailySentTime(db *sql.DB, chatID int64) (time.Time, error) {
	var lastDailySent sql.NullTime
	err := db.QueryRow("SELECT lastDailySent FROM users WHERE chatID = ?", chatID).Scan(&lastDailySent)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to query lastDailySent: %w", err)
	}
	if lastDailySent.Valid {
		return lastDailySent.Time, nil
	}
	return time.Time{}, nil
}

func GetLastWeeklySentTime(db *sql.DB, chatID int64) (time.Time, error) {
	var lastWeeklySent sql.NullTime
	err := db.QueryRow("SELECT lastWeeklySent FROM users WHERE chatID = ?", chatID).Scan(&lastWeeklySent)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to query lastWeeklySent: %w", err)
	}
	if lastWeeklySent.Valid {
		return lastWeeklySent.Time, nil
	}
	return time.Time{}, nil
}

func UpdateLastDailySent(db *sql.DB, chatID int64, lastSent time.Time) error {
	_, err := db.Exec("UPDATE users SET lastDailySent = ? WHERE chatID = ?", lastSent, chatID)
	if err != nil {
		return fmt.Errorf("failed to update lastDailySent: %w", err)
	}
	return nil
}

func UpdateLastWeeklySent(db *sql.DB, chatID int64, lastSent time.Time) error {
	_, err := db.Exec("UPDATE users SET lastWeeklySent = ? WHERE chatID = ?", lastSent, chatID)
	if err != nil {
		return fmt.Errorf("failed to update lastWeeklySent: %w", err)
	}
	return nil
}

func GetUserPreferences(db *sql.DB, chatID int64) (string, string, int, error) {
	var dailyNotificationTime, weeklyNotificationTime string
	var reminderOffset int
	err := db.QueryRow("SELECT dailyNotificationTime, weeklyNotificationTime, reminderOffset FROM users WHERE chatID = ?", chatID).Scan(&dailyNotificationTime, &weeklyNotificationTime, &reminderOffset)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to query user preferences: %w", err)
	}
	return dailyNotificationTime, weeklyNotificationTime, reminderOffset, nil
}

func UpdateUserPreferences(db *sql.DB, chatID int64, dailyNotificationTime, weeklyNotificationTime string, reminderOffset int) error {
	_, err := db.Exec("UPDATE users SET dailyNotificationTime = ?, weeklyNotificationTime = ?, reminderOffset = ? WHERE chatID = ?", dailyNotificationTime, weeklyNotificationTime, reminderOffset, chatID)
	if err != nil {
		return fmt.Errorf("failed to update user preferences: %w", err)
	}
	return nil
}

func GetAllUsers(db *sql.DB) ([]User, error) {
	rows, err := db.Query("SELECT chatID, webcalURL, lastDailySent, lastWeeklySent, dailyNotificationTime, weeklyNotificationTime, reminderOffset FROM users")
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		var lastDailySent, lastWeeklySent sql.NullTime
		err := rows.Scan(&user.ChatID, &user.WebcalURL, &lastDailySent, &lastWeeklySent, &user.DailyNotificationTime, &user.WeeklyNotificationTime, &user.ReminderOffset)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}

		if lastDailySent.Valid {
			user.LastDailySent = lastDailySent.Time
		}
		if lastWeeklySent.Valid {
			user.LastWeeklySent = lastWeeklySent.Time
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}

	return users, nil
}
