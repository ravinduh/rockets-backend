package transport

import (
	"context"
	"fmt"
	"rockets-backend/models"
	"rockets-backend/service"

	"github.com/go-kit/kit/endpoint"
)

type Endpoints struct {
	HealthCheck    endpoint.Endpoint
	ProcessMessage endpoint.Endpoint
	GetRocket      endpoint.Endpoint
	GetAllRockets  endpoint.Endpoint
	GetEventStatus endpoint.Endpoint
}

func MakeEndpoints(svc service.Service) Endpoints {
	return Endpoints{
		HealthCheck:    MakeHealthCheckEndpoint(svc),
		ProcessMessage: MakeProcessMessageEndpoint(svc),
		GetRocket:      MakeGetRocketEndpoint(svc),
		GetAllRockets:  MakeGetAllRocketsEndpoint(svc),
		GetEventStatus: MakeGetEventStatusEndpoint(svc),
	}
}

func MakeHealthCheckEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return svc.HealthCheck(), nil
	}
}

func MakeProcessMessageEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(models.IncomingMessage)
		event, err := svc.IngestMessage(ctx, req)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":   "ingested",
			"event_id": event.ID,
		}, nil
	}
}

type GetRocketRequest struct {
	ID string `json:"id"`
}

func MakeGetRocketEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetRocketRequest)
		rocket, err := svc.GetRocket(ctx, req.ID)
		if err != nil {
			return nil, err
		}
		if rocket == nil {
			return nil, fmt.Errorf("rocket not found")
		}
		return rocket, nil
	}
}

type GetAllRocketsRequest struct {
	SortBy string `json:"sortBy"`
}

func MakeGetAllRocketsEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetAllRocketsRequest)
		rockets, err := svc.GetAllRockets(ctx, req.SortBy)
		if err != nil {
			return nil, err
		}
		return rockets, nil
	}
}

type GetEventStatusRequest struct {
	EventID int64 `json:"event_id"`
}

func MakeGetEventStatusEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetEventStatusRequest)
		event, err := svc.GetEventStatus(ctx, req.EventID)
		if err != nil {
			return nil, err
		}
		if event == nil {
			return nil, fmt.Errorf("event not found")
		}
		return event, nil
	}
}
