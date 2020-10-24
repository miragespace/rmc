package instance

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zllovesuki/rmc/auth"
	"github.com/zllovesuki/rmc/host"
	"github.com/zllovesuki/rmc/spec/broker"
	"github.com/zllovesuki/rmc/spec/protocol"
	"github.com/zllovesuki/rmc/subscription"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ServiceOptions contains the configuration for Service router
type ServiceOptions struct {
	SubscriptionManager *subscription.Manager
	HostManager         *host.Manager
	InstanceManager     *Manager
	Producer            broker.Producer
	Logger              *zap.Logger
}

// Service is the instance API router
type Service struct {
	ServiceOptions
}

// NewService will create an instance of the instance API router
func NewService(option ServiceOptions) (*Service, error) {
	if option.SubscriptionManager == nil {
		return nil, fmt.Errorf("nil SubscriptionManager is invalid")
	}
	if option.HostManager == nil {
		return nil, fmt.Errorf("nil HostManager is invalid")
	}
	if option.InstanceManager == nil {
		return nil, fmt.Errorf("nil InstanceManager is invalid")
	}
	if option.Producer == nil {
		return nil, fmt.Errorf("nil Broker is invalid")
	}
	if option.Logger == nil {
		return nil, fmt.Errorf("nil Logger is invalid")
	}
	return &Service{
		ServiceOptions: option,
	}, nil
}

func (s *Service) getInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	instanceID := chi.URLParam(r, "id")
	claims := ctx.Value(auth.Context).(*auth.Claims)

	logger := s.Logger.With(
		zap.String("CustomerID", claims.ID),
		zap.String("InstanceID", instanceID),
	)

	inst, err := s.InstanceManager.GetByID(ctx, instanceID)
	if err != nil {
		logger.Error("Unable to query instance",
			zap.Error(err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if inst == nil || inst.CustomerID != claims.ID || inst.Status != StatusActive {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(inst)
}

// ControlRequest contains the request from client to control an existing instance.
type ControlRequest struct {
	Action string `json:"action"`
}

func (s *Service) controlInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	instanceID := chi.URLParam(r, "id")
	claims := ctx.Value(auth.Context).(*auth.Claims)

	logger := s.Logger.With(
		zap.String("CustomerID", claims.ID),
		zap.String("InstanceID", instanceID),
	)

	inst, err := s.InstanceManager.GetByID(ctx, instanceID)
	if err != nil {
		logger.Error("Unable to query instance",
			zap.Error(err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if inst == nil || inst.CustomerID != claims.ID || inst.Status != StatusActive {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	logger = logger.With(
		zap.String("SubscriptionID", inst.SubscriptionID),
		zap.String("HostName", inst.HostName),
	)

	var req ControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var action protocol.ControlRequest_ControlAction
	var nextState string
	switch req.Action {
	case "Start":
		if inst.State != StateStopped {
			http.Error(w, "instance not in stopped state", http.StatusBadRequest)
			return
		}
		action = protocol.ControlRequest_START
		nextState = StateStarting
	case "Stop":
		if inst.State != StateRunning {
			http.Error(w, "instance not in running state", http.StatusBadRequest)
			return
		}
		action = protocol.ControlRequest_STOP
		nextState = StateStopping
	default:
		http.Error(w, "unknown action", http.StatusBadRequest)
		return
	}

	inst.State = nextState

	if err := s.InstanceManager.Update(ctx, inst); err != nil {
		logger.Error("Unable to update instance status",
			zap.Error(err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := s.Producer.SendControlRequest(&host.Host{
		Name: inst.HostName,
	}, &protocol.ControlRequest{
		Instance: &protocol.Instance{
			InstanceID: inst.ID,
		},
		Action: action,
	}); err != nil {
		logger.Error("Unable to send control request",
			zap.Error(err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (s *Service) deleteInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	instanceID := chi.URLParam(r, "id")
	claims := ctx.Value(auth.Context).(*auth.Claims)

	logger := s.Logger.With(
		zap.String("CustomerID", claims.ID),
		zap.String("InstanceID", instanceID),
	)

	inst, err := s.InstanceManager.GetByID(ctx, instanceID)
	if err != nil {
		logger.Error("Unable to query instance",
			zap.Error(err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if inst == nil || inst.CustomerID != claims.ID || inst.Status != StatusActive {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if inst.State != StateStopped {
		http.Error(w, "cannot delete when it is running", http.StatusBadRequest)
		return
	}

	logger = logger.With(
		zap.String("SubscriptionID", inst.SubscriptionID),
		zap.String("HostName", inst.HostName),
	)

	host, err := s.HostManager.GetHostByName(ctx, inst.HostName)
	if err != nil {
		logger.Error("Unable to get instance Host",
			zap.Error(err),
		)
	}

	if err := s.Producer.SendProvisionRequest(host, &protocol.ProvisionRequest{
		Instance: &protocol.Instance{
			InstanceID: inst.ID,
		},
		Action: protocol.ProvisionRequest_DELETE,
	}); err != nil {
		logger.Error("Unable to send provision DELETE request",
			zap.Error(err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := s.SubscriptionManager.CancelSubscription(ctx, inst.SubscriptionID); err != nil {
		logger.Error("Unable to cancel Stripe Subscription",
			zap.Error(err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	inst.State = StateStopping
	inst.Status = StatusTerminated

	if err := s.InstanceManager.Update(ctx, inst); err != nil {
		logger.Error("Unable to update instance status for DELETE",
			zap.Error(err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// NewInstanceRequest contains the request from client to provision a new instance.
// A valid subscription must be set up before a new instance can be provisioned
type NewInstanceRequest struct {
	ServerVersion  string `json:"serverVersion"`
	IsJavaEdition  bool   `json:"isJavaEdition"`
	SubscriptionID string `json:"subscriptionId"`
}

func (s *Service) newInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims := ctx.Value(auth.Context).(*auth.Claims)

	logger := s.Logger.With(zap.String("CustomerID", claims.ID))

	var req NewInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: validate server versions/java or not

	valid, err := s.SubscriptionManager.ValidSubscription(ctx, claims.ID, req.SubscriptionID)
	if err != nil {
		logger.Error("Unable to verify subscription validity",
			zap.Error(err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if !valid {
		http.Error(w, "not authorized", http.StatusUnauthorized)
		return
	}

	logger = logger.With(zap.String("SubscriptionID", req.SubscriptionID))

	// make sure user is not double dipping
	existingInst, err := s.InstanceManager.GetBySubscriptionId(ctx, req.SubscriptionID)
	if err != nil {
		logger.Error("Unable to verify duplicate subscription existence",
			zap.Error(err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if existingInst != nil {
		http.Error(w, "duplicate subscription", http.StatusConflict)
		return
	}

	uuid, err := uuid.NewRandom()
	if err != nil {
		logger.Error("Unable to get a random UUID",
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

	if host == nil {
		// TODO: make it more obvious to user and to operator
		logger.Error("No available host for provisioning")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	logger = logger.With(zap.String("HostName", host.Name))

	inst := Instance{
		ID:             uuid.String(),
		CustomerID:     claims.ID,
		SubscriptionID: req.SubscriptionID,
		HostName:       host.Name,
		ServerVersion:  req.ServerVersion,
		IsJavaEdition:  req.IsJavaEdition,
		State:          StateProvisioning,
		Status:         StatusActive,
	}

	if err := s.InstanceManager.Create(ctx, &inst); err != nil {
		logger.Error("Unable to create instance",
			zap.Error(err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := s.Producer.SendProvisionRequest(host, &protocol.ProvisionRequest{
		Instance: &protocol.Instance{
			InstanceID:    inst.ID,
			Version:       req.ServerVersion,
			IsJavaEdition: req.IsJavaEdition,
		},
		Action: protocol.ProvisionRequest_CREATE,
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
	r := chi.NewRouter().With(auth.ClaimCheck(s.Logger))

	r.Post("/", s.newInstance)
	r.Get("/{id}", s.getInstance)
	r.Post("/{id}", s.controlInstance)
	r.Delete("/{id}", s.deleteInstance)

	return r
}
