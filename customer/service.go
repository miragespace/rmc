package customer

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zllovesuki/rmc/auth"

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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	logger := s.Logger.With(zap.String("email", req.Email))

	if err := validate.Struct(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	b64Email := base64.StdEncoding.EncodeToString([]byte(req.Email))
	if err := s.Auth.Request(r.Context(), b64Email, req.Email); err != nil {
		logger.Error("Unable to send login PIN",
			zap.Error(err),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) handleLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	b64Email := chi.URLParam(r, "uid")
	token := chi.URLParam(r, "token")

	emailBytes, err := base64.StdEncoding.DecodeString(b64Email)
	if err != nil {
		http.Error(w, "invalid uid", http.StatusBadRequest)
		return
	}

	email := string(emailBytes)

	logger := s.Logger.With(zap.String("email", email))

	valid, err := s.Auth.Verify(r.Context(), b64Email, token)
	if err != nil {
		logger.Error("Unable to verify login PIN",
			zap.Error(err),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !valid {
		http.Error(w, "not authorized", http.StatusUnauthorized)
		return
	}

	// "upsert" a customer
	cust, err := s.CustomerManager.GetByEmail(ctx, email)
	if err != nil {
		logger.Error("Unable to create Customer",
			zap.Error(err),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if cust == nil {
		// new customer! yay
		cust, err = s.CustomerManager.New(ctx, email)
		if err != nil {
			logger.Error("Unable to create Customer",
				zap.Error(err),
			)
			http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Token string `json:"token"`
	}{
		Token: jwtToken,
	})
}

// Router will return the routes under customer API
func (s *Service) Router() http.Handler {
	r := chi.NewRouter()

	r.Post("/", s.requestLogin)
	r.Get("/{uid}/{token}", s.handleLogin)

	return r
}
