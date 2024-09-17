package database

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"path/filepath"
	"time"
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

	stmt, err := db.Prepare(createTableSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare create table statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("failed to execute create table statement: %w", err)
	}

	return db, nil
}

func InsertUser(db *sql.DB, chatID int64, webcalURL string) error {
	insertSQL := `INSERT OR REPLACE INTO users (chatID, webcalURL, lastDailySent, lastWeeklySent) VALUES (?, ?, NULL, NULL)`
	stmt, err := db.Prepare(insertSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(chatID, webcalURL)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}
	return nil
}

func GetWebCalLink(db *sql.DB, chatID int64) (string, error) {
	query := `SELECT webcalURL FROM users WHERE chatID = ?`
	stmt, err := db.Prepare(query)
	if err != nil {
		return "", fmt.Errorf("failed to prepare query statement: %w", err)
	}
	defer stmt.Close()

	var webcalURL string
	err = stmt.QueryRow(chatID).Scan(&webcalURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("failed to query user WebCal link: %w", err)
	}
	return webcalURL, nil
}
