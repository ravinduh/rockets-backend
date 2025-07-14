package worker

import (
	"context"
	"rockets-backend/models"
	"rockets-backend/pkg"
	"rockets-backend/repository"
	"rockets-backend/service"
	"strconv"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// EventProcessor handles background processing of rocket events
type EventProcessor struct {
	service      service.Service
	repository   repository.RocketRepository
	logger       log.Logger
	pollInterval time.Duration
	batchSize    int
	workerCount  int
	stopChan     chan struct{}
	wg           sync.WaitGroup
	running      bool
	mu           sync.RWMutex
}

// Config holds configuration for the event processor
type Config struct {
	PollInterval time.Duration // How often to check for new events
	BatchSize    int           // How many events to process at once
	WorkerCount  int           // Number of concurrent workers
}

// DefaultConfig returns sensible default configuration
func DefaultConfig() Config {
	pollInterval, _ := strconv.Atoi(pkg.GetEnv("POLLING_INTEVAL_SECONDS", "1"))
	batchSize, _ := strconv.Atoi(pkg.GetEnv("POLLING_BATCH_SIZE", "10"))
	workerCount, _ := strconv.Atoi(pkg.GetEnv("POLLING_WORKER_COUNT", "2"))

	return Config{
		PollInterval: time.Duration(pollInterval) * time.Second,
		BatchSize:    batchSize,
		WorkerCount:  workerCount,
	}
}

// NewEventProcessor creates a new event processor
func NewEventProcessor(svc service.Service, repo repository.RocketRepository, logger log.Logger, config Config) *EventProcessor {
	return &EventProcessor{
		service:      svc,
		repository:   repo,
		logger:       logger,
		pollInterval: config.PollInterval,
		batchSize:    config.BatchSize,
		workerCount:  config.WorkerCount,
		stopChan:     make(chan struct{}),
	}
}

// Start begins processing events in the background
func (p *EventProcessor) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return nil // Already running
	}

	p.running = true
	_ = level.Info(p.logger).Log("msg", "starting event processor", "workers", p.workerCount, "pollInterval",
		p.pollInterval, "batchSize", p.batchSize)

	// Start worker goroutines
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}

	return nil
}

// Stop gracefully shuts down the event processor
func (p *EventProcessor) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil // Already stopped
	}

	_ = level.Info(p.logger).Log("msg", "stopping event processor")

	// Signal all workers to stop
	close(p.stopChan)

	// Wait for all workers to finish
	p.wg.Wait()

	p.running = false
	_ = level.Info(p.logger).Log("msg", "event processor stopped")

	return nil
}

// IsRunning returns whether the processor is currently running
func (p *EventProcessor) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// worker is the main processing loop for each worker goroutine
func (p *EventProcessor) worker(ctx context.Context, workerID int) {
	defer p.wg.Done()

	_ = level.Debug(p.logger).Log("msg", "worker started", "worker_id", workerID)

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			_ = level.Debug(p.logger).Log("msg", "worker stopping", "worker_id", workerID)
			return
		case <-ctx.Done():
			_ = level.Debug(p.logger).Log("msg", "worker context cancelled", "worker_id", workerID)
			return
		case <-ticker.C:
			p.processEvents(ctx, workerID)
		}
	}
}

// processEvents fetches and processes a batch of pending events
func (p *EventProcessor) processEvents(ctx context.Context, workerID int) {
	// Get pending events from repository
	events, err := p.repository.GetPendingEvents(p.batchSize)
	if err != nil {
		_ = level.Error(p.logger).Log("msg", "failed to get pending events", "worker_id", workerID, "error", err)
		return
	}

	if len(events) == 0 {
		// No events to process
		return
	}

	_ = level.Debug(p.logger).Log("msg", "processing events", "worker_id", workerID, "count", len(events))

	// Process each event
	for _, event := range events {
		if err := p.processEvent(ctx, &event, workerID); err != nil {
			_ = level.Error(p.logger).Log("msg", "failed to process event", "worker_id", workerID, "event_id",
				event.ID, "error", err)
			continue
		}
	}

	_ = level.Debug(p.logger).Log("msg", "finished processing batch", "worker_id", workerID, "processed", len(events))
}

// processEvent processes a single event
func (p *EventProcessor) processEvent(ctx context.Context, event *models.RocketEvent, workerID int) error {
	start := time.Now()

	_ = level.Debug(p.logger).Log("msg", "processing event", "worker_id", workerID, "event_id", event.ID, "type",
		event.MessageType, "channel", event.Channel)

	// Process the event using the service
	err := p.service.ProcessEvent(ctx, event)

	duration := time.Since(start)

	if err != nil {
		_ = level.Error(p.logger).Log("msg", "event processing failed", "worker_id", workerID, "event_id", event.ID,
			"duration", duration, "error", err)
		return err
	}

	_ = level.Info(p.logger).Log("msg", "event processed successfully", "worker_id", workerID, "event_id", event.ID,
		"type", event.MessageType, "duration", duration)
	return nil
}
