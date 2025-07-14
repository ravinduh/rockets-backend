package context

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const RequestIDKey contextKey = "request_id"

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// GenerateRequestID creates a new UUID for request tracking
func GenerateRequestID() string {
	return uuid.New().String()
}

// GetOrGenerateRequestID gets request ID from context or generates a new one
func GetOrGenerateRequestID(ctx context.Context) string {
	if requestID := GetRequestID(ctx); requestID != "" {
		return requestID
	}
	return GenerateRequestID()
}