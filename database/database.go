package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	ChatID         int64
	WebcalURL      string
	LastDailySent  time.Time
	LastWeeklySent time.Time
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
		return nil, fmt.Errorf("failed to create users table: %w", err)
	}

	return db, nil
}

func InsertUser(db *sql.DB, chatID int64, webcalURL string) error {
	insertSQL := `INSERT OR REPLACE INTO users (chatID, webcalURL, lastDailySent, lastWeeklySent) VALUES (?, ?, NULL, NULL)`
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

func GetAllUsers(db *sql.DB) ([]User, error) {
	rows, err := db.Query("SELECT chatID, webcalURL, lastDailySent, lastWeeklySent FROM users")
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		var lastDailySent, lastWeeklySent sql.NullTime
		err := rows.Scan(&user.ChatID, &user.WebcalURL, &lastDailySent, &lastWeeklySent)
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
