package auth

import "github.com/johnsto/go-passwordless"

// Environment is the type for defining the running environment
type Environment string

// define constants
const (
	EnvDevelopment Environment = "Dev"
	EnvProduction  Environment = "Prod"
)

// Auth provides passwordless authentication
type Auth struct {
	pw      *passwordless.Passwordless
	options Options
}
