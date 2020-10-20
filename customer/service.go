package customer

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/zllovesuki/rmc/auth"
)

// Options contains the configuration for Service router
type Options struct {
	Auth            *auth.Auth
	CustomerManager *Manager
}

// Service is the customer API router
type Service struct {
	Option Options
}

// LoginRequest is the model of user request for login
type LoginRequest struct {
	Email string `json:"email"`
}

// NewService will create an instance of the customer API router
func NewService(option Options) (*Service, error) {
	if option.Auth == nil {
		return nil, fmt.Errorf("nil Auth is invalid")
	}
	if option.CustomerManager == nil {
		return nil, fmt.Errorf("nil CustomerManager is invalid")
	}
	return &Service{
		Option: option,
	}, nil
}

func (s *Service) requestLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: check if email is valid

	if err := s.Option.Auth.Request(r.Context(), req.Email, req.Email); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) handleLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	email := chi.URLParam(r, "uid")
	token := chi.URLParam(r, "token")

	valid, err := s.Option.Auth.Verify(r.Context(), email, token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !valid {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// "upsert" a customer
	cust, err := s.Option.CustomerManager.GetByEmail(ctx, email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if cust == nil {
		// new customer! yay
		cust, err = s.Option.CustomerManager.NewCustomer(ctx, email)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// TODO: Generate TTL token

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cust)
}

// Router will return the routes under customer API
func (s *Service) Router() http.Handler {
	r := chi.NewRouter()

	r.Post("/", s.requestLogin)
	r.Get("/{uid}/{token}", s.handleLogin)

	return r
}
