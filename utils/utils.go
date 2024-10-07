package utils

import (
	"fmt"
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

func GetDayWithSuffix(day int) string {
	if day >= 11 && day <= 13 {
		return fmt.Sprintf("%dth", day)
	}
	switch day % 10 {
	case 1:
		return fmt.Sprintf("%dst", day)
	case 2:
		return fmt.Sprintf("%dnd", day)
	case 3:
		return fmt.Sprintf("%drd", day)
	default:
		return fmt.Sprintf("%dth", day)
	}
}

func FormatWeekRange(monday, friday time.Time) string {
	daySuffixMonday := GetDayWithSuffix(monday.Day())
	daySuffixFriday := GetDayWithSuffix(friday.Day())

	formattedMonday := monday.Format("Mon,") + " " + daySuffixMonday + " " + monday.Format("Jan")
	formattedFriday := friday.Format("Fri,") + " " + daySuffixFriday + " " + friday.Format("Jan")

	return fmt.Sprintf("%s - %s", formattedMonday, formattedFriday)
}
