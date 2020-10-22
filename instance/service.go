package instance

import (
	"fmt"
	"net/http"

	"github.com/zllovesuki/rmc/auth"
	"github.com/zllovesuki/rmc/host"
	"go.uber.org/zap"

	"github.com/go-chi/chi"
)

// Options contains the configuration for Service router
type Options struct {
	Auth            *auth.Auth
	HostManager     *host.Manager
	InstanceManager *Manager
	Logger          *zap.Logger
}

// Service is the instance API router
type Service struct {
	Options
}

func NewService(option Options) (*Service, error) {
	if option.Auth == nil {
		return nil, fmt.Errorf("nil Auth is invalid")
	}
	if option.HostManager == nil {
		return nil, fmt.Errorf("nil HostManager is invalid")
	}
	if option.InstanceManager == nil {
		return nil, fmt.Errorf("nil InstanceManager is invalid")
	}
	if option.Logger == nil {
		return nil, fmt.Errorf("nil Logger is invalid")
	}
	return &Service{
		Options: option,
	}, nil
}

func (s *Service) Router() http.Handler {
	r := chi.NewRouter()

	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "this is a test for jwt token.")
	})

	return r
}
