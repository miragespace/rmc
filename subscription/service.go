package subscription

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/zllovesuki/rmc/auth"
	resp "github.com/zllovesuki/rmc/response"

	"github.com/go-chi/chi"
	"github.com/stripe/stripe-go/v71"
	"go.uber.org/zap"
)

type ServiceOptions struct {
	SubscriptionManager *Manager
	Logger              *zap.Logger
}

type Service struct {
	ServiceOptions
}

func NewService(option ServiceOptions) (*Service, error) {
	if option.SubscriptionManager == nil {
		return nil, fmt.Errorf("nil SubscriptionManager is invalid")
	}
	if option.Logger == nil {
		return nil, fmt.Errorf("nil Logger is invalid")
	}
	return &Service{
		ServiceOptions: option,
	}, nil
}

type PaymentSetupRequest struct {
	PaymentMethodID string `json:"paymentMethodId"`
}

func (s *Service) setupPayment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := ctx.Value(auth.Context).(*auth.Claims)

	logger := s.Logger.With(zap.String("CustomerID", claims.ID))

	var req PaymentSetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.WriteError(w, r, resp.ErrInvalidJson())
		return
	}

	if len(req.PaymentMethodID) < 10 { // the number "10" here is arbitary
		resp.WriteError(w, r, resp.ErrBadRequest().AddMessages("Invalid PaymentMethodID"))
		return
	}

	logger = logger.With(zap.String("PaymentMethodID", req.PaymentMethodID))

	opt := AttachPaymentOptions{
		CustomerID:      claims.ID,
		PaymentMethodID: req.PaymentMethodID,
	}

	if err := s.SubscriptionManager.AttachPayment(ctx, opt); err != nil {
		logger.Error("Unable to attach payment to customer",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to attach payment").WithResult(err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

type SubscriptionSetupRequest struct {
	PlanID string `json:"planId"`
}

func (s *Service) setupSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := ctx.Value(auth.Context).(*auth.Claims)

	logger := s.Logger.With(zap.String("CustomerID", claims.ID))

	var req SubscriptionSetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.WriteError(w, r, resp.ErrInvalidJson())
		return
	}

	plan, ok := s.SubscriptionManager.GetDefinedPlanByID(req.PlanID)
	if !ok {
		resp.WriteError(w, r, resp.ErrBadRequest().AddMessages("Specified Plan is invalid or has retired"))
		return
	}
	logger = logger.With(zap.String("PlanID", req.PlanID))

	opt := CreateFromOptionOption{
		CustomerID: claims.ID,
		Plan:       plan,
	}

	sub, err := s.SubscriptionManager.CreateSubscriptionFromPlan(ctx, opt)

	if err != nil {
		logger.Error("Unable to setup subscription in Stripe",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to setup subscription", "Payment gateway returns error"))
		return
	}

	var subscription Subscription
	if err := subscription.FromStripeResponse(sub, plan); err != nil {
		logger.Error("Unable to construct Subscription from Stripe response",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to setup subscription", err.Error()))
		return
	}

	if err := s.SubscriptionManager.Create(ctx, &subscription); err != nil {
		logger.Error("Unable to save subscription record to database",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to setup subscription", "Database returns error"))
		return
	}

	resp.WriteResponse(w, r, struct {
		StripeResponse *stripe.Subscription `json:"stripeResponse"`
		Subscription   *Subscription        `json:"subscription"`
	}{
		StripeResponse: sub,
		Subscription:   &subscription,
	})
}

func (s *Service) listPlans(w http.ResponseWriter, r *http.Request) {
	resp.WriteResponse(w, r, s.SubscriptionManager.ListDefinedPlans())
}

func (s *Service) listSubscriptions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := ctx.Value(auth.Context).(*auth.Claims)
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

	logger := s.Logger.With(zap.String("CustomerID", claims.ID))

	opt := ListOption{
		CustomerID: claims.ID,
		Before:     parsedTime,
		Limit:      10,
	}

	results, err := s.SubscriptionManager.List(ctx, opt)
	if err != nil {
		logger.Error("Unable to list subscriptions by customer id",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Cannot get the list of subscriptions"))
		return
	}

	resp.WriteResponse(w, r, results)
}

func (s *Service) Router() http.Handler {
	r := chi.NewRouter()

	r.Get("/plans", s.listPlans)

	r.Get("/", s.listSubscriptions)
	r.Post("/initialSetup", s.setupPayment)
	r.Post("/", s.setupSubscription)

	return r
}
