package customer

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/zllovesuki/rmc/auth"
)

type Options struct {
	Auth            *auth.Auth
	CustomerManager *Manager
}

type Service struct {
	Option Options
}

type Request struct {
	Email string `json:"email"`
}

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
	var req Request
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
	params := mux.Vars(r)
	email := params["uid"]
	token := params["token"]

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
	cust, err := s.Option.CustomerManager.GetByEmail(email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if cust == nil {
		// new customer! yay
		cust, err = s.Option.CustomerManager.NewCustomer(email)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// TODO: Generate TTL token

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cust)
}

func (s *Service) Mount(r *mux.Router) {
	r.HandleFunc("/", s.requestLogin).Methods("POST")
	r.HandleFunc("/{uid}/{token}", s.handleLogin).Methods("GET")
}
