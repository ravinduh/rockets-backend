# Rockets Backend Service 🚀

A Go-based backend service that tracks rocket state changes through message processing and provides REST APIs for querying rocket data.

## Quick Start

**Prerequisites:**
- Docker and Docker Compose

**Running the service:**
```bash
# Start the service and PostgreSQL database
docker compose up --build

# Or run in background
docker compose up --build -d

# View logs
docker compose logs -f

# Stop the service
docker compose down
```
The service will be available at `http://localhost:8088` with PostgreSQL automatically configured.

## Running Tests

**Prerequisites:**
- Docker and Docker Compose

**Run integration tests:**
```bash
# Run all tests (automatically handles PostgreSQL setup)
./test.sh
```

**What the test script does:**
- Starts PostgreSQL container with Docker Compose
- Waits for database to be ready
- Creates test database and applies schema
- Runs all integration tests against real PostgreSQL
- Cleans up containers automatically

**Test database configuration:**
- Database: `rockets_test`
- Host: `localhost:5432`
- User/Password: `postgres/postgres`
- Environment variables can override defaults (see `testutil/database.go`)


## API Response Format

Download postman collection for easier testing:
[postman collection](./postman_collection.json)

All API responses follow a consistent format:

**Success Response:**
```json
{
  "request_id": "uuid-v4",
  "data": { /* response data */ }
}
```

**Error Response:**
```json
{
  "request_id": "uuid-v4", 
  "error": "error message"
}
```

**Key Points:**
- `request_id`: Always present for request tracing
- `data`: Present only on successful responses (omitted on errors)  
- `error`: Present only on error responses (omitted on success)
- HTTP status codes: 200 (success), 400 (bad request), 404 (not found), etc.


## API Endpoints

**Available Endpoints:**
- `GET /health` - Health check
- `POST /messages` - Ingest rocket messages (async)
- `GET /rockets` - Get all rockets with optional sorting
- `GET /rockets/{id}` - Get specific rocket by channel ID
- `GET /events/{event_id}` - Get event processing status

### Health Check
```
GET /health
Request-Id: optional-custom-uuid (optional header)
```
Returns service health status.

**Success Response:**
```json
{
  "request_id": "uuid-v4",
  "data": {
    "status": "OK",
    "service": "rockets-backend"
  }
}
```

### Process Messages (for rockets test program)
```
POST /messages
Content-Type: application/json
Request-Id: optional-custom-uuid (optional header)

{
  "metadata": {
    "channel": "193270a9-c9cf-404a-8f83-838e71d9ae67",
    "messageNumber": 1,
    "messageTime": "2022-02-02T19:39:05.86337+01:00",
    "messageType": "RocketLaunched"
  },
  "message": {
    "type": "Falcon-9",
    "launchSpeed": 500,
    "mission": "ARTEMIS"
  }
}
```

**Success Response:**
```json
{
  "request_id": "uuid-v4",
  "data": {
    "status": "ingested",
    "event_id": 123
  }
}
```

**Error Response:**
```json
{
  "request_id": "uuid-v4",
  "error": "failed to process message: unknown message type"
}
```

### Get All Rockets
```
GET /rockets?sortBy=type
Request-Id: optional-custom-uuid (optional header)
```
Query parameters:
- `sortBy`: type, speed, mission, status, launchTime, lastUpdated (optional)

**Success Response:**
```json
{
  "request_id": "uuid-v4",
  "data": [
    {
      "id": "193270a9-c9cf-404a-8f83-838e71d9ae67",
      "type": "Falcon-9",
      "currentSpeed": 15000,
      "mission": "ARTEMIS",
      "status": "active",
      "launchTime": "2022-02-02T19:39:05.86337+01:00",
      "lastUpdated": "2022-02-02T19:40:15.86337+01:00"
    }
  ]
}
```

**Error Response:**
```json
{
  "request_id": "uuid-v4",
  "error": "database connection failed"
}
```

### Get Specific Rocket
```
GET /rockets/{id}
Request-Id: optional-custom-uuid (optional header)
```
Returns rocket state by channel ID.

**Success Response:**
```json
{
  "request_id": "uuid-v4",
  "data": {
    "id": "193270a9-c9cf-404a-8f83-838e71d9ae67",
    "type": "Falcon-9",
    "currentSpeed": 15000,
    "mission": "ARTEMIS",
    "status": "active",
    "launchTime": "2022-02-02T19:39:05.86337+01:00",
    "lastUpdated": "2022-02-02T19:40:15.86337+01:00"
  }
}
```

**Error Response (404):**
```json
{
  "request_id": "uuid-v4",
  "error": "rocket not found"
}
```

### Get Event Status
```
GET /events/{event_id}
Request-Id: optional-custom-uuid (optional header)
```
Check the processing status of a specific event.

**Success Response:**
```json
{
  "request_id": "uuid-v4",
  "data": {
    "id": 123,
    "channel": "193270a9-c9cf-404a-8f83-838e71d9ae67",
    "message_number": 1,
    "message_type": "RocketLaunched",
    "status": "processed",
    "received_at": "2022-02-02T19:39:05.86337+01:00",
    "processed_at": "2022-02-02T19:39:06.12345+01:00"
  }
}
```

**Error Response (404):**
```json
{
  "request_id": "uuid-v4",
  "error": "event not found"
}
```


## Testing with Rockets Program

Run the provided rockets test program:
```bash
./rockets launch "http://localhost:8088/messages" --message-delay=500ms --concurrency-level=1
```

### Log Format
All logs include request ID for traceability:
```
level=info requestId=abc-123 msg="event processed successfully" eventId=123 type=RocketLaunched channel=xyz messageNumber=1
```

## Design Descisions

### **Asynchronous Message Processing**
- POST /messages stores events immediately (~1-5ms response)
- Set of workers polls and processes events asynchronously in background


### **Scalability**
- The main reason for the separation of consumption and processing is to be able to scale separately later
- In a dstributed environment, we could use a queing solution like kafka, rabbitMQ [Improvement]
- Workers process events in configurable batches (default: 10 events)
- Configurable worker count (default: 2 workers)
- Configurable worker polling (default: 1 second)
- Failed events marked with status and can create retry workflows if needed [Improvement]
- Events are persisted before processing begins

## Technology stack
- **Go-kit Framework**: Transport layer, endpoints, and service separation
- **PostgreSQL**: Robust database with UUID and JSONB support


## Database Schema

### rockets
- `id` (UUID): Rocket channel/identifier
- `type` (VARCHAR): Rocket type (e.g., "Falcon-9")
- `current_speed` (INTEGER): Current rocket speed
- `mission` (VARCHAR): Current mission
- `status` (VARCHAR): active, exploded
- `explosion_reason` (VARCHAR): Reason if exploded
- `launch_time` (TIMESTAMP): When rocket was launched
- `last_updated` (TIMESTAMP): Last state update
- `last_message_number` (INTEGER): Last processed message number

### rocket_events
- `id` (SERIAL): Event ID
- `channel` (UUID): Rocket channel
- `message_number` (INTEGER): Message sequence number  
- `message_type` (VARCHAR): Type of message (RocketLaunched, etc.)
- `message_data` (JSONB): Raw message payload
- `received_at` (TIMESTAMP): When event was received
- `processed_at` (TIMESTAMP): When event was processed (nullable)
- `status` (VARCHAR): pending, processing, processed, failed
- `error_message` (TEXT): Error details if processing failed (nullable)
- **Unique Constraint**: `(channel, message_number)` prevents duplicate message processing

## Message Types Supported

1. **RocketLaunched**: Initial rocket launch
2. **RocketSpeedIncreased**: Speed increase events
3. **RocketSpeedDecreased**: Speed decrease events
4. **RocketExploded**: Rocket explosion events
5. **RocketMissionChanged**: Mission change events
