package utils

import (
	"strings"
	"time"
)

var ukLocation, _ = time.LoadLocation("Europe/London")

func IsValidTime(timeStr string) bool {
	_, err := time.Parse("15:04", timeStr)
	return err == nil
}

func IsValidDay(dayStr string) bool {
	days := []string{"MON", "TUE", "WED", "THU", "FRI", "SAT", "SUN"}
	dayStr = strings.ToUpper(dayStr)
	for _, day := range days {
		if day == dayStr {
			return true
		}
	}
	return false
}

func GetNextTime(timeStr string) time.Time {
	now := time.Now().In(ukLocation)
	parsedTime, _ := time.Parse("15:04", timeStr)
	nextTime := time.Date(now.Year(), now.Month(), now.Day(), parsedTime.Hour(), parsedTime.Minute(), 0, 0, now.Location())
	if nextTime.Before(now) {
		nextTime = nextTime.Add(24 * time.Hour)
	}
	return nextTime
}

func GetNextWeekTime(weekTimeStr string) time.Time {
	parts := strings.SplitN(weekTimeStr, " ", 2)
	dayStr, timeStr := parts[0], parts[1]
	weekday := getWeekday(dayStr)
	now := time.Now().In(ukLocation)
	parsedTime, _ := time.Parse("15:04", timeStr)
	nextTime := time.Date(now.Year(), now.Month(), now.Day(), parsedTime.Hour(), parsedTime.Minute(), 0, 0, now.Location())
	for nextTime.Weekday() != weekday || nextTime.Before(now) {
		nextTime = nextTime.Add(24 * time.Hour)
	}
	return nextTime
}

func getWeekday(dayStr string) time.Weekday {
	switch strings.ToUpper(dayStr) {
	case "MON":
		return time.Monday
	case "TUE":
		return time.Tuesday
	case "WED":
		return time.Wednesday
	case "THU":
		return time.Thursday
	case "FRI":
		return time.Friday
	case "SAT":
		return time.Saturday
	case "SUN":
		return time.Sunday
	default:
		return time.Sunday
	}
}
