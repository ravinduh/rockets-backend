-- Rockets database schema
-- Run this to set up the database tables

-- Table to store current rocket state
CREATE TABLE IF NOT EXISTS rockets (
    id UUID PRIMARY KEY,
    type VARCHAR(100) NOT NULL,
    current_speed INTEGER NOT NULL DEFAULT 0,
    mission VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, exploded
    explosion_reason VARCHAR(255) NULL,
    launch_time TIMESTAMP NOT NULL,
    last_updated TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_message_number INTEGER NOT NULL DEFAULT 0
);

-- Table for async event processing
CREATE TABLE IF NOT EXISTS rocket_events (
    id SERIAL PRIMARY KEY,
    channel UUID NOT NULL,
    message_number INTEGER NOT NULL,
    message_type VARCHAR(50) NOT NULL,
    message_data JSONB NOT NULL,
    received_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    error_message TEXT NULL,
    UNIQUE(channel, message_number)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_rockets_status ON rockets(status);
CREATE INDEX IF NOT EXISTS idx_rockets_last_updated ON rockets(last_updated);
CREATE INDEX IF NOT EXISTS idx_rockets_type ON rockets(type);
CREATE INDEX IF NOT EXISTS idx_rocket_events_status ON rocket_events(status);
CREATE INDEX IF NOT EXISTS idx_rocket_events_channel ON rocket_events(channel);
CREATE INDEX IF NOT EXISTS idx_rocket_events_received_at ON rocket_events(received_at);