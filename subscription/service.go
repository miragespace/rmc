package subscription

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/miragespace/rmc/auth"
	resp "github.com/miragespace/rmc/response"

	"github.com/go-chi/chi"
	"github.com/stripe/stripe-go/v72"
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

func (s *Service) createStripeSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := ctx.Value(auth.Context).(*auth.Claims)

	logger := s.Logger.With(zap.String("CustomerID", claims.ID))

	var req SubscriptionSetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.WriteError(w, r, resp.ErrInvalidJson())
		return
	}

	plan, err := s.SubscriptionManager.GetPlan(ctx, req.PlanID)
	if err != nil {
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to setup subscription - Database returns error"))
		return
	}
	if plan == nil {
		resp.WriteError(w, r, resp.ErrNotFound().AddMessages("Specified Plan does not exist"))
		return
	}
	if plan.Retired {
		resp.WriteError(w, r, resp.ErrBadRequest().AddMessages("Specified Plan has retired"))
		return
	}
	logger = logger.With(zap.String("PlanID", req.PlanID))

	opt := CreateFromPlanOption{
		CustomerID: claims.ID,
		Plan:       *plan,
	}

	sub, err := s.SubscriptionManager.CreateSubscriptionFromPlan(ctx, opt)

	if err != nil {
		if stripeErr, ok := err.(*stripe.Error); ok {
			if stripeErr.Code == stripe.ErrorCodeResourceMissing && strings.Contains(stripeErr.Msg, "no attached payment source") {
				resp.WriteError(w, r, resp.ErrForbidden().WithMessage("No payment method on file"))
				return
			}
		}
		logger.Error("Unable to setup subscription in Stripe",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to setup subscription - Payment gateway returns error"))
		return
	}

	resp.WriteResponse(w, r, sub)
}

func (s *Service) createSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := ctx.Value(auth.Context).(*auth.Claims)
	id := chi.URLParam(r, "id")

	logger := s.Logger.With(zap.String("CustomerID", claims.ID))

	sub, err := s.SubscriptionManager.GetStripe(ctx, GetOption{
		SubscriptionID: id,
	})

	if err != nil {
		logger.Error("Unable to fetch subscription from Stripe",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to setup subscription - Payment gateway returned error"))
		return
	}

	if sub.Customer.ID != claims.ID {
		resp.WriteError(w, r, resp.ErrNotFound().AddMessages("Unable to setup subscription - Subscription not found"))
		return
	}

	if sub.Status != stripe.SubscriptionStatusActive {
		resp.WriteError(w, r, resp.ErrBadRequest().AddMessages("Unable to setup scription - Subscription is not active"))
		return
	}

	logger = logger.With(zap.String("SubscriptionID", sub.ID))

	// we have to lookup the actual planID from the subscription
	var planID string
	for _, item := range sub.Items.Data {
		if item.Price != nil {
			planID = item.Price.Product.ID
		}
	}
	if planID == "" {
		logger.Error("No Plan found for subscription")
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to lookup subscription's associated Plan"))
		return
	}
	logger = logger.With(zap.String("PlanID", planID))

	plan, err := s.SubscriptionManager.GetPlan(ctx, planID)
	if err != nil {
		logger.Error("Unable to lookup Plan from database",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to setup subscription - Database returns error"))
		return
	}
	if plan == nil {
		resp.WriteError(w, r, resp.ErrNotFound().AddMessages("Specified Plan does not exist"))
		return
	}
	if plan.Retired {
		resp.WriteError(w, r, resp.ErrBadRequest().AddMessages("Specified Plan has retired"))
		return
	}
	logger = logger.With(zap.String("PlanID", planID))

	var subscription Subscription
	if err := subscription.fromStripeResponse(sub, plan); err != nil {
		logger.Error("Unable to construct Subscription from Stripe response",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to setup subscription - "+err.Error()))
		return
	}

	if err := s.SubscriptionManager.Create(ctx, &subscription); err != nil {
		logger.Error("Unable to save subscription record to database",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to setup subscription - Database returns error"))
		return
	}

	resp.WriteResponse(w, r, subscription)

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

func (s *Service) getSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := ctx.Value(auth.Context).(*auth.Claims)
	id := chi.URLParam(r, "id")

	sub, err := s.SubscriptionManager.Get(ctx, GetOption{
		CustomerID:     claims.ID,
		SubscriptionID: id,
	})
	if err != nil {
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to fetch subscription"))
		return
	}

	if sub == nil {
		resp.WriteError(w, r, resp.ErrBadRequest().AddMessages("Cannot find subscription with specific ID"))
		return
	}

	resp.WriteResponse(w, r, sub)
}

func (s *Service) getSubscriptionUsage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := ctx.Value(auth.Context).(*auth.Claims)
	id := chi.URLParam(r, "id")

	usages, err := s.SubscriptionManager.GetUsage(ctx, GetOption{
		CustomerID:     claims.ID,
		SubscriptionID: id,
	})
	if err != nil {
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to fetch usages"))
		return
	}

	if usages == nil {
		resp.WriteError(w, r, resp.ErrBadRequest().AddMessages("Cannot find usages with specific subscription ID"))
		return
	}

	resp.WriteResponse(w, r, usages)
}

func (s *Service) createPlans(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	plans := make([]Plan, 0, 1)

	if err := json.NewDecoder(r.Body).Decode(&plans); err != nil {
		resp.WriteError(w, r, resp.ErrInvalidJson())
		return
	}

	err := s.SubscriptionManager.createPlans(ctx, plans)
	if err != nil {
		s.Logger.Error("Unable to create plans",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().WithResult(err).AddMessages("Unable to create plans"))
		return
	}

	resp.WriteResponse(w, r, plans)
}

func (s *Service) listPlans(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	plans, err := s.SubscriptionManager.listPlans(ctx)
	if err != nil {
		resp.WriteError(w, r, resp.ErrUnexpected().AddMessages("Unable to list plans"))
		return
	}

	resp.WriteResponse(w, r, plans)
}

func (s *Service) AdminRouter() http.Handler {
	r := chi.NewRouter()

	r.Post("/plans", s.createPlans)
	r.Get("/plans", s.listPlans)

	return r
}

func (s *Service) Router() http.Handler {
	r := chi.NewRouter()

	r.Get("/plans", s.listPlans)

	r.Get("/", s.listSubscriptions)
	r.Get("/{id}", s.getSubscription)
	r.Get("/{id}/usages", s.getSubscriptionUsage)
	r.Post("/initialSetup", s.setupPayment)
	r.Post("/", s.createStripeSubscription)
	r.Put("/{id}", s.createSubscription)
	return r
}
