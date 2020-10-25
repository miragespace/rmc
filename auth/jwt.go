package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	resp "github.com/zllovesuki/rmc/response"
	"go.uber.org/zap"
)

var bearerPrefix = "Bearer "

// CreateTokenFromClaims will create a signed jwt token that contains the given Claims
func (a *Auth) CreateTokenFromClaims(claims Claims) (string, error) {
	var expirationTime time.Time
	if a.Environment == EnvDevelopment {
		expirationTime = time.Now().Add(time.Hour * 24)
	} else {
		expirationTime = time.Now().Add(time.Minute * 30)
	}
	claims.StandardClaims = jwt.StandardClaims{
		ExpiresAt: expirationTime.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtKey)
}

func (a *Auth) verifyToken(token string) (*Claims, error) {
	claims := &Claims{}
	jwtToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return a.jwtKey, nil
	})
	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			return nil, nil
		}
		if _, ok := err.(*jwt.ValidationError); ok {
			return nil, nil
		}
		return nil, err
	}
	if !jwtToken.Valid {
		return nil, nil
	}
	return claims, nil
}

// Middleware returns a http middleware to verify Bearer in the header
// TODO: Implement refresh mechanism
func (a *Auth) Middleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			n := len(bearerPrefix)
			if len(auth) < n || auth[:n] != bearerPrefix {
				resp.WriteError(w, r, resp.ErrNoBearer())
				return
			}
			claims, err := a.verifyToken(auth[n:])
			if err != nil {
				a.Logger.Error("Cannot verify JWT token",
					zap.Error(err),
				)
				resp.WriteError(w, r, resp.ErrUnexpected())
				return
			}
			if claims == nil {
				resp.WriteError(w, r, resp.ErrNoBearer())
				return
			}

			ctx := context.WithValue(r.Context(), Context, claims)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimCheck returns a http middlware to authenticated route to ensure that Claims exists in the context
func (a *Auth) ClaimCheck() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, ok := r.Context().Value(Context).(*Claims)
			if !ok {
				a.Logger.Error("Context has no Claims")
				resp.WriteError(w, r, resp.ErrUnexpected())
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
