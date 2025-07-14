package models

import (
	"encoding/json"
	"time"
)

// RocketEvent represents a raw rocket message stored for processing
type RocketEvent struct {
	ID            int64     `json:"id" db:"id"`
	Channel       UUID      `json:"channel" db:"channel"`
	MessageNumber int       `json:"message_number" db:"message_number"`
	MessageType   string    `json:"message_type" db:"message_type"`
	MessageData   json.RawMessage `json:"message_data" db:"message_data"`
	ReceivedAt    time.Time `json:"received_at" db:"received_at"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty" db:"processed_at"`
	Status        string    `json:"status" db:"status"`
	ErrorMessage  *string   `json:"error_message,omitempty" db:"error_message"`
}

// EventStatus constants
const (
	EventStatusPending    = "pending"
	EventStatusProcessing = "processing"
	EventStatusProcessed  = "processed"
	EventStatusFailed     = "failed"
)

