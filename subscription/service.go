package subscription

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zllovesuki/rmc/auth"

	"github.com/go-chi/chi"
	"github.com/stripe/stripe-go/v71"
	"github.com/stripe/stripe-go/v71/customer"
	"github.com/stripe/stripe-go/v71/paymentmethod"
	"github.com/stripe/stripe-go/v71/sub"
	"go.uber.org/zap"
)

type ServiceOptions struct {
	Logger *zap.Logger
}

type Service struct {
	ServiceOptions
}

func NewService(option ServiceOptions) (*Service, error) {
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	logger = logger.With(zap.String("PaymentMethodID", req.PaymentMethodID))

	params := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(claims.ID),
	}
	pm, err := paymentmethod.Attach(
		req.PaymentMethodID,
		params,
	)
	if err != nil {
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			Error error `json:"error"`
		}{
			err,
		})
		return
	}

	customerParams := &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(pm.ID),
		},
	}
	if _, err := customer.Update(
		claims.ID,
		customerParams,
	); err != nil {
		logger.Error("Unable to update payment method in Stripe",
			zap.Error(err),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type SubscriptionSetupRequest struct {
	PriceID string `json:"priceId"` // TODO: replace this with PlanID
}

func (s *Service) setupSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims := ctx.Value(auth.Context).(*auth.Claims)

	logger := s.Logger.With(zap.String("CustomerID", claims.ID))

	var req SubscriptionSetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	subscriptionParams := &stripe.SubscriptionParams{
		Customer: stripe.String(claims.ID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Plan: stripe.String(req.PriceID),
			},
		},
	}
	subscriptionParams.AddExpand("latest_invoice.payment_intent")
	subscriptionParams.AddExpand("pending_setup_intent")

	subscription, err := sub.New(subscriptionParams)

	if err != nil {
		logger.Error("Unable to setup subscription in Stripe",
			zap.Error(err),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscription)
}

func (s *Service) Router() http.Handler {
	r := chi.NewRouter().With(auth.ClaimCheck(s.Logger))

	r.Post("/initialSetup", s.setupPayment)
	r.Post("/", s.setupSubscription)

	return r
}
