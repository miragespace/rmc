package auth

import (
	"context"
	"io"

	"github.com/johnsto/go-passwordless"
)

func (a *Auth) getTransport() string {
	if a.Environment == EnvProduction {
		return "Email"
	}
	return "Log"
}

// Request will send a link to email with the login token
func (a *Auth) Request(ctx context.Context, uid, recipient string) error {
	return a.pw.RequestToken(ctx, a.getTransport(), uid, recipient)
}

// Verify checks if the login token is valid and corresonds to the user
func (a *Auth) Verify(ctx context.Context, uid, token string) (bool, error) {
	valid, err := a.pw.VerifyToken(ctx, uid, token)
	switch err {
	case passwordless.ErrNoResponseWriter, passwordless.ErrNoStore, passwordless.ErrNoTransport, passwordless.ErrNotValidForContext:
		return valid, err
	default:
		return valid, nil
	}
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
			"Your token (expires in 15 minutes) is " + token + " - or use the following link: " +
			link + "\n\n" +
			"(If you were did not request or were not expecting this email, " +
			"you can safely ignore it.)"
		html := "<!doctype html><html><body>" +
			"<p>You (or someone who knows your email address) wants " +
			"to sign in to " + options.Name + ".</p>" +
			"<p>Your token (expires in 15 minutes) is <b>" + token + "</b> - or <a href=\"" + link + "\">" +
			"click here</a> to sign in automatically.</p>" +
			"<p>(If you did not request or were not expecting this email, " +
			"you can safely ignore it.)</p></body></html>"

		e.AddBody("text/plain", text)
		e.AddBody("text/html", html)

		_, err := e.Write(w)

		return err
	}
}
