package database

import (
	"database/sql"
	"fmt"

	"github.com/artem-streltsov/ucl-timetable-bot/models"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
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

func (db *DB) GetUser(chatID int64) (*models.User, error) {
	row := db.conn.QueryRow(`SELECT chat_id, username, webcal_url, daily_time, weekly_time, reminder_offset FROM users WHERE chat_id = ?`, chatID)
	var user models.User
	err := row.Scan(&user.ChatID, &user.Username, &user.WebCalURL, &user.DailyTime, &user.WeeklyTime, &user.ReminderOffset)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

func (db *DB) GetUserByUsername(username string) (*models.User, error) {
	row := db.conn.QueryRow(`SELECT chat_id, username, webcal_url, daily_time, weekly_time, reminder_offset FROM users WHERE username = ?`, username)
	var user models.User
	err := row.Scan(&user.ChatID, &user.Username, &user.WebCalURL, &user.DailyTime, &user.WeeklyTime, &user.ReminderOffset)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

func (db *DB) SaveUser(user *models.User) error {
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

func (db *DB) GetAllUsers() ([]*models.User, error) {
	rows, err := db.conn.Query(`SELECT chat_id, username, webcal_url, daily_time, weekly_time, reminder_offset FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ChatID, &user.Username, &user.WebCalURL, &user.DailyTime, &user.WeeklyTime, &user.ReminderOffset)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, nil
}

func (db *DB) AreFriends(userID1, userID2 int64) (bool, error) {
	if userID1 > userID2 {
		userID1, userID2 = userID2, userID1
	}
	row := db.conn.QueryRow(`SELECT 1 FROM friends WHERE user_id1 = ? AND user_id2 = ?`, userID1, userID2)
	var exists int
	err := row.Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (db *DB) FriendRequestExists(requestorID, requesteeID int64) (bool, error) {
	row := db.conn.QueryRow(`SELECT 1 FROM friend_requests WHERE requestor_id = ? AND requestee_id = ?`, requestorID, requesteeID)
	var exists int
	err := row.Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (db *DB) AddFriendRequest(requestorID, requesteeID int64) error {
	_, err := db.conn.Exec(`INSERT INTO friend_requests (requestor_id, requestee_id) VALUES (?, ?)`, requestorID, requesteeID)
	return err
}

func (db *DB) AcceptFriendRequest(requestorID, requesteeID int64) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	user1, user2 := requestorID, requesteeID
	if user1 > user2 {
		user1, user2 = user2, user1
	}
	_, err = tx.Exec(`INSERT INTO friends (user_id1, user_id2) VALUES (?, ?)`, user1, user2)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM friend_requests WHERE requestor_id = ? AND requestee_id = ?`, requestorID, requesteeID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) GetPendingFriendRequests(userID int64) ([]int64, error) {
	rows, err := db.conn.Query(`SELECT requestor_id FROM friend_requests WHERE requestee_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requestors []int64
	for rows.Next() {
		var requestorID int64
		if err := rows.Scan(&requestorID); err != nil {
			return nil, err
		}
		requestors = append(requestors, requestorID)
	}
	return requestors, nil
}
