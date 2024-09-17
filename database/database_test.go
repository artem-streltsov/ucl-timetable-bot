package database_test

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"testing"

	"github.com/artem-streltsov/ucl-timetable-bot/database"
)

const testDBPath = "./testdata/test.db"

func setupTestDB(t *testing.T) *sql.DB {
	db, err := database.InitDB(testDBPath)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
	return db
}

func teardownTestDB(t *testing.T, db *sql.DB) {
	db.Close()
	err := os.Remove(testDBPath)
	if err != nil {
		t.Fatalf("Failed to remove test database file: %v", err)
	}
}

func TestInitDB(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	if _, err := os.Stat(testDBPath); os.IsNotExist(err) {
		t.Fatalf("Database file was not created: %v", err)
	}
}

func TestInsertUser(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	err := database.InsertUser(db, 12345, "https://example.com/webcal")
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	var chatID int64
	var webcalURL string
	var lastDailySent sql.NullString
	var lastWeeklySent sql.NullString

	row := db.QueryRow("SELECT chatID, webcalURL, lastDailySent, lastWeeklySent FROM users WHERE chatID = ?", 12345)
	err = row.Scan(&chatID, &webcalURL, &lastDailySent, &lastWeeklySent)
	if err != nil {
		t.Fatalf("Failed to query user: %v", err)
	}

	if chatID != 12345 {
		t.Errorf("Expected chatID 12345, got %d", chatID)
	}
	if webcalURL != "https://example.com/webcal" {
		t.Errorf("Expected webcalURL 'https://example.com/webcal', got %s", webcalURL)
	}
	if lastDailySent.Valid {
		t.Errorf("Expected lastDailySent to be NULL")
	}
	if lastWeeklySent.Valid {
		t.Errorf("Expected lastWeeklySent to be NULL")
	}
}

func TestInsertUserDuplicate(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	err := database.InsertUser(db, 12345, "https://example.com/webcal")
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	err = database.InsertUser(db, 12345, "https://example.com/webcal")
	if err == nil {
		t.Fatalf("Expected error when inserting duplicate user, got none")
	}
}
