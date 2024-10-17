package database

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

type User struct {
	ChatID     int64
	WebCalURL  string
	DailyTime  string
	WeeklyTime string
}

func New(dbPath string) (*DB, error) {
	dbConn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	_, err = dbConn.Exec(`CREATE TABLE IF NOT EXISTS users (
		chat_id INTEGER PRIMARY KEY, 
		webcal_url TEXT, 
		daily_time TEXT, 
		weekly_time TEXT
	)`)
	if err != nil {
		return nil, err
	}
	return &DB{conn: dbConn}, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) GetUser(chatID int64) (*User, error) {
	row := db.conn.QueryRow(`SELECT chat_id, webcal_url, daily_time, weekly_time FROM users WHERE chat_id = ?`, chatID)
	var user User
	err := row.Scan(&user.ChatID, &user.WebCalURL, &user.DailyTime, &user.WeeklyTime)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

func (db *DB) SaveUser(user *User) error {
	_, err := db.conn.Exec(`INSERT INTO users (chat_id, webcal_url, daily_time, weekly_time) 
		VALUES (?, ?, ?, ?) 
		ON CONFLICT(chat_id) DO UPDATE SET 
			webcal_url=excluded.webcal_url, 
			daily_time=excluded.daily_time, 
			weekly_time=excluded.weekly_time`,
		user.ChatID, user.WebCalURL, user.DailyTime, user.WeeklyTime)
	return err
}

func (db *DB) GetAllUsers() ([]*User, error) {
	rows, err := db.conn.Query(`SELECT chat_id, webcal_url, daily_time, weekly_time FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ChatID, &user.WebCalURL, &user.DailyTime, &user.WeeklyTime)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, nil
}
