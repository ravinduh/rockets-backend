package models

import "time"

// IncomingMessage represents the message structure from rockets
type IncomingMessage struct {
	Metadata MessageMetadata `json:"metadata"`
	Message  interface{}     `json:"message"`
}

// MessageMetadata contains the message metadata
type MessageMetadata struct {
	Channel       UUID      `json:"channel"`
	MessageNumber int       `json:"messageNumber"`
	MessageTime   time.Time `json:"messageTime"`
	MessageType   string    `json:"messageType"`
}

// RocketLaunchedMessage payload
type RocketLaunchedMessage struct {
	Type        string `json:"type"`
	LaunchSpeed int    `json:"launchSpeed"`
	Mission     string `json:"mission"`
}

// RocketSpeedIncreasedMessage payload
type RocketSpeedIncreasedMessage struct {
	By int `json:"by"`
}

// RocketSpeedDecreasedMessage payload
type RocketSpeedDecreasedMessage struct {
	By int `json:"by"`
}

// RocketExplodedMessage payload
type RocketExplodedMessage struct {
	Reason string `json:"reason"`
}

// RocketMissionChangedMessage payload
type RocketMissionChangedMessage struct {
	NewMission string `json:"newMission"`
}