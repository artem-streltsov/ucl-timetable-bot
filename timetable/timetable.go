package timetable

import (
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	ical "github.com/arran4/golang-ical"
)

var ukLocation, _ = time.LoadLocation("Europe/London")

type Lecture struct {
	Title    string
	Start    time.Time
	End      time.Time
	Location string
}

func FetchCalendar(link string) (*ical.Calendar, error) {
	if strings.HasPrefix(strings.ToLower(link), "webcal://") {
		link = "https://" + link[len("webcal://"):]
	}
	resp, err := http.Get(link)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	cal, err := ical.ParseCalendar(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	return cal, nil
}

func GetLectures(cal *ical.Calendar, day time.Time) ([]Lecture, error) {
	var lectures []Lecture
	for _, event := range cal.Events() {
		start, err := event.GetStartAt()
		if err != nil {
			continue
		}
		end, err := event.GetEndAt()
		if err != nil {
			continue
		}
		start = start.In(ukLocation)
		end = end.In(ukLocation)
		if start.Year() == day.Year() && start.YearDay() == day.YearDay() {
			lecture := Lecture{
				Title:    event.GetProperty(ical.ComponentPropertySummary).Value,
				Start:    start,
				End:      end,
				Location: event.GetProperty(ical.ComponentPropertyLocation).Value,
			}
			lectures = append(lectures, lecture)
		}
	}
	sort.SliceStable(lectures, func(i, j int) bool {
		return lectures[i].Start.Before(lectures[j].Start)
	})
	return lectures, nil
}

func GetLecturesInRange(cal *ical.Calendar, startDay, endDay time.Time) (map[string][]Lecture, error) {
	lecturesMap := make(map[string][]Lecture)
	day := startDay
	for !day.After(endDay) {
		lectures, err := GetLectures(cal, day)
		if err != nil {
			return nil, err
		}
		if len(lectures) > 0 {
			dayKey := day.Format("Monday")
			lecturesMap[dayKey] = lectures
		}
		day = day.AddDate(0, 0, 1)
	}
	return lecturesMap, nil
}

func FormatLectures(lectures []Lecture) string {
	var sb strings.Builder
	for _, lecture := range lectures {
		title := CleanTitle(lecture.Title)
		start := lecture.Start.Format("15:04")
		end := lecture.End.Format("15:04")
		location := lecture.Location

		sb.WriteString("📚 " + "*" + title + "*" + "\n")
		sb.WriteString("⏰ " + start + " - " + end + "\n")
		sb.WriteString("📍 " + location + "\n\n")
	}
	return sb.String()
}

func CleanTitle(title string) string {
	reBrackets := regexp.MustCompile(`\s*\[.*?\]`)
	title = reBrackets.ReplaceAllString(title, "")
	reLevel := regexp.MustCompile(`\s*Level\s*\d+$`)
	title = reLevel.ReplaceAllString(title, "")
	title = strings.TrimSpace(title)
	return title
}
