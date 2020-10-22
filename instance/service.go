package instance

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zllovesuki/rmc/auth"
	"github.com/zllovesuki/rmc/broker"
	"github.com/zllovesuki/rmc/host"
	"github.com/zllovesuki/rmc/spec"
	"go.uber.org/zap"

	"github.com/go-chi/chi"
)

// Options contains the configuration for Service router
type Options struct {
	Auth            *auth.Auth
	HostManager     *host.Manager
	InstanceManager *Manager
	Broker          broker.Broker
	Logger          *zap.Logger
}

// Service is the instance API router
type Service struct {
	Options
}

// NewService will create an instance of the instance API router
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
	if option.Broker == nil {
		return nil, fmt.Errorf("nil Broker is invalid")
	}
	if option.Logger == nil {
		return nil, fmt.Errorf("nil Logger is invalid")
	}
	return &Service{
		Options: option,
	}, nil
}

func (s *Service) deleteInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	instanceID := chi.URLParam(r, "id")
	logger := s.Logger.With(zap.String("InstanceID", instanceID))

	claims, ok := ctx.Value(auth.Context).(*auth.Claims)
	if !ok {
		logger.Error("Context has no Claims")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	inst, err := s.InstanceManager.GetByID(ctx, instanceID)
	if err != nil {
		logger.Error("Unable to query instance",
			zap.Error(err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if inst.CustomerID != claims.ID || inst == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	logger = logger.With(zap.String("HostName", inst.HostName))

	host, err := s.HostManager.GetHostByName(ctx, inst.HostName)
	if err != nil {
		logger.Error("Unable to get instance Host",
			zap.Error(err),
		)
	}

	if err := s.Broker.SendProvisionRequest(host, &spec.ProvisionRequest{
		Instance: &spec.Instance{
			InstanceID: inst.ID,
		},
		Action: spec.ProvisionRequest_DELETE,
	}); err != nil {
		logger.Error("Unable to send provision DELETE request",
			zap.Error(err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// TODO: Cancel Stripe Subscription

	w.WriteHeader(http.StatusNoContent)
}

type NewInstanceRequest struct {
	ServerVersion string `json:"serverVersion"`
	IsJavaEdition bool   `json:"isJavaEdition"`
	// TODO: figure out how the Stripe API works in the front end
}

func (s *Service) newInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims, ok := ctx.Value(auth.Context).(*auth.Claims)
	if !ok {
		s.Logger.Error("Context has no Claims")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	logger := s.Logger.With(zap.String("CustomerID", claims.ID))

	var req NewInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: Validate user request
	// TODO: PriceID

	inst := Instance{
		ID:             "1234ewrekwr", // TODO: UUID generator
		CustomerID:     claims.ID,
		SubscriptionID: "to-be-populated", // TODO: Stripe
		ServerVersion:  req.ServerVersion,
		IsJavaEdition:  req.IsJavaEdition,
		State:          "Provisioning", // TODO: make enum
		Status:         "Active",       // TODO: make enum
	}

	if err := s.InstanceManager.NewInstance(ctx, &inst); err != nil {
		logger.Error("Unable to create instance",
			zap.Error(err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	host, err := s.HostManager.NextAvailableHost(ctx)
	if err != nil {
		logger.Error("Unable to find next available host",
			zap.Error(err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	logger = logger.With(zap.String("HostName", host.Name))

	if err := s.Broker.SendProvisionRequest(host, &spec.ProvisionRequest{
		Instance: &spec.Instance{
			Version:       req.ServerVersion,
			IsJavaEdition: req.IsJavaEdition,
		},
		Action: spec.ProvisionRequest_CREATE,
	}); err != nil {
		logger.Error("Unable to send provision CREATE request",
			zap.Error(err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(inst)

}

// Router will return the routes under instance API
func (s *Service) Router() http.Handler {
	r := chi.NewRouter()

	r.Post("/", s.newInstance)
	r.Delete("/{id}", s.deleteInstance)
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "this is a test for jwt token.")
	})

	return r
}
