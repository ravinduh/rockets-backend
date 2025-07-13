package service

import "github.com/go-kit/kit/log"

type Service interface {
	HealthCheck() interface{}
}

type service struct {
	logger log.Logger
}

func (s service) HealthCheck() interface{} {
	return "OK"
}

// NewService returns a naive, stateless implementation of rockets backend
func NewService(logger log.Logger) Service {
	return &service{
		logger: logger,
	}
}
