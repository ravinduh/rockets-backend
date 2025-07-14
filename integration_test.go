package main

import (
	"context"
	"encoding/json"
	"rockets-backend/models"
	"rockets-backend/repository"
	"rockets-backend/service"
	"rockets-backend/testutil"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
)

// TestRocketLifecycleIntegrationDB tests a complete rocket lifecycle with actual PostgreSQL database
func TestRocketLifecycleIntegrationDB(t *testing.T) {
	testutil.SkipIfNoTestDB(t)

	// Setup real database connection
	db := testutil.SetupTestDB(t)
	defer db.Close()
	defer testutil.CleanupTestDB(t, db)

	// Create real repository and service
	logger := log.NewNopLogger()
	repo := repository.NewPostgresRocketRepository(db)
	svc := service.NewService(logger, repo)

	ctx := context.Background()
	rocketChannel := uuid.New().String()

	// Test Phase 1: Rocket Launch
	t.Log("Phase 1: Processing RocketLaunched event with real database")
	launchData, _ := json.Marshal(map[string]interface{}{
		"type":        "Falcon-9",
		"launchSpeed": 500,
		"mission":     "ARTEMIS",
	})

	launchEvent := &models.RocketEvent{
		Channel:       rocketChannel,
		MessageNumber: 1,
		MessageType:   "RocketLaunched",
		MessageData:   launchData,
		Status:        models.EventStatusPending,
		ReceivedAt:    time.Now(),
	}

	// Test event creation in database
	err := repo.CreateRocketEvent(launchEvent)
	testutil.AssertNoError(t, err)
	testutil.AssertNotEqual(t, 0, launchEvent.ID) // Should have auto-generated ID

	// Process the event
	err = svc.ProcessEvent(ctx, launchEvent)
	testutil.AssertNoError(t, err)

	// Verify rocket was created in database
	rocket, err := repo.GetRocket(rocketChannel)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, rocket)
	testutil.AssertEqual(t, "Falcon-9", rocket.Type)
	testutil.AssertEqual(t, 500, rocket.CurrentSpeed)
	testutil.AssertEqual(t, "ARTEMIS", rocket.Mission)
	testutil.AssertEqual(t, "active", rocket.Status)
	testutil.AssertEqual(t, 1, rocket.LastMessageNumber)

	// Verify event status was updated in database
	savedEvent, err := repo.GetRocketEvent(launchEvent.ID)
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, models.EventStatusProcessed, savedEvent.Status)
	testutil.AssertNotNil(t, savedEvent.ProcessedAt)

	// Test Phase 2: Speed Increase
	t.Log("Phase 2: Processing RocketSpeedIncreased event")
	speedIncreaseData, _ := json.Marshal(map[string]interface{}{
		"by": 300,
	})

	speedEvent := &models.RocketEvent{
		Channel:       rocketChannel,
		MessageNumber: 2,
		MessageType:   "RocketSpeedIncreased",
		MessageData:   speedIncreaseData,
		Status:        models.EventStatusPending,
		ReceivedAt:    time.Now(),
	}

	err = repo.CreateRocketEvent(speedEvent)
	testutil.AssertNoError(t, err)

	err = svc.ProcessEvent(ctx, speedEvent)
	testutil.AssertNoError(t, err)

	// Verify speed was updated in database
	rocket, err = repo.GetRocket(rocketChannel)
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, 800, rocket.CurrentSpeed) // 500 + 300
	testutil.AssertEqual(t, 2, rocket.LastMessageNumber)

	// Test Phase 3: Out-of-Order Message (should be ignored)
	t.Log("Phase 3: Testing out-of-order message handling")
	outOfOrderData, _ := json.Marshal(map[string]interface{}{
		"by": 500,
	})

	outOfOrderEvent := &models.RocketEvent{
		Channel:       rocketChannel,
		MessageNumber: 1, // Lower than current last message number
		MessageType:   "RocketSpeedIncreased",
		MessageData:   outOfOrderData,
		Status:        models.EventStatusPending,
		ReceivedAt:    time.Now(),
	}

	err = repo.CreateRocketEvent(outOfOrderEvent)
	testutil.AssertNoError(t, err)

	err = svc.ProcessEvent(ctx, outOfOrderEvent)
	testutil.AssertNoError(t, err)

	// Verify rocket state unchanged in database
	rocket, err = repo.GetRocket(rocketChannel)
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, 800, rocket.CurrentSpeed)    // Should remain unchanged
	testutil.AssertEqual(t, 2, rocket.LastMessageNumber) // Should remain unchanged

	// Test Phase 4: Rocket Explosion
	t.Log("Phase 4: Processing RocketExploded event")
	explosionData, _ := json.Marshal(map[string]interface{}{
		"reason": "engine malfunction",
	})

	explosionEvent := &models.RocketEvent{
		Channel:       rocketChannel,
		MessageNumber: 3,
		MessageType:   "RocketExploded",
		MessageData:   explosionData,
		Status:        models.EventStatusPending,
		ReceivedAt:    time.Now(),
	}

	err = repo.CreateRocketEvent(explosionEvent)
	testutil.AssertNoError(t, err)

	err = svc.ProcessEvent(ctx, explosionEvent)
	testutil.AssertNoError(t, err)

	// Verify explosion was recorded in database
	rocket, err = repo.GetRocket(rocketChannel)
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, "exploded", rocket.Status)
	testutil.AssertEqual(t, 0, rocket.CurrentSpeed) // Speed reset to 0
	testutil.AssertNotNil(t, rocket.ExplosionReason)
	testutil.AssertEqual(t, "engine malfunction", *rocket.ExplosionReason)
	testutil.AssertEqual(t, 3, rocket.LastMessageNumber)

	// Test Phase 5: Database Queries
	t.Log("Phase 5: Testing database query functionality")

	// Test GetAllRockets
	rockets, err := repo.GetAllRockets("")
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, 1, len(rockets))
	testutil.AssertEqual(t, rocketChannel, rockets[0].ID)

	// Test GetPendingEvents (should have none pending)
	pendingEvents, err := repo.GetPendingEvents(10)
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, 0, len(pendingEvents)) // All events should be processed

	t.Log("Database integration test completed successfully - verified actual database persistence")
}

// TestMultipleRocketsIntegrationDB tests processing multiple rockets with real database
func TestMultipleRocketsIntegrationDB(t *testing.T) {
	testutil.SkipIfNoTestDB(t)

	db := testutil.SetupTestDB(t)
	defer db.Close()
	defer testutil.CleanupTestDB(t, db)

	logger := log.NewNopLogger()
	repo := repository.NewPostgresRocketRepository(db)
	svc := service.NewService(logger, repo)

	ctx := context.Background()

	// Create 3 different rockets
	rockets := []struct {
		channel string
		rType   string
		speed   int
		mission string
	}{
		{uuid.New().String(), "Falcon-9", 500, "ARTEMIS"},
		{uuid.New().String(), "Falcon-Heavy", 800, "MARS-MISSION"},
		{uuid.New().String(), "Starship", 1200, "ISS-SUPPLY"},
	}

	// Launch all rockets and verify database persistence
	for i, r := range rockets {
		t.Logf("Launching rocket %d: %s", i+1, r.rType)

		launchData, _ := json.Marshal(map[string]interface{}{
			"type":        r.rType,
			"launchSpeed": r.speed,
			"mission":     r.mission,
		})

		event := &models.RocketEvent{
			Channel:       r.channel,
			MessageNumber: 1,
			MessageType:   "RocketLaunched",
			MessageData:   launchData,
			Status:        models.EventStatusPending,
			ReceivedAt:    time.Now(),
		}

		err := repo.CreateRocketEvent(event)
		testutil.AssertNoError(t, err)

		err = svc.ProcessEvent(ctx, event)
		testutil.AssertNoError(t, err)

		// Verify each rocket in database
		savedRocket, err := repo.GetRocket(r.channel)
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, r.rType, savedRocket.Type)
		testutil.AssertEqual(t, r.speed, savedRocket.CurrentSpeed)
		testutil.AssertEqual(t, r.mission, savedRocket.Mission)
		testutil.AssertEqual(t, "active", savedRocket.Status)
	}

	// Verify all rockets exist in database
	allRockets, err := repo.GetAllRockets("")
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, 3, len(allRockets))

	// Test sorting
	sortedRockets, err := repo.GetAllRockets("speed")
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, 3, len(sortedRockets))
	// Should be sorted by speed: Falcon-9 (500), Falcon-Heavy (800), Starship (1200)
	testutil.AssertEqual(t, 500, sortedRockets[0].CurrentSpeed)
	testutil.AssertEqual(t, 800, sortedRockets[1].CurrentSpeed)
	testutil.AssertEqual(t, 1200, sortedRockets[2].CurrentSpeed)

	t.Log("Multiple rockets database integration test completed successfully")
}

// TestDatabaseConstraintsDB tests database constraints and error handling
func TestDatabaseConstraintsDB(t *testing.T) {
	testutil.SkipIfNoTestDB(t)

	db := testutil.SetupTestDB(t)
	defer db.Close()
	defer testutil.CleanupTestDB(t, db)

	repo := repository.NewPostgresRocketRepository(db)
	channel := uuid.New().String()

	// Test unique constraint on (channel, message_number)
	event1 := &models.RocketEvent{
		Channel:       channel,
		MessageNumber: 1,
		MessageType:   "RocketLaunched",
		MessageData:   []byte(`{"type":"Falcon-9","launchSpeed":500,"mission":"TEST"}`),
		Status:        models.EventStatusPending,
		ReceivedAt:    time.Now(),
	}

	event2 := &models.RocketEvent{
		Channel:       channel,
		MessageNumber: 1, // Same message number
		MessageType:   "RocketSpeedIncreased",
		MessageData:   []byte(`{"by":100}`),
		Status:        models.EventStatusPending,
		ReceivedAt:    time.Now(),
	}

	// First event should succeed
	err := repo.CreateRocketEvent(event1)
	testutil.AssertNoError(t, err)

	// Second event with same channel+message_number should succeed due to ON CONFLICT DO UPDATE
	err = repo.CreateRocketEvent(event2)
	testutil.AssertNoError(t, err)

	// Both events should have same ID due to upsert
	testutil.AssertEqual(t, event1.ID, event2.ID)

	t.Log("Database constraints test completed successfully")
}
