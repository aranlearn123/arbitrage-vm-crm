package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

func queryTime(c *fiber.Ctx, name string) (time.Time, error) {
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

func queryFloat(c *fiber.Ctx, name string, defaultValue float64) (float64, error) {
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

func queryInt(c *fiber.Ctx, name string, defaultValue int) (int, error) {
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
