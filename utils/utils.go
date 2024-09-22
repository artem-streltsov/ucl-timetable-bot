package utils

import (
	"time"
)

var ukLocation *time.Location

func init() {
	var err error
	ukLocation, err = time.LoadLocation("Europe/London")
	if err != nil {
		panic("Failed to load UK time zone: " + err.Error())
	}
}

func ParseTimeUK(timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return time.Time{}, err
	}
	return t.In(ukLocation), nil
}

func CurrentTimeUK() time.Time {
	return time.Now().In(ukLocation)
}

func TimeToEpoch(t time.Time) int64 {
	return t.Unix()
}

func EpochToTimeUK(epoch int64) time.Time {
	return time.Unix(epoch, 0).In(ukLocation)
}

func IsZeroEpoch(epoch int64) bool {
	return epoch == 0
}

func CurrentTimeEpoch() int64 {
	return time.Now().Unix()
}

func GetUKLocation() *time.Location {
	return ukLocation
}
