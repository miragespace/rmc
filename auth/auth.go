package auth

import (
	"fmt"
	"net/smtp"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-redis/redis/v7"
	"github.com/johnsto/go-passwordless"
	"go.uber.org/zap"
)

// ContextKey is a defined type to be used in context.Context containing the Claims
type ContextKey string

// Context is key used in context.Context containing the Claims
const Context ContextKey = "authContext"

// Environment is the type for defining the running environment
type Environment string

// define constants
const (
	EnvDevelopment Environment = "Dev"
	EnvProduction  Environment = "Prod"
)

// Auth provides passwordless authentication
type Auth struct {
	Options
	pw     *passwordless.Passwordless
	jwtKey []byte
}

// Claims is the struct for jwt token
type Claims struct {
	jwt.StandardClaims
	Email string `json:"email"`
	ID    string `json:"id"`
}

// Options provides initialization parameters for Auth
type Options struct {
	Redis  redis.UniversalClient
	Logger *zap.Logger

	JWTSigningKey string

	Environment Environment
	SMTPAuth    smtp.Auth
	From        string
	Hostname    string
	EmailOption EmailOption
}

// EmailOption specifies the name shown in the email, and the LinkGenerator for the login link
type EmailOption struct {
	Name          string
	LinkGenerator LinkGenerator
}

// LinkGenerator is used to generator a login link
type LinkGenerator func(uid, token string) string

func (o *Options) validate() error {
	if o == nil {
		return fmt.Errorf("nil option is invalid")
	}
	if o.Redis == nil {
		return fmt.Errorf("nil redisClient is invalid")
	}
	if o.Logger == nil {
		return fmt.Errorf("nil Logger is invalid")
	}
	if len(o.JWTSigningKey) < 16 {
		return fmt.Errorf("jwt signing key must be longer than 16 characters")
	}
	if o.Environment == "" {
		o.Environment = EnvDevelopment
	}
	if o.SMTPAuth == nil {
		return fmt.Errorf("nil SMTPAuth is invalid")
	}
	if o.From == "" {
		return fmt.Errorf("Empty from is invalid")
	}
	if o.Hostname == "" {
		return fmt.Errorf("Empty hostname is invalid")
	}
	if o.EmailOption.Name == "" {
		return fmt.Errorf("Empty EmailOption.Name is invalid")
	}
	if o.EmailOption.LinkGenerator == nil {
		return fmt.Errorf("nil EmailOption.LinkGenerator is invalid")
	}

	return nil
}

// New will return a new instance of Auth for authentication
func New(option Options) (*Auth, error) {
	if err := option.validate(); err != nil {
		return nil, err
	}

	pw := passwordless.New(passwordless.NewRedisStore(option.Redis))
	pw.SetTransport("Log", passwordless.LogTransport{
		MessageFunc: func(token, uid string) string {
			return option.EmailOption.LinkGenerator(uid, token)
		},
	}, passwordless.NewCrockfordGenerator(8), time.Minute*30)
	pw.SetTransport("Email", passwordless.NewSMTPTransport(
		option.Hostname,
		option.From,
		option.SMTPAuth,
		composeFuncGetter(option.EmailOption),
	), passwordless.NewCrockfordGenerator(32), time.Minute*15)

	return &Auth{
		Options: option,
		pw:      pw,
		jwtKey:  []byte(option.JWTSigningKey),
	}, nil
}
