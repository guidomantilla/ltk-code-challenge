package core

import "time"

type Event struct {
	Id          string    `json:"id,omitempty"`
	Title       string    `json:"title,omitempty"`
	Description string    `json:"description,omitempty"`
	StartTime   time.Time `json:"start_time,omitempty"`
	EndTime     time.Time `json:"end_time,omitempty"`
	CreatedAt   time.Time `json:"createdAt,omitempty"`
}
