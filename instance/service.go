package instance

import (
	"fmt"
	"net/http"

	"github.com/zllovesuki/rmc/auth"
	"go.uber.org/zap"

	"github.com/go-chi/chi"
)

// Options contains the configuration for Service router
type Options struct {
	Auth            *auth.Auth
	InstanceManager *Manager
	Logger          *zap.Logger
}

// Service is the instance API router
type Service struct {
	Option Options
}

func NewService(option Options) (*Service, error) {
	if option.Auth == nil {
		return nil, fmt.Errorf("nil Auth is invalid")
	}
	if option.InstanceManager == nil {
		return nil, fmt.Errorf("nil InstanceManager is invalid")
	}
	if option.Logger == nil {
		return nil, fmt.Errorf("nil Logger is invalid")
	}
	return &Service{
		Option: option,
	}, nil
}

func (s *Service) Router() http.Handler {
	r := chi.NewRouter()

	return r
}
