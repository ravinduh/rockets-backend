#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to cleanup on exit
cleanup() {
    print_status "Cleaning up..."
    docker compose down -v --remove-orphans 2>/dev/null || true
}

# Set trap to cleanup on script exit
trap cleanup EXIT

# Check if Docker is running
if ! docker info >/dev/null 2>&1; then
    print_error "Docker is not running. Please start Docker and try again."
    exit 1
fi

# Check if docker compose is available
if ! command -v docker compose &> /dev/null; then
    print_error "docker compose is not installed. Please install docker compose and try again."
    exit 1
fi

print_status "Starting PostgreSQL for tests..."

# Stop any existing containers
docker compose down -v --remove-orphans 2>/dev/null || true

# Start PostgreSQL service
if ! docker compose up -d postgres; then
    print_error "Failed to start PostgreSQL container"
    exit 1
fi

print_status "Waiting for PostgreSQL to be ready..."

# Wait for PostgreSQL to be ready
max_attempts=30
attempt=1

while [ $attempt -le $max_attempts ]; do
    if docker compose exec -T postgres pg_isready -U postgres >/dev/null 2>&1; then
        print_status "PostgreSQL is ready!"
        break
    fi
    
    if [ $attempt -eq $max_attempts ]; then
        print_error "PostgreSQL failed to start within 30 seconds"
        exit 1
    fi
    
    print_status "Waiting for PostgreSQL... (attempt $attempt/$max_attempts)"
    sleep 1
    ((attempt++))
done

# Set test environment variables
export TEST_DB_HOST=localhost
export TEST_DB_PORT=5432
export TEST_DB_USER=postgres
export TEST_DB_PASSWORD=postgres
export TEST_DB_NAME=rockets_test
export TEST_DB_SSLMODE=disable

print_status "Creating test database..."

# Create test database
docker compose exec -T postgres psql -U postgres -c "DROP DATABASE IF EXISTS rockets_test;" 2>/dev/null || true
docker compose exec -T postgres psql -U postgres -c "CREATE DATABASE rockets_test;" 

# Apply schema to test database
docker compose exec -T postgres psql -U postgres -d rockets_test < schema.sql

print_status "Running integration tests..."

# Run the tests
if go test -v ./... -count=1; then
    print_status "All tests passed!"
    exit_code=0
else
    print_error "Tests failed!"
    exit_code=1
fi

print_status "Test run completed."
exit $exit_code