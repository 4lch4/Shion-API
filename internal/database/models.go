package database

import (
	"gorm.io/gorm"
)

type EventEntry struct {
	gorm.Model

	// The type of event.
	Type string `json:"type" binding:"required"`

	// The data associated with the event.
	Data string `json:"data" binding:"required"`

	// The timestamp of the event.
	Timestamp string `json:"timestamp"`
}
