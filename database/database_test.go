package database_test

import (
	"os"
	"testing"
	"time"

	"github.com/artem-streltsov/ucl-timetable-bot/database"
	"github.com/stretchr/testify/assert"
)

const testDBPath = "./testdata/test.db"

func TestInitDB(t *testing.T) {
	defer os.Remove(testDBPath)

	db, err := database.InitDB(testDBPath)
	assert.NoError(t, err)
	assert.NotNil(t, db)

	_, err = os.Stat(testDBPath)
	assert.NoError(t, err)

	db.Close()
}

func TestInsertAndGetUser(t *testing.T) {
	defer os.Remove(testDBPath)

	db, err := database.InitDB(testDBPath)
	assert.NoError(t, err)
	defer db.Close()

	chatID := int64(123456)
	webcalURL := "https://example.com/calendar"

	err = database.InsertUser(db, chatID, webcalURL)
	assert.NoError(t, err)

	retrievedURL, err := database.GetWebCalURL(db, chatID)
	assert.NoError(t, err)
	assert.Equal(t, webcalURL, retrievedURL)
}

func TestGetLastSentTimes(t *testing.T) {
	defer os.Remove(testDBPath)

	db, err := database.InitDB(testDBPath)
	assert.NoError(t, err)
	defer db.Close()

	chatID := int64(123456)
	webcalURL := "https://example.com/calendar"

	err = database.InsertUser(db, chatID, webcalURL)
	assert.NoError(t, err)

	now := time.Now().UTC()

	err = database.UpdateLastDailySent(db, chatID, now)
	assert.NoError(t, err)

	lastDailySent, err := database.GetLastDailySentTime(db, chatID)
	assert.NoError(t, err)
	assert.WithinDuration(t, now, lastDailySent, time.Second)

	err = database.UpdateLastWeeklySent(db, chatID, now)
	assert.NoError(t, err)

	lastWeeklySent, err := database.GetLastWeeklySentTime(db, chatID)
	assert.NoError(t, err)
	assert.WithinDuration(t, now, lastWeeklySent, time.Second)
}

func TestGetUserPreferences(t *testing.T) {
	defer os.Remove(testDBPath)

	db, err := database.InitDB(testDBPath)
	assert.NoError(t, err)
	defer db.Close()

	chatID := int64(123456)
	webcalURL := "https://example.com/calendar"

	err = database.InsertUser(db, chatID, webcalURL)
	assert.NoError(t, err)

	dailyTime, weeklyTime, offset, err := database.GetUserPreferences(db, chatID)
	assert.NoError(t, err)
	assert.Equal(t, "18:00", dailyTime)
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
	defer os.Remove(testDBPath)

	db, err := database.InitDB(testDBPath)
	assert.NoError(t, err)
	defer db.Close()

	users := []struct {
		chatID    int64
		webcalURL string
	}{
		{123456, "https://example.com/calendar1"},
		{789012, "https://example.com/calendar2"},
	}

	for _, user := range users {
		err = database.InsertUser(db, user.chatID, user.webcalURL)
		assert.NoError(t, err)
	}

	retrievedUsers, err := database.GetAllUsers(db)
	assert.NoError(t, err)
	assert.Len(t, retrievedUsers, len(users))

	for i, user := range retrievedUsers {
		assert.Equal(t, users[i].chatID, user.ChatID)
		assert.Equal(t, users[i].webcalURL, user.WebcalURL)
	}
}

func TestUpdateLastDailySent(t *testing.T) {
	defer os.Remove(testDBPath)

	db, err := database.InitDB(testDBPath)
	assert.NoError(t, err)
	defer db.Close()

	chatID := int64(123456)
	webcalURL := "https://example.com/calendar"
	err = database.InsertUser(db, chatID, webcalURL)
	assert.NoError(t, err)

	now := time.Now()

	err = database.UpdateLastDailySent(db, chatID, now)
	assert.NoError(t, err)

	lastDailySent, err := database.GetLastDailySentTime(db, chatID)
	assert.NoError(t, err)
	assert.WithinDuration(t, now, lastDailySent, time.Second)
}

func TestUpdateLastWeeklySent(t *testing.T) {
	defer os.Remove(testDBPath)

	db, err := database.InitDB(testDBPath)
	assert.NoError(t, err)
	defer db.Close()

	chatID := int64(123456)
	webcalURL := "https://example.com/calendar"
	err = database.InsertUser(db, chatID, webcalURL)
	assert.NoError(t, err)

	now := time.Now()

	err = database.UpdateLastWeeklySent(db, chatID, now)
	assert.NoError(t, err)

	lastWeeklySent, err := database.GetLastWeeklySentTime(db, chatID)
	assert.NoError(t, err)
	assert.WithinDuration(t, now, lastWeeklySent, time.Second)
}

func TestUpdateUserPreferences(t *testing.T) {
	defer os.Remove(testDBPath)

	db, err := database.InitDB(testDBPath)
	assert.NoError(t, err)
	defer db.Close()

	chatID := int64(123456)
	webcalURL := "https://example.com/calendar"

	err = database.InsertUser(db, chatID, webcalURL)
	assert.NoError(t, err)

	err = database.UpdateUserPreferences(db, chatID, "07:30", "SAT 09:00", 20)
	assert.NoError(t, err)

	dailyTime, weeklyTime, offset, err := database.GetUserPreferences(db, chatID)
	assert.NoError(t, err)
	assert.Equal(t, "07:30", dailyTime)
	assert.Equal(t, "SAT 09:00", weeklyTime)
	assert.Equal(t, 20, offset)
}
