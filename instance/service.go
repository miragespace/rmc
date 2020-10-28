package instance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/zllovesuki/rmc/auth"
	"github.com/zllovesuki/rmc/host"
	resp "github.com/zllovesuki/rmc/response"
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
	LifecycleManager    LifecycleManager
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
	if option.LifecycleManager == nil {
		return nil, fmt.Errorf("nil LifecycleManager is invalid")
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
	claims := ctx.Value(auth.Context).(*auth.Claims)
	instanceID := chi.URLParam(r, "id")

	logger := s.Logger.With(
		zap.String("CustomerID", claims.ID),
		zap.String("InstanceID", instanceID),
	)

	opt := GetOption{
		InstanceID:  instanceID,
		WithHistory: true,
	}
	inst, err := s.InstanceManager.Get(ctx, opt)
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
	before := r.URL.Query().Get("before")

	var parsedTime time.Time
	if before != "" {
		var err error
		parsedTime, err = time.Parse(time.RFC3339Nano, before)
		if err != nil {
			resp.WriteError(w, r, resp.ErrBadRequest().AddMessages("Invalid before param"))
			return
		}
	}

	logger := s.Logger.With(
		zap.String("CustomerID", claims.ID),
	)

	opt := ListOption{
		CustomerID:        claims.ID,
		IncludeTerminated: all,
		Before:            parsedTime,
		Limit:             10,
	}
	results, err := s.InstanceManager.List(ctx, opt)
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

	lambda := func(current *Instance, desired *Instance) (shouldSave bool, respError interface{}) {
		if current == nil || current.CustomerID != claims.ID || current.Status != StatusActive {
			respError = resp.ErrNotFound().AddMessages("Cannot find instance with specific ID")
			return
		}

		var nextState string
		switch req.Action {
		case "Start":
			if current.State != StateStopped {
				respError = resp.ErrBadRequest().AddMessages("Instance not in 'Stopped' state")
				return
			}
			nextState = StateStarting
		case "Stop":
			if current.State != StateRunning {
				respError = resp.ErrBadRequest().AddMessages("Instance not in 'Running' state")
				return
			}
			nextState = StateStopping
		default:
			respError = resp.ErrBadRequest().AddMessages("Unknown action")
			return
		}

		// trigger history insertion
		desired.PreviousState = current.State
		desired.State = nextState
		shouldSave = true
		return
	}

	lambdaResult := s.InstanceManager.LambdaUpdate(ctx, instanceID, lambda)

	if lambdaResult.ReturnValue != nil {
		resp.WriteError(w, r, lambdaResult.ReturnValue.(*resp.Error))
		return
	}

	if lambdaResult.TxError != nil {
		logger.Error("Unable to update instance status",
			zap.Error(lambdaResult.TxError),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to update Instance status"))
		return
	}

	go func(inst *Instance) {
		opt := LifecycleOption{
			HostName:   inst.HostName,
			InstanceID: inst.ID,
			Parameters: nil,
		}
		var err error
		switch inst.State {
		case StateStopping:
			err = s.LifecycleManager.Stop(opt)
		case StateStarting:
			err = s.LifecycleManager.Start(opt)
		}
		if err != nil {
			logger.Error("Unable to send control request",
				zap.Error(err),
				zap.String("HostName", inst.HostName),
			)
			// fail through: as long as database state is consistent, manual mediation is possible
		}
	}(lambdaResult.Instance)

	// background task should handle the usage update, if START/STOP was successful

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

	lambda := func(current *Instance, desired *Instance) (shouldSave bool, respError interface{}) {
		if current == nil || current.CustomerID != claims.ID || current.Status != StatusActive {
			respError = resp.ErrNotFound().AddMessages("Cannot find instance with specific ID")
			return
		}

		if current.State != StateStopped {
			respError = resp.ErrBadRequest().AddMessages("Instance not in 'Stopped' state")
			return
		}

		// trigger history insertion
		desired.PreviousState = current.State
		desired.State = StateRemoving
		shouldSave = true
		return
	}

	lambdaResult := s.InstanceManager.LambdaUpdate(ctx, instanceID, lambda)

	if lambdaResult.ReturnValue != nil {
		resp.WriteError(w, r, lambdaResult.ReturnValue.(*resp.Error))
		return
	}

	if lambdaResult.TxError != nil {
		logger.Error("Unable to delete instance",
			zap.Error(lambdaResult.TxError),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to delete Instance"))
		return
	}

	go func(inst *Instance) {
		opt := LifecycleOption{
			HostName:   inst.HostName,
			InstanceID: inst.ID,
			Parameters: &inst.Parameters,
		}
		if err := s.LifecycleManager.Delete(opt); err != nil {
			logger.Error("Unable to send DELETE provision request",
				zap.Error(err),
				zap.String("HostName", inst.HostName),
			)
			// fail through: as long as database state is consistent, manual mediation is possible
		}
	}(lambdaResult.Instance)

	// background task should handle cancelling subscription, if DELETE was successful

	w.WriteHeader(http.StatusNoContent)
}

// NewInstanceRequest contains the request from client to provision a new instance.
// A valid subscription must be set up before a new instance can be provisioned
type NewInstanceRequest struct {
	ServerVersion  string `json:"serverVersion"` // e.g. 1.16.3
	ServerEdition  string `json:"serverEdition"` // "java" or "bedrock"
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

	subOpt := subscription.GetOption{
		CustomerID:     claims.ID,
		SubscriptionID: req.SubscriptionID,
	}
	sub, err := s.SubscriptionManager.Get(ctx, subOpt)
	if err != nil {
		logger.Error("Unable to verify subscription validity",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to create Instance"))
		return
	}
	if sub == nil || sub.State != subscription.StateActive {
		resp.WriteError(w, r, resp.ErrBadRequest().AddMessages("Invalid Subscription: non-existent or inactive"))
		return
	}

	logger = logger.With(zap.String("SubscriptionID", req.SubscriptionID))

	// make sure user is not double dipping
	opt := GetOption{
		SubscriptionID: req.SubscriptionID,
		WithHistory:    false,
	}
	existingInst, err := s.InstanceManager.Get(ctx, opt)
	if err != nil {
		logger.Error("Unable to verify duplicate subscription existence",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to create Instance"))
		return
	}
	if existingInst != nil {
		resp.WriteError(w, r, resp.ErrConflict().AddMessages("Duplicate subscription found"))
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
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to create Instance", "No suitable host available"))
		return
	}

	logger = logger.With(zap.String("HostName", host.Name))

	// TODO: validate server versions/java or not
	plan, ok := s.SubscriptionManager.GetDefinedPlanByID(sub.PlanID)
	if !ok {
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to create Instance", "Subscription is invalid or is tied to a retired Plan"))
		return
	}
	instanceParams := plan.Parameters
	instanceParams["ServerVersion"] = req.ServerVersion
	instanceParams["ServerEdition"] = req.ServerEdition

	inst := Instance{
		ID:             uuid.New().String(),
		CustomerID:     claims.ID,
		SubscriptionID: req.SubscriptionID,
		HostName:       host.Name,
		Parameters:     instanceParams,
		PreviousState:  StateUnknown,
		State:          StateProvisioning,
		Status:         StatusActive,
	}

	if err := s.InstanceManager.Create(ctx, &inst); err != nil {
		logger.Error("Unable to create instance",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to create Instance"))
		return
	}

	go func() {
		opt := LifecycleOption{
			HostName:   host.Name,
			InstanceID: inst.ID,
			Parameters: &inst.Parameters,
		}
		if err := s.LifecycleManager.Create(opt); err != nil {
			logger.Error("Unable to send CREATE provision request",
				zap.Error(err),
				zap.String("HostName", inst.HostName),
			)
			// fail through: as long as database state is consistent, manual mediation is possible
		}
	}()

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
