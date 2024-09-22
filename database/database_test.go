package database_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/artem-streltsov/ucl-timetable-bot/database"
	"github.com/artem-streltsov/ucl-timetable-bot/utils"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDBPath = "./testdata/test.db"

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	os.Remove(testDBPath)

	dir := filepath.Dir(testDBPath)
	err := os.MkdirAll(dir, 0755)
	require.NoError(t, err)

	db, err := sql.Open("sqlite3", testDBPath)
	require.NoError(t, err)

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	require.NoError(t, err)

	m, err := migrate.NewWithDatabaseInstance("file://../migrations", "sqlite3", driver)
	require.NoError(t, err)

	err = m.Up()
	require.NoError(t, err)

	return db
}

func teardownTestDB(t *testing.T, db *sql.DB) {
	t.Helper()
	db.Close()
	os.Remove(testDBPath)
}

func TestInitDB(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	_, err := os.Stat(testDBPath)
	assert.NoError(t, err)
}

func TestInsertAndGetUser(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	chatID := int64(123456)
	webcalURL := "https://example.com/calendar"

	err := database.InsertUser(db, chatID, webcalURL)
	assert.NoError(t, err)

	retrievedURL, err := database.GetWebCalURL(db, chatID)
	assert.NoError(t, err)
	assert.Equal(t, webcalURL, retrievedURL)
}

func TestGetLastSentTimes(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	chatID := int64(123456)
	webcalURL := "https://example.com/calendar"

	err := database.InsertUser(db, chatID, webcalURL)
	assert.NoError(t, err)

	now := utils.CurrentTimeUK()
	nowEpoch := utils.TimeToEpoch(now)

	err = database.UpdateLastDailySent(db, chatID, nowEpoch)
	assert.NoError(t, err)

	lastDailySentEpoch, err := database.GetLastDailySentTime(db, chatID)
	assert.NoError(t, err)
	assert.Equal(t, nowEpoch, lastDailySentEpoch)

	err = database.UpdateLastWeeklySent(db, chatID, nowEpoch)
	assert.NoError(t, err)

	lastWeeklySentEpoch, err := database.GetLastWeeklySentTime(db, chatID)
	assert.NoError(t, err)
	assert.Equal(t, nowEpoch, lastWeeklySentEpoch)
}

func TestGetUserPreferences(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	chatID := int64(123456)
	webcalURL := "https://example.com/calendar"

	err := database.InsertUser(db, chatID, webcalURL)
	assert.NoError(t, err)

	dailyTime, weeklyTime, offset, err := database.GetUserPreferences(db, chatID)
	assert.NoError(t, err)
	assert.Equal(t, "07:00", dailyTime)
	assert.Equal(t, "SUN 18:00", weeklyTime)
	assert.Equal(t, 30, offset)

	err = database.UpdateUserPreferences(db, chatID, "08:00", "SAT 12:00", 15)
	assert.NoError(t, err)

	dailyTime, weeklyTime, offset, err = database.GetUserPreferences(db, chatID)
	assert.NoError(t, err)
	assert.Equal(t, "08:00", dailyTime)
	assert.Equal(t, "SAT 12:00", weeklyTime)
	assert.Equal(t, 15, offset)
}

func TestGetAllUsers(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	users := []struct {
		chatID    int64
		webcalURL string
	}{
		{123456, "https://example.com/calendar1"},
		{789012, "https://example.com/calendar2"},
	}

	for _, user := range users {
		err := database.InsertUser(db, user.chatID, user.webcalURL)
		assert.NoError(t, err)
	}

	retrievedUsers, err := database.GetAllUsers(db)
	assert.NoError(t, err)
	assert.Len(t, retrievedUsers, len(users))

	for i, user := range retrievedUsers {
		assert.Equal(t, users[i].chatID, user.ChatID)
		assert.Equal(t, users[i].webcalURL, user.WebcalURL)
		assert.Equal(t, int64(0), user.LastDailySent)
		assert.Equal(t, int64(0), user.LastWeeklySent)
	}
}

func TestUpdateLastDailySent(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	chatID := int64(123456)
	webcalURL := "https://example.com/calendar"
	err := database.InsertUser(db, chatID, webcalURL)
	assert.NoError(t, err)

	now := utils.CurrentTimeUK()
	nowEpoch := utils.TimeToEpoch(now)

	err = database.UpdateLastDailySent(db, chatID, nowEpoch)
	assert.NoError(t, err)

	lastDailySentEpoch, err := database.GetLastDailySentTime(db, chatID)
	assert.NoError(t, err)
	assert.Equal(t, nowEpoch, lastDailySentEpoch)
}

func TestUpdateLastWeeklySent(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	chatID := int64(123456)
	webcalURL := "https://example.com/calendar"
	err := database.InsertUser(db, chatID, webcalURL)
	assert.NoError(t, err)

	now := utils.CurrentTimeUK()
	nowEpoch := utils.TimeToEpoch(now)

	err = database.UpdateLastWeeklySent(db, chatID, nowEpoch)
	assert.NoError(t, err)

	lastWeeklySentEpoch, err := database.GetLastWeeklySentTime(db, chatID)
	assert.NoError(t, err)
	assert.Equal(t, nowEpoch, lastWeeklySentEpoch)
}

func TestUpdateUserPreferences(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	chatID := int64(123456)
	webcalURL := "https://example.com/calendar"

	err := database.InsertUser(db, chatID, webcalURL)
	assert.NoError(t, err)

	err = database.UpdateUserPreferences(db, chatID, "07:30", "SAT 09:00", 20)
	assert.NoError(t, err)

	dailyTime, weeklyTime, offset, err := database.GetUserPreferences(db, chatID)
	assert.NoError(t, err)
	assert.Equal(t, "07:30", dailyTime)
	assert.Equal(t, "SAT 09:00", weeklyTime)
	assert.Equal(t, 20, offset)
}
