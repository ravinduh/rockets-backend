package service

import (
	"context"
	"encoding/json"
	"fmt"
	"rockets-backend/models"
	pkgContext "rockets-backend/pkg/context"
	"rockets-backend/repository"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type Service interface {
	HealthCheck() interface{}

	// Message ingestion (fast, async)
	IngestMessage(ctx context.Context, msg models.IncomingMessage) (*models.RocketEvent, error)

	// Event processing (background)
	ProcessEvent(ctx context.Context, event *models.RocketEvent) error

	// Rocket queries
	GetRocket(ctx context.Context, id models.UUID) (*models.Rocket, error)
	GetAllRockets(ctx context.Context, sortBy string) ([]models.Rocket, error)

	// Event status
	GetEventStatus(ctx context.Context, eventID int64) (*models.RocketEvent, error)
}

type service struct {
	logger     log.Logger
	repository repository.RocketRepository
}

func (s service) HealthCheck() interface{} {
	return map[string]string{"status": "OK", "service": "rockets-backend"}
}

// IngestMessage quickly stores the incoming message for async processing
func (s service) IngestMessage(ctx context.Context, msg models.IncomingMessage) (*models.RocketEvent, error) {
	requestID := pkgContext.GetRequestID(ctx)

	// Convert message to JSON for storage
	messageData, err := json.Marshal(msg.Message)
	if err != nil {
		_ = level.Error(s.logger).Log("request_id", requestID, "msg", "failed to marshal message", "error", err)
		return nil, fmt.Errorf("failed to marshal message data: %w", err)
	}

	// Create rocket event
	event := &models.RocketEvent{
		Channel:       msg.Metadata.Channel,
		MessageNumber: msg.Metadata.MessageNumber,
		MessageType:   msg.Metadata.MessageType,
		MessageData:   messageData,
		Status:        models.EventStatusPending,
	}

	// Store event in database
	err = s.repository.CreateRocketEvent(event)
	if err != nil {
		_ = level.Error(s.logger).Log("request_id", requestID, "msg", "failed to store event", "channel", msg.Metadata.Channel, "messageNumber", msg.Metadata.MessageNumber, "error", err)
		return nil, fmt.Errorf("failed to store rocket event: %w", err)
	}

	_ = level.Info(s.logger).Log("request_id", requestID, "msg", "message ingested", "event_id", event.ID, "channel", msg.Metadata.Channel, "messageNumber", msg.Metadata.MessageNumber, "type", msg.Metadata.MessageType)

	return event, nil
}

// ProcessEvent processes a single event and updates rocket state
func (s service) ProcessEvent(ctx context.Context, event *models.RocketEvent) error {
	requestID := pkgContext.GetRequestID(ctx)

	// Mark event as processing
	err := s.repository.UpdateEventStatus(event.ID, models.EventStatusProcessing, nil)
	if err != nil {
		_ = level.Error(s.logger).Log("request_id", requestID, "msg", "failed to mark event as processing",
			"event_id", event.ID, "error", err)
		return fmt.Errorf("failed to mark event as processing: %w", err)
	}

	// Get existing rocket or prepare new one
	rocket, err := s.repository.GetRocket(event.Channel)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to get rocket: %v", err)
		s.repository.UpdateEventStatus(event.ID, models.EventStatusFailed, &errorMsg)
		return fmt.Errorf("failed to get rocket: %w", err)
	}

	// Check message ordering - only process if message number is higher
	if rocket != nil && event.MessageNumber <= rocket.LastMessageNumber {
		_ = level.Debug(s.logger).Log("request_id", requestID, "msg", "ignoring out-of-order event",
			"event_id", event.ID, "channel", event.Channel, "messageNumber", event.MessageNumber,
			"lastProcessed", rocket.LastMessageNumber)
		s.repository.MarkEventProcessed(event.ID)
		return nil
	}

	// Process the event based on type
	if rocket == nil {
		rocket = &models.Rocket{
			ID:                event.Channel,
			Status:            "active",
			LastMessageNumber: 0,
		}
	}

	// Unmarshal message data and process
	var messageData interface{}
	if err := json.Unmarshal(event.MessageData, &messageData); err != nil {
		errorMsg := fmt.Sprintf("failed to unmarshal message data: %v", err)
		s.repository.UpdateEventStatus(event.ID, models.EventStatusFailed, &errorMsg)
		return fmt.Errorf("failed to unmarshal message data: %w", err)
	}

	switch event.MessageType {
	case "RocketLaunched":
		err = s.processRocketLaunchedFromData(rocket, event.MessageData)
	case "RocketSpeedIncreased":
		err = s.processRocketSpeedIncreasedFromData(rocket, event.MessageData)
	case "RocketSpeedDecreased":
		err = s.processRocketSpeedDecreasedFromData(rocket, event.MessageData)
	case "RocketExploded":
		err = s.processRocketExplodedFromData(rocket, event.MessageData)
	case "RocketMissionChanged":
		err = s.processRocketMissionChangedFromData(rocket, event.MessageData)
	default:
		errorMsg := fmt.Sprintf("unknown message type: %s", event.MessageType)
		s.repository.UpdateEventStatus(event.ID, models.EventStatusFailed, &errorMsg)
		return fmt.Errorf("unknown message type: %s", event.MessageType)
	}

	if err != nil {
		errorMsg := fmt.Sprintf("failed to process %s message: %v", event.MessageType, err)
		s.repository.UpdateEventStatus(event.ID, models.EventStatusFailed, &errorMsg)
		return fmt.Errorf("failed to process %s message: %w", event.MessageType, err)
	}

	// Update rocket state
	rocket.LastMessageNumber = event.MessageNumber
	rocket.LastUpdated = time.Now()

	// Save rocket (upsert - create or update)
	err = s.repository.UpsertRocket(rocket)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to save rocket: %v", err)
		s.repository.UpdateEventStatus(event.ID, models.EventStatusFailed, &errorMsg)
		return fmt.Errorf("failed to save rocket: %w", err)
	}

	// Mark event as processed
	err = s.repository.MarkEventProcessed(event.ID)
	if err != nil {
		_ = level.Error(s.logger).Log("request_id", requestID, "msg", "failed to mark event as processed",
			"event_id", event.ID, "error", err)
		return fmt.Errorf("failed to mark event as processed: %w", err)
	}

	_ = level.Info(s.logger).Log("request_id", requestID, "msg", "event processed successfully", "event_id", event.ID,
		"type", event.MessageType, "channel", event.Channel, "messageNumber", event.MessageNumber)
	return nil
}

// GetEventStatus returns the current status of an event
func (s service) GetEventStatus(ctx context.Context, eventID int64) (*models.RocketEvent, error) {
	requestID := pkgContext.GetRequestID(ctx)
	_ = level.Debug(s.logger).Log("request_id", requestID, "msg", "getting event status", "event_id", eventID)

	event, err := s.repository.GetRocketEvent(eventID)
	if err != nil {
		_ = level.Error(s.logger).Log("request_id", requestID, "msg", "failed to get event", "event_id", eventID,
			"error", err)
		return nil, err
	}

	_ = level.Debug(s.logger).Log("request_id", requestID, "msg", "event status retrieved", "event_id", eventID,
		"found", event != nil)
	return event, nil
}

// Processing methods that work with raw JSON data (for async event processing)
func (s service) processRocketLaunchedFromData(rocket *models.Rocket, messageData json.RawMessage) error {
	var payload models.RocketLaunchedMessage
	if err := json.Unmarshal(messageData, &payload); err != nil {
		return fmt.Errorf("failed to parse RocketLaunched payload: %w", err)
	}

	rocket.Type = payload.Type
	rocket.CurrentSpeed = payload.LaunchSpeed
	rocket.Mission = payload.Mission
	rocket.LaunchTime = time.Now() // Use current time since we don't have message time in stored data
	rocket.Status = "active"

	return nil
}

func (s service) processRocketSpeedIncreasedFromData(rocket *models.Rocket, messageData json.RawMessage) error {
	var payload models.RocketSpeedIncreasedMessage
	if err := json.Unmarshal(messageData, &payload); err != nil {
		return fmt.Errorf("failed to parse RocketSpeedIncreased payload: %w", err)
	}

	rocket.CurrentSpeed += payload.By
	return nil
}

func (s service) processRocketSpeedDecreasedFromData(rocket *models.Rocket, messageData json.RawMessage) error {
	var payload models.RocketSpeedDecreasedMessage
	if err := json.Unmarshal(messageData, &payload); err != nil {
		return fmt.Errorf("failed to parse RocketSpeedDecreased payload: %w", err)
	}

	rocket.CurrentSpeed -= payload.By
	if rocket.CurrentSpeed < 0 {
		rocket.CurrentSpeed = 0
	}
	return nil
}

func (s service) processRocketExplodedFromData(rocket *models.Rocket, messageData json.RawMessage) error {
	var payload models.RocketExplodedMessage
	if err := json.Unmarshal(messageData, &payload); err != nil {
		return fmt.Errorf("failed to parse RocketExploded payload: %w", err)
	}

	rocket.Status = "exploded"
	rocket.ExplosionReason = &payload.Reason
	rocket.CurrentSpeed = 0

	return nil
}

func (s service) processRocketMissionChangedFromData(rocket *models.Rocket, messageData json.RawMessage) error {
	var payload models.RocketMissionChangedMessage
	if err := json.Unmarshal(messageData, &payload); err != nil {
		return fmt.Errorf("failed to parse RocketMissionChanged payload: %w", err)
	}

	rocket.Mission = payload.NewMission
	return nil
}

func (s service) GetRocket(ctx context.Context, id models.UUID) (*models.Rocket, error) {
	requestID := pkgContext.GetRequestID(ctx)
	_ = level.Debug(s.logger).Log("request_id", requestID, "msg", "getting rocket", "rocket_id", id)

	rocket, err := s.repository.GetRocket(id)
	if err != nil {
		_ = level.Error(s.logger).Log("request_id", requestID, "msg", "failed to get rocket", "rocket_id", id,
			"error", err)
		return nil, err
	}

	_ = level.Debug(s.logger).Log("request_id", requestID, "msg", "rocket retrieved", "rocket_id", id,
		"found", rocket != nil)
	return rocket, nil
}

func (s service) GetAllRockets(ctx context.Context, sortBy string) ([]models.Rocket, error) {
	requestID := pkgContext.GetRequestID(ctx)
	_ = level.Debug(s.logger).Log("request_id", requestID, "msg", "getting all rockets", "sortBy", sortBy)

	rockets, err := s.repository.GetAllRockets(sortBy)
	if err != nil {
		_ = level.Error(s.logger).Log("request_id", requestID, "msg", "failed to get rockets", "sortBy", sortBy,
			"error", err)
		return nil, err
	}

	_ = level.Debug(s.logger).Log("request_id", requestID, "msg", "rockets retrieved", "count", len(rockets),
		"sortBy", sortBy)
	return rockets, nil
}

// NewService returns a rockets backend service
func NewService(logger log.Logger, repo repository.RocketRepository) Service {
	return &service{
		logger:     logger,
		repository: repo,
	}
}
