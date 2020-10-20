package auth

import (
	"context"
	"fmt"
	"io"
	"net/smtp"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/johnsto/go-passwordless"
)

// Auth provides passwordless authentication
type Auth struct {
	pw *passwordless.Passwordless
}

// Options provides initialization parameters for Auth
type Options struct {
	Redis redis.UniversalClient

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
	pw.SetTransport("Email", passwordless.NewSMTPTransport(
		option.Hostname,
		option.From,
		option.SMTPAuth,
		composeFuncGetter(option.EmailOption),
	), passwordless.NewCrockfordGenerator(32), time.Minute*15)

	return &Auth{
		pw: pw,
	}, nil
}

// Request will send a link to email with the login token
func (a *Auth) Request(ctx context.Context, uid, recipient string) error {
	return a.pw.RequestToken(ctx, "Email", uid, recipient)
}

// Verify checks if the login token is valid and corresonds to the user
func (a *Auth) Verify(ctx context.Context, uid, token string) (bool, error) {
	return a.pw.VerifyToken(ctx, uid, token)
}

func composeFuncGetter(options EmailOption) passwordless.ComposerFunc {
	return func(ctx context.Context, token, uid, recipient string, w io.Writer) error {
		e := &passwordless.Email{
			Subject: "Login Token for " + options.Name,
			To:      recipient,
		}

		link := options.LinkGenerator(uid, token)

		text := "You (or someone who knows your email address) wants " +
			"to sign in to " + options.Name + ".\n\n" +
			"Your PIN is " + token + " - or use the following link: " +
			link + "\n\n" +
			"(If you were did not request or were not expecting this email, " +
			"you can safely ignore it.)"
		html := "<!doctype html><html><body>" +
			"<p>You (or someone who knows your email address) wants " +
			"to sign in to " + options.Name + ".</p>" +
			"<p>Your PIN is <b>" + token + "</b> - or <a href=\"" + link + "\">" +
			"click here</a> to sign in automatically.</p>" +
			"<p>(If you did not request or were not expecting this email, " +
			"you can safely ignore it.)</p></body></html>"

		e.AddBody("text/plain", text)
		e.AddBody("text/html", html)

		_, err := e.Write(w)

		return err
	}
}
