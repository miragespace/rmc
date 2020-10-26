package host

import (
	"fmt"
	"net/http"

	resp "github.com/zllovesuki/rmc/response"

	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

// ServiceOptions contains the configuration for Service router
type ServiceOptions struct {
	HostManager *Manager
	Logger      *zap.Logger
}

// Service is the host API router
type Service struct {
	ServiceOptions
}

// NewService will create an instance of the host API router
func NewService(option ServiceOptions) (*Service, error) {
	if option.HostManager == nil {
		return nil, fmt.Errorf("nil HostManager is invalid")
	}
	if option.Logger == nil {
		return nil, fmt.Errorf("nil Logger is invalid")
	}
	return &Service{
		ServiceOptions: option,
	}, nil
}

func (s *Service) listHosts(w http.ResponseWriter, r *http.Request) {
	results, err := s.HostManager.List(r.Context())
	if err != nil {
		s.Logger.Error("Unable to list hosts",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Cannot get the list of instances"))
		return
	}

	type Result struct {
		Name    string `json:"name"`
		IsAlive bool   `json:"isAlive"`
	}
	publicResults := make([]Result, len(results), len(results))
	for i := range results {
		publicResults[i] = Result{
			Name:    results[i].Name,
			IsAlive: results[i].Alive(),
		}
	}

	resp.WriteResponse(w, r, publicResults)
}

// Router will return the routes under host API
func (s *Service) Router() http.Handler {
	r := chi.NewRouter()

	r.Get("/", s.listHosts)

	return r
}
