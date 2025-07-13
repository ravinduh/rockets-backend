package http_transport

import (
	"context"
	"encoding/json"
	"net/http"
	"rockets-backend/transport"

	goKitHttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

func NewHttpService(endpoints transport.Endpoints) http.Handler {
	r := mux.NewRouter()

	r.Methods("GET").Path("/health").Handler(goKitHttp.NewServer(
		endpoints.HealthCheck,
		decodeRequest,
		encodeResponse,
	))

	return r

}

func decodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Add("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(response)
}
