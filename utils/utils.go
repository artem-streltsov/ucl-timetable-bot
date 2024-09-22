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

func TimeToEpoch(t time.Time) int64 {
	return t.Unix()
}

func EpochToTime(epoch int64) time.Time {
	return time.Unix(epoch, 0)
}

func IsZeroEpoch(epoch int64) bool {
	return epoch == 0
}

func CurrentTimeEpoch() int64 {
	return time.Now().Unix()
}
