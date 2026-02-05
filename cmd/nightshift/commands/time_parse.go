package commands

import (
	"fmt"
	"strings"
	"time"
)

var timeLayouts = []string{
	time.RFC3339,
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02",
}

func parseTimeInput(input string, loc *time.Location) (time.Time, error) {
	value := strings.TrimSpace(strings.ToLower(input))
	now := time.Now().In(loc)

	switch value {
	case "now":
		return now, nil
	case "today":
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc), nil
	case "yesterday":
		y := now.AddDate(0, 0, -1)
		return time.Date(y.Year(), y.Month(), y.Day(), 0, 0, 0, 0, loc), nil
	case "tomorrow":
		t := now.AddDate(0, 0, 1)
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc), nil
	}

	for _, layout := range timeLayouts {
		if layout == time.RFC3339 {
			if parsed, err := time.Parse(layout, input); err == nil {
				return parsed.In(loc), nil
			}
			continue
		}
		if parsed, err := time.ParseInLocation(layout, input, loc); err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid time %q (use YYYY-MM-DD or RFC3339)", input)
}

func parseClock(input string) (int, int, error) {
	parts := strings.Split(strings.TrimSpace(input), ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid clock time %q (use HH:MM)", input)
	}
	hour, err := parseInt(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid hour %q", parts[0])
	}
	minute, err := parseInt(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid minute %q", parts[1])
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("invalid clock time %q", input)
	}
	return hour, minute, nil
}

func parseInt(value string) (int, error) {
	var n int
	_, err := fmt.Sscanf(strings.TrimSpace(value), "%d", &n)
	return n, err
}
