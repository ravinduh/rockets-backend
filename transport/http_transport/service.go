package http_transport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"rockets-backend/models"
	pkgContext "rockets-backend/pkg/context"
	"rockets-backend/pkg/response"
	"rockets-backend/transport"
	"strconv"

	goKitHttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

func NewHttpService(endpoints transport.Endpoints) http.Handler {
	r := mux.NewRouter()

	// Apply request ID middleware to all routes
	r.Use(requestIDMiddleware)

	// Health check endpoint
	r.Methods("GET").Path("/health").Handler(goKitHttp.NewServer(
		endpoints.HealthCheck,
		decodeEmptyRequest,
		encodeResponse,
		goKitHttp.ServerBefore(extractRequestID),
		goKitHttp.ServerErrorEncoder(encodeErrorResponse),
	))

	// Message processing endpoint (for rockets test program)
	r.Methods("POST").Path("/messages").Handler(goKitHttp.NewServer(
		endpoints.ProcessMessage,
		decodeMessageRequest,
		encodeResponse,
		goKitHttp.ServerBefore(extractRequestID),
		goKitHttp.ServerErrorEncoder(encodeErrorResponse),
	))

	// Get specific rocket
	r.Methods("GET").Path("/rockets/{id}").Handler(goKitHttp.NewServer(
		endpoints.GetRocket,
		decodeGetRocketRequest,
		encodeResponse,
		goKitHttp.ServerBefore(extractRequestID),
		goKitHttp.ServerErrorEncoder(encodeErrorResponse),
	))

	// Get all rockets
	r.Methods("GET").Path("/rockets").Handler(goKitHttp.NewServer(
		endpoints.GetAllRockets,
		decodeGetAllRocketsRequest,
		encodeResponse,
		goKitHttp.ServerBefore(extractRequestID),
		goKitHttp.ServerErrorEncoder(encodeErrorResponse),
	))

	// Get event status
	r.Methods("GET").Path("/events/{id}").Handler(goKitHttp.NewServer(
		endpoints.GetEventStatus,
		decodeGetEventStatusRequest,
		encodeResponse,
		goKitHttp.ServerBefore(extractRequestID),
		goKitHttp.ServerErrorEncoder(encodeErrorResponse),
	))


	return r
}

func decodeEmptyRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

func decodeMessageRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var msg models.IncomingMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func decodeGetRocketRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	return transport.GetRocketRequest{ID: vars["id"]}, nil
}

func decodeGetAllRocketsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	sortBy := r.URL.Query().Get("sortBy")
	return transport.GetAllRocketsRequest{SortBy: sortBy}, nil
}

func decodeGetEventStatusRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	eventIDStr := vars["id"]
	
	eventID, err := strconv.ParseInt(eventIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid event ID: %s", eventIDStr)
	}
	
	return transport.GetEventStatusRequest{EventID: eventID}, nil
}

// requestIDMiddleware adds request ID to the HTTP request context
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for existing Request-Id header
		requestID := r.Header.Get("Request-Id")
		if requestID == "" {
			// Generate new request ID if not provided
			requestID = pkgContext.GenerateRequestID()
		}
		
		// Add request ID to context
		ctx := pkgContext.WithRequestID(r.Context(), requestID)
		r = r.WithContext(ctx)
		
		// Add request ID to response header
		w.Header().Set("Request-Id", requestID)
		
		next.ServeHTTP(w, r)
	})
}

// extractRequestID extracts request ID from HTTP request and adds to go-kit context
func extractRequestID(ctx context.Context, r *http.Request) context.Context {
	if requestID := pkgContext.GetRequestID(r.Context()); requestID != "" {
		return pkgContext.WithRequestID(ctx, requestID)
	}
	return ctx
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, responseData interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	
	requestID := pkgContext.GetRequestID(ctx)
	apiResponse := response.New(requestID, responseData, nil)
	
	return json.NewEncoder(w).Encode(apiResponse)
}

func encodeErrorResponse(ctx context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	
	// Set appropriate HTTP status based on error type
	statusCode := http.StatusBadRequest
	if err.Error() == "rocket not found" {
		statusCode = http.StatusNotFound
	}
	w.WriteHeader(statusCode)
	
	requestID := pkgContext.GetRequestID(ctx)
	apiResponse := response.New(requestID, nil, err)
	
	json.NewEncoder(w).Encode(apiResponse)
}
