package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func queryTime(c *Context, name string) (time.Time, error) {
	value := strings.TrimSpace(c.Query(name))
	if value == "" {
		return time.Time{}, nil
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid %s time format", name)
}

func queryFloat(c *Context, name string, defaultValue float64) (float64, error) {
	value := strings.TrimSpace(c.Query(name))
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s value", name)
	}
	return parsed, nil
}

func queryInt(c *Context, name string, defaultValue int) (int, error) {
	value := strings.TrimSpace(c.Query(name))
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s value", name)
	}
	return parsed, nil
}
