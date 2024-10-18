package database

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

type User struct {
	ChatID         int64
	Username       string
	WebCalURL      string
	DailyTime      string
	WeeklyTime     string
	ReminderOffset string
}

func New(dbPath string) (*DB, error) {
	dbConn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	if err := runMigrations(dbConn); err != nil {
		return nil, err
	}
	return &DB{conn: dbConn}, nil
}

func runMigrations(db *sql.DB) error {
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("could not create migration driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"sqlite3", driver)
	if err != nil {
		return fmt.Errorf("could not create migration instance: %v", err)
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %v", err)
	}

	return nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) GetUser(chatID int64) (*User, error) {
	row := db.conn.QueryRow(`SELECT chat_id, username, webcal_url, daily_time, weekly_time, reminder_offset FROM users WHERE chat_id = ?`, chatID)
	var user User
	err := row.Scan(&user.ChatID, &user.Username, &user.WebCalURL, &user.DailyTime, &user.WeeklyTime, &user.ReminderOffset)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

func (db *DB) SaveUser(user *User) error {
	_, err := db.conn.Exec(`INSERT INTO users (chat_id, username, webcal_url, daily_time, weekly_time, reminder_offset) 
		VALUES (?, ?, ?, ?, ?, ?) 
		ON CONFLICT(chat_id) DO UPDATE SET 
			username=excluded.username, 
			webcal_url=excluded.webcal_url, 
			daily_time=excluded.daily_time, 
			weekly_time=excluded.weekly_time,
            reminder_offset=excluded.reminder_offset`,
		user.ChatID, user.Username, user.WebCalURL, user.DailyTime, user.WeeklyTime, user.ReminderOffset)
	return err
}

func (db *DB) GetAllUsers() ([]*User, error) {
	rows, err := db.conn.Query(`SELECT chat_id, username, webcal_url, daily_time, weekly_time, reminder_offset FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ChatID, &user.Username, &user.WebCalURL, &user.DailyTime, &user.WeeklyTime, &user.ReminderOffset)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, nil
}
