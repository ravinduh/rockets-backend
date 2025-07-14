package models

import (
	"time"
)

// Rocket represents the current state of a rocket
type Rocket struct {
	ID                UUID      `json:"id" db:"id"`
	Type              string    `json:"type" db:"type"`
	CurrentSpeed      int       `json:"currentSpeed" db:"current_speed"`
	Mission           string    `json:"mission" db:"mission"`
	Status            string    `json:"status" db:"status"`
	ExplosionReason   *string   `json:"explosionReason,omitempty" db:"explosion_reason"`
	LaunchTime        time.Time `json:"launchTime" db:"launch_time"`
	LastUpdated       time.Time `json:"lastUpdated" db:"last_updated"`
	LastMessageNumber int       `json:"-" db:"last_message_number"`
}

// ProcessedMessage tracks which messages have been processed to avoid duplicates
type ProcessedMessage struct {
	Channel     UUID      `db:"channel"`
	MessageNumber int     `db:"message_number"`
	MessageType string    `db:"message_type"`
	ProcessedAt time.Time `db:"processed_at"`
}

// UUID type alias for rocket IDs
type UUID = string