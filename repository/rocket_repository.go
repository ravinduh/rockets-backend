package repository

import (
	"database/sql"
	"fmt"
	"rockets-backend/models"

	_ "github.com/lib/pq"
)

type RocketRepository interface {
	// Rocket operations
	GetRocket(id models.UUID) (*models.Rocket, error)
	GetAllRockets(sortBy string) ([]models.Rocket, error)
	UpsertRocket(rocket *models.Rocket) error

	// Event operations
	CreateRocketEvent(event *models.RocketEvent) error
	GetRocketEvent(id int64) (*models.RocketEvent, error)
	GetPendingEvents(limit int) ([]models.RocketEvent, error)
	UpdateEventStatus(id int64, status string, errorMessage *string) error
	MarkEventProcessed(id int64) error
}

type PostgresRocketRepository struct {
	db *sql.DB
}

func NewPostgresRocketRepository(db *sql.DB) RocketRepository {
	return &PostgresRocketRepository{db: db}
}

func (r *PostgresRocketRepository) GetRocket(id models.UUID) (*models.Rocket, error) {
	query := `
		SELECT id, type, current_speed, mission, status, explosion_reason, 
		       launch_time, last_updated, last_message_number
		FROM rockets WHERE id = $1`

	rocket := &models.Rocket{}
	err := r.db.QueryRow(query, id).Scan(
		&rocket.ID, &rocket.Type, &rocket.CurrentSpeed, &rocket.Mission,
		&rocket.Status, &rocket.ExplosionReason, &rocket.LaunchTime,
		&rocket.LastUpdated, &rocket.LastMessageNumber,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get rocket: %w", err)
	}

	return rocket, nil
}

func (r *PostgresRocketRepository) GetAllRockets(sortBy string) ([]models.Rocket, error) {
	validSorts := map[string]string{
		"type":        "type",
		"speed":       "current_speed",
		"mission":     "mission",
		"status":      "status",
		"launchTime":  "launch_time",
		"lastUpdated": "last_updated",
	}

	orderBy := "last_updated DESC"
	if sort, ok := validSorts[sortBy]; ok {
		orderBy = fmt.Sprintf("%s ASC", sort)
	}

	query := fmt.Sprintf(`
		SELECT id, type, current_speed, mission, status, explosion_reason,
		       launch_time, last_updated, last_message_number
		FROM rockets ORDER BY %s`, orderBy)

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query rockets: %w", err)
	}
	defer rows.Close()

	var rockets []models.Rocket
	for rows.Next() {
		rocket := models.Rocket{}
		err := rows.Scan(
			&rocket.ID, &rocket.Type, &rocket.CurrentSpeed, &rocket.Mission,
			&rocket.Status, &rocket.ExplosionReason, &rocket.LaunchTime,
			&rocket.LastUpdated, &rocket.LastMessageNumber,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rocket: %w", err)
		}
		rockets = append(rockets, rocket)
	}

	return rockets, nil
}

func (r *PostgresRocketRepository) UpsertRocket(rocket *models.Rocket) error {
	query := `
		INSERT INTO rockets (id, type, current_speed, mission, status, explosion_reason,
		                    launch_time, last_updated, last_message_number)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			type = EXCLUDED.type,
			current_speed = EXCLUDED.current_speed,
			mission = EXCLUDED.mission,
			status = EXCLUDED.status,
			explosion_reason = EXCLUDED.explosion_reason,
			launch_time = EXCLUDED.launch_time,
			last_updated = EXCLUDED.last_updated,
			last_message_number = EXCLUDED.last_message_number
		WHERE EXCLUDED.last_message_number > rockets.last_message_number 
		   OR rockets.last_message_number IS NULL`

	_, err := r.db.Exec(query,
		rocket.ID, rocket.Type, rocket.CurrentSpeed, rocket.Mission,
		rocket.Status, rocket.ExplosionReason, rocket.LaunchTime,
		rocket.LastUpdated, rocket.LastMessageNumber,
	)

	if err != nil {
		return fmt.Errorf("failed to create rocket: %w", err)
	}

	return nil
}

// Event operations
func (r *PostgresRocketRepository) CreateRocketEvent(event *models.RocketEvent) error {
	query := `
		INSERT INTO rocket_events (channel, message_number, message_type, message_data, status)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (channel, message_number) DO UPDATE SET
			message_type = EXCLUDED.message_type,
			message_data = EXCLUDED.message_data,
			received_at = CURRENT_TIMESTAMP
		RETURNING id, received_at`

	err := r.db.QueryRow(query,
		event.Channel, event.MessageNumber, event.MessageType,
		event.MessageData, models.EventStatusPending,
	).Scan(&event.ID, &event.ReceivedAt)

	if err != nil {
		return fmt.Errorf("failed to create rocket event: %w", err)
	}

	event.Status = models.EventStatusPending
	return nil
}

func (r *PostgresRocketRepository) GetRocketEvent(id int64) (*models.RocketEvent, error) {
	query := `
		SELECT id, channel, message_number, message_type, message_data, 
		       received_at, processed_at, status, error_message
		FROM rocket_events WHERE id = $1`

	event := &models.RocketEvent{}
	err := r.db.QueryRow(query, id).Scan(
		&event.ID, &event.Channel, &event.MessageNumber, &event.MessageType,
		&event.MessageData, &event.ReceivedAt, &event.ProcessedAt,
		&event.Status, &event.ErrorMessage,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get rocket event: %w", err)
	}

	return event, nil
}

func (r *PostgresRocketRepository) GetPendingEvents(limit int) ([]models.RocketEvent, error) {
	query := `
		SELECT id, channel, message_number, message_type, message_data,
		       received_at, processed_at, status, error_message
		FROM rocket_events 
		WHERE status = $1 
		ORDER BY received_at
		LIMIT $2`

	rows, err := r.db.Query(query, models.EventStatusPending, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending events: %w", err)
	}
	defer rows.Close()

	var events []models.RocketEvent
	for rows.Next() {
		event := models.RocketEvent{}
		err := rows.Scan(
			&event.ID, &event.Channel, &event.MessageNumber, &event.MessageType,
			&event.MessageData, &event.ReceivedAt, &event.ProcessedAt,
			&event.Status, &event.ErrorMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, event)
	}

	return events, nil
}

func (r *PostgresRocketRepository) UpdateEventStatus(id int64, status string, errorMessage *string) error {
	query := `
		UPDATE rocket_events 
		SET status = $2::varchar, 
		    error_message = $3::text, 
		    processed_at = CASE WHEN $2 IN ('processed', 'failed') THEN CURRENT_TIMESTAMP ELSE processed_at END
		WHERE id = $1`

	// Convert *string to sql.NullString to handle nil properly
	var errorParam sql.NullString
	if errorMessage != nil {
		errorParam = sql.NullString{String: *errorMessage, Valid: true}
	}

	_, err := r.db.Exec(query, id, status, errorParam)
	if err != nil {
		return fmt.Errorf("failed to update event status: %w", err)
	}

	return nil
}

func (r *PostgresRocketRepository) MarkEventProcessed(id int64) error {
	return r.UpdateEventStatus(id, models.EventStatusProcessed, nil)
}
