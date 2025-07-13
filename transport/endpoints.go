package transport

import (
	"context"
	"rockets-backend/service"

	"github.com/go-kit/kit/endpoint"
)

type Endpoints struct {
	HealthCheck endpoint.Endpoint
}

func MakeEndpoints(svc service.Service) Endpoints {
	return Endpoints{
		HealthCheck: MakeHealthCheckEndpoint(svc),
	}
}

func MakeHealthCheckEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return svc.HealthCheck(), nil
	}
}
