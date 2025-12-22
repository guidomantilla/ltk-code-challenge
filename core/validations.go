package core

import (
	"errors"
	"strings"
)

func ValidateEvent(event Event) error {
	event.Title = strings.TrimSpace(event.Title)
	if len(event.Title) == 0 {
		return errors.New("title is required")
	}

	if len(event.Title) > 100 {
		return errors.New("title is too long (100 characters tops)")
	}

	if event.EndTime.Before(event.StartTime) {
		return errors.New("end time must be after start time")
	}

	return nil
}
