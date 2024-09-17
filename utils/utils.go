package utils

import (
	"time"
)

func ParseTimeUTC(timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, timeStr)
}

func CurrentTimeUTC() time.Time {
	return time.Now().UTC()
}
