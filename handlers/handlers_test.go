package handlers_test

import (
	"database/sql"
	"log"
	"testing"

	"github.com/artem-streltsov/ucl-timetable-bot/handlers"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}

	schema := `
        CREATE TABLE users (
            chatID INTEGER PRIMARY KEY,
            webcalURL TEXT,
            lastDailySent DATETIME,
            lastWeeklySent DATETIME
        );
        `
	_, err = db.Exec(schema)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func TestValidateWebCalLink(t *testing.T) {
	cases := []struct {
		input    string
		expected string
		valid    bool
	}{
		{"webcal://example.com", "https://example.com", true},
		{"WEBCAL://example.com", "https://example.com", true},
		{"http://example.com", "", false},
		{"example.com", "", false},
	}

	for _, c := range cases {
		result, valid := handlers.ValidateWebCalLink(c.input)
		assert.Equal(t, c.expected, result)
		assert.Equal(t, c.valid, valid)
	}
}

func TestSaveWebCalLink(t *testing.T) {
	db, err := initDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	chatID := int64(123)
	webcalURL := "https://example.com"

	err = handlers.SaveWebCalLink(db, chatID, webcalURL)
	assert.NoError(t, err)

	var url string
	row := db.QueryRow("SELECT webcalURL FROM users WHERE chatID = ?", chatID)
	err = row.Scan(&url)
	assert.NoError(t, err)
	assert.Equal(t, webcalURL, url)
}
