package customer

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zllovesuki/rmc/auth"
	resp "github.com/zllovesuki/rmc/response"

	"github.com/go-chi/chi"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

var validate *validator.Validate = validator.New()

// ServiceOptions contains the configuration for Service router
type ServiceOptions struct {
	Auth            *auth.Auth
	CustomerManager *Manager
	Logger          *zap.Logger
}

// Service is the customer API router
type Service struct {
	ServiceOptions
}

// LoginRequest is the model of user request for login pin
type LoginRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// NewService will create an instance of the customer API router
func NewService(option ServiceOptions) (*Service, error) {
	if option.Auth == nil {
		return nil, fmt.Errorf("nil Auth is invalid")
	}
	if option.CustomerManager == nil {
		return nil, fmt.Errorf("nil CustomerManager is invalid")
	}
	if option.Logger == nil {
		return nil, fmt.Errorf("nil Logger is invalid")
	}
	return &Service{
		ServiceOptions: option,
	}, nil
}

func (s *Service) requestLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.WriteError(w, r, resp.ErrInvalidJson())
		return
	}
	logger := s.Logger.With(zap.String("email", req.Email))

	if err := validate.Struct(&req); err != nil {
		resp.WriteError(w, r, resp.ErrBadRequest().WithMessage("Invalid email"))
		return
	}

	hexEmail := hex.EncodeToString([]byte(req.Email))
	if err := s.Auth.Request(r.Context(), hexEmail, req.Email); err != nil {
		logger.Error("Unable to send login PIN",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().WithMessage("Unable to request login token"))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) handleLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hexEmail := chi.URLParam(r, "uid")
	token := chi.URLParam(r, "token")

	emailBytes, err := hex.DecodeString(hexEmail)
	if err != nil {
		resp.WriteError(w, r, resp.ErrBadRequest().WithMessage("Invalid UID was provided"))
		return
	}

	email := string(emailBytes)

	logger := s.Logger.With(zap.String("email", email))

	valid, err := s.Auth.Verify(r.Context(), hexEmail, token)
	if err != nil {
		logger.Error("Unable to verify login PIN",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrVerifyToken())
		return
	}

	if !valid {
		resp.WriteError(w, r, resp.ErrUnauthorized().WithMessage("Invalid token"))
		return
	}

	// "upsert" a customer
	cust, err := s.CustomerManager.GetByEmail(ctx, email)
	if err != nil {
		logger.Error("Unable to create Customer",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrVerifyToken())
		return
	}

	if cust == nil {
		// new customer! yay
		cust, err = s.CustomerManager.New(ctx, email)
		if err != nil {
			logger.Error("Unable to create Customer",
				zap.Error(err),
			)
			resp.WriteError(w, r, resp.ErrVerifyToken())
			return
		}
	}

	jwtToken, err := s.Auth.CreateTokenFromClaims(auth.Claims{
		ID:    cust.ID,
		Email: cust.Email,
	})
	if err != nil {
		logger.Error("Unable to generate token",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrVerifyToken())
		return
	}

	resp.WriteResponse(w, r, struct {
		Token string `json:"token"`
	}{
		Token: jwtToken,
	})
}

func (s *Service) getStripeCustomer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := ctx.Value(auth.Context).(*auth.Claims)

	logger := s.Logger.With(
		zap.String("CustomerID", claims.ID),
	)

	cust, err := s.CustomerManager.GetStripe(ctx, claims.ID)
	if err != nil {
		logger.Error("Cannot get Stripe customer",
			zap.Error(err),
		)
		resp.WriteError(w, r, resp.ErrUnexpected().WithMessage("Cannot fetch details from Stripe"))
		return
	}

	resp.WriteResponse(w, r, cust)
}

// Router will return the routes under customer API
func (s *Service) Router() http.Handler {
	r := chi.NewRouter()

	r.Post("/", s.requestLogin)
	r.Get("/{uid}/{token}", s.handleLogin)

	r.With(s.Auth.Middleware(), s.Auth.ClaimCheck()).Get("/stripe", s.getStripeCustomer)

	return r
}
