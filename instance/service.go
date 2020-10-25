package instance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/zllovesuki/rmc/auth"
	"github.com/zllovesuki/rmc/host"
	resp "github.com/zllovesuki/rmc/response"
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
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Cannot get details about the instance"))
		return
	}

	if inst == nil || inst.CustomerID != claims.ID || inst.Status != StatusActive {
		resp.WriteError(w, r, resp.ErrNotFound().AddMessages("Cannot find instance with specific ID"))
		return
	}

	resp.WriteResponse(w, r, inst)
}

func (s *Service) listInstances(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := ctx.Value(auth.Context).(*auth.Claims)
	all := r.URL.Query().Get("all") != ""

	logger := s.Logger.With(
		zap.String("CustomerID", claims.ID),
	)

	results, err := s.InstanceManager.List(ctx, claims.ID, all)
	if err != nil {
		logger.Error("Unable to list instances by customer id",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Cannot get the list of instances"))
		return
	}

	resp.WriteResponse(w, r, results)
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

	var req ControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.WriteError(w, r, resp.ErrInvalidJson())
		return
	}

	lambda := func(currenState *Instance, desiredState *Instance) (shouldSave bool) {
		if currenState == nil || currenState.CustomerID != claims.ID || currenState.Status != StatusActive {
			resp.WriteError(w, r, resp.ErrNotFound().AddMessages("Cannot find instance with specific ID"))
			return
		}
		logger = logger.With(
			zap.String("HostName", currenState.HostName),
		)

		var action protocol.ControlRequest_ControlAction
		var nextState string
		switch req.Action {
		case "Start":
			if currenState.State != StateStopped {
				resp.WriteError(w, r, resp.ErrBadRequest().AddMessages("Instance not in 'Stopped' state"))
				return
			}
			action = protocol.ControlRequest_START
			nextState = StateStarting
		case "Stop":
			if currenState.State != StateRunning {
				resp.WriteError(w, r, resp.ErrBadRequest().AddMessages("Instance not in 'Running' state"))
				return
			}
			action = protocol.ControlRequest_STOP
			nextState = StateStopping
		default:
			resp.WriteError(w, r, resp.ErrBadRequest().AddMessages("Unknown action"))
			return
		}

		if err := s.Producer.SendControlRequest(&host.Host{
			Name: currenState.HostName,
		}, &protocol.ControlRequest{
			Instance: &protocol.Instance{
				InstanceID: currenState.ID,
			},
			Action: action,
		}); err != nil {
			logger.Error("Unable to send control request",
				zap.Error(err),
			)
			resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to update Instance status"))
			return
		}

		desiredState.State = nextState
		shouldSave = true
		return
	}

	inst, err := s.InstanceManager.LambdaUpdate(ctx, instanceID, lambda)

	if err != nil {
		logger.Error("Unable to update instance status",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to update Instance status"))
		return
	}

	if inst == nil {
		// response was already sent in lambda
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

	lambda := func(currentState *Instance, desireState *Instance) (shouldSave bool) {
		if currentState == nil || currentState.CustomerID != claims.ID || currentState.Status != StatusActive {
			resp.WriteError(w, r, resp.ErrNotFound().AddMessages("Cannot find instance with specific ID"))
			return
		}

		if currentState.State != StateStopped {
			resp.WriteError(w, r, resp.ErrBadRequest().AddMessages("Instance not in 'Stopped' state"))
			return
		}
		logger = logger.With(
			zap.String("HostName", currentState.HostName),
		)

		host, err := s.HostManager.GetHostByName(ctx, currentState.HostName)
		if err != nil {
			logger.Error("Unable to get instance Host",
				zap.Error(err),
			)
			resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to delete Instance"))
			return
		}

		if err := s.Producer.SendProvisionRequest(host, &protocol.ProvisionRequest{
			Instance: &protocol.Instance{
				InstanceID: currentState.ID,
			},
			Action: protocol.ProvisionRequest_DELETE,
		}); err != nil {
			logger.Error("Unable to send DELETE provision request",
				zap.Error(err),
			)
			resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to delete Instance"))
			return
		}

		desireState.Status = StatusTerminated
		shouldSave = true
		return
	}

	inst, err := s.InstanceManager.LambdaUpdate(ctx, instanceID, lambda)
	if err != nil {
		logger.Error("Unable to delete instance",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to delete Instance"))
		return
	}

	if inst == nil {
		// response was already sent in lambda
		return
	}

	if err := s.SubscriptionManager.CancelSubscription(ctx, inst.SubscriptionID); err != nil {
		logger.With(zap.String("SubscriptionID", inst.SubscriptionID)).
			Error("Unable to cancel Stripe Subscription",
				zap.Error(err),
			)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to delete Instance"))
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
		resp.WriteError(w, r, resp.ErrInvalidJson())
		return
	}

	// TODO: validate server versions/java or not

	valid, err := s.SubscriptionManager.ValidSubscription(ctx, claims.ID, req.SubscriptionID)
	if err != nil {
		logger.Error("Unable to verify subscription validity",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to create Instance"))
		return
	}
	if !valid {
		resp.WriteError(w, r, resp.ErrConflict().WithMessage("Duplicate subscription found"))
		return
	}

	logger = logger.With(zap.String("SubscriptionID", req.SubscriptionID))

	// make sure user is not double dipping
	existingInst, err := s.InstanceManager.GetBySubscriptionId(ctx, req.SubscriptionID)
	if err != nil {
		logger.Error("Unable to verify duplicate subscription existence",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to create Instance"))
		return
	}
	if existingInst != nil {
		resp.WriteError(w, r, resp.ErrConflict().WithMessage("Duplicate subscription found"))
		return
	}

	uuid, err := uuid.NewRandom()
	if err != nil {
		logger.Error("Unable to get a random UUID",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to create Instance"))
		return
	}

	host, err := s.HostManager.NextAvailableHost(ctx)
	if err != nil {
		logger.Error("Unable to find next available host",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to create Instance"))
		return
	}

	if host == nil {
		// TODO: make it more obvious to user and to operator
		logger.Error("No available host for provisioning")
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to create Instance"))
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
		CreatedAt:      time.Now(),
	}

	// even if the user tries to race condition newInstance, uniqueIndex on SubscriptionID should prevent it
	if err := s.InstanceManager.Create(ctx, &inst); err != nil {
		logger.Error("Unable to create instance",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to create Instance"))
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
		logger.Error("Unable to send CREATE provision request",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to create Instance"))
		return
	}

	resp.WriteResponse(w, r, inst)
}

// Router will return the routes under instance API
func (s *Service) Router() http.Handler {
	r := chi.NewRouter()

	r.Get("/", s.listInstances)
	r.Post("/", s.newInstance)
	r.Get("/{id}", s.getInstance)
	r.Post("/{id}", s.controlInstance)
	r.Delete("/{id}", s.deleteInstance)

	return r
}
