package models

type User struct {
	ChatID         int64
	Username       string
	WebCalURL      string
	DailyTime      string
	WeeklyTime     string
	ReminderOffset string
}
