package user

import (
	"context"
	"database/sql"
	systemErrors "errors"
	"fmt"
	"log"
	"runtime/debug"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/executioncontext"
)

const (
	// RequestUserAccessToken command bus contract
	RequestUserAccessToken = "request-user-access-token"
	// ChangeUserEmailAddress command bus contract
	ChangeUserEmailAddress = "change-user-email-address"
	// RegisterUserWithEmail command bus contract
	RegisterUserWithEmail = "register-user-with-email"
	// RegisterUserWithFacebook command bus contract
	RegisterUserWithFacebook = "register-user-with-facebook"
	// RegisterUserWithGoogle command bus contract
	RegisterUserWithGoogle = "register-user-with-google"
)

// NewCommandFromPayload builds command by contract from json payload
func NewCommandFromPayload(contract string, payload []byte) (domain.Command, error) {
	switch contract {
	case RegisterUserWithEmail:
		registerWithEmail := RegisterWithEmail{}
		err := unmarshalPayload(payload, &registerWithEmail)

		return registerWithEmail, err
	case RegisterUserWithGoogle:
		registerWithGoogle := RegisterWithGoogle{}
		err := unmarshalPayload(payload, &registerWithGoogle)

		return registerWithGoogle, err
	case RegisterUserWithFacebook:
		registerWithFacebook := RegisterWithFacebook{}
		err := unmarshalPayload(payload, &registerWithFacebook)

		return registerWithFacebook, err
	case ChangeUserEmailAddress:
		changeEmailAddress := ChangeEmailAddress{}
		err := unmarshalPayload(payload, &changeEmailAddress)

		return changeEmailAddress, err
	case RequestUserAccessToken:
		requestAccessToken := RequestAccessToken{}
		err := unmarshalPayload(payload, &requestAccessToken)

		return requestAccessToken, err
	default:
		return nil, errors.New("Invalid command contract")
	}
}

// RequestAccessToken command
type RequestAccessToken struct {
	Email EmailAddress `json:"email"`
}

// GetName returns command name
func (c RequestAccessToken) GetName() string {
	return fmt.Sprintf("%T", c)
}

// OnRequestAccessToken creates command handler
func OnRequestAccessToken(repository Repository, db *sql.DB) commandbus.CommandHandler {
	fn := func(ctx context.Context, c RequestAccessToken, out chan<- error) {
		// this goroutine runs independently to request's goroutine,
		// therefore recover middleware will not recover from panic to prevent crash
		defer recoverCommandHandler(out)

		var id string
		row := db.QueryRowContext(ctx, `SELECT id FROM users WHERE emailAddress=? LIMIT 1`, c.Email.String())
		if err := row.Scan(&id); err != nil {
			if systemErrors.Is(err, sql.ErrNoRows) {
				out <- errors.Wrap(fmt.Errorf("%s: %w", err, application.ErrNotFound))
				return
			}
			out <- errors.Wrap(err)
			return
		}
		if id == "" {
			out <- application.ErrNotFound
			return
		}

		userID, err := uuid.Parse(id)
		if err != nil {
			out <- errors.Wrap(err)
			return
		}

		u, err := repository.Get(ctx, userID)
		if err != nil {
			out <- errors.Wrap(err)
			return
		}

		if err := u.RequestAccessToken(); err != nil {
			out <- errors.Wrap(err)
			return
		}

		out <- repository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), u)
	}

	return commandbus.CommandHandler(fn)
}

// ChangeEmailAddress command
type ChangeEmailAddress struct {
	ID    uuid.UUID    `json:"id"`
	Email EmailAddress `json:"email"`
}

// GetName returns command name
func (c ChangeEmailAddress) GetName() string {
	return fmt.Sprintf("%T", c)
}

// OnChangeEmailAddress creates command handler
func OnChangeEmailAddress(repository Repository, db *sql.DB) commandbus.CommandHandler {
	fn := func(ctx context.Context, c ChangeEmailAddress, out chan<- error) {
		// this goroutine runs independently to request's goroutine,
		// therefore recover middleware will not recover from panic to prevent crash
		defer recoverCommandHandler(out)

		var totalUsers int32

		row := db.QueryRowContext(ctx, `SELECT COUNT(distinctId) FROM users WHERE emailAddress=?`, c.Email.String())
		if err := row.Scan(&totalUsers); err != nil {
			out <- errors.Wrap(err)
			return
		}

		if totalUsers != 0 {
			out <- errors.Wrap(application.ErrInvalid)
			return
		}

		u, err := repository.Get(ctx, c.ID)
		if err != nil {
			out <- errors.Wrap(err)
			return
		}

		if err := u.ChangeEmailAddress(c.Email); err != nil {
			out <- errors.Wrap(err)
			return
		}

		out <- repository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), u)
	}

	return commandbus.CommandHandler(fn)
}

// RegisterWithEmail command
type RegisterWithEmail struct {
	Email EmailAddress `json:"email"`
}

// GetName returns command name
func (c RegisterWithEmail) GetName() string {
	return fmt.Sprintf("%T", c)
}

// OnRegisterWithEmail creates command handler
func OnRegisterWithEmail(repository Repository, db *sql.DB) commandbus.CommandHandler {
	fn := func(ctx context.Context, c RegisterWithEmail, out chan<- error) {
		// this goroutine runs independently to request's goroutine,
		// therefore recover middleware will not recover from panic to prevent crash
		defer recoverCommandHandler(out)

		var totalUsers int32

		row := db.QueryRowContext(ctx, `SELECT COUNT(distinctId) FROM users WHERE emailAddress=?`, c.Email.String())
		if err := row.Scan(&totalUsers); err != nil {
			out <- errors.Wrap(err)
			return
		}

		if totalUsers != 0 {
			out <- errors.Wrap(application.ErrInvalid)
			return
		}

		id, err := uuid.NewRandom()
		if err != nil {
			out <- errors.Wrap(err)
			return
		}

		u := New()
		if err := u.RegisterWithEmail(id, c.Email); err != nil {
			out <- errors.Wrap(err)
			return
		}

		out <- repository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), u)
	}

	return commandbus.CommandHandler(fn)
}

// RegisterWithFacebook command
type RegisterWithFacebook struct {
	Email      EmailAddress `json:"email"`
	FacebookID string       `json:"facebook_id"`
}

// GetName returns command name
func (c RegisterWithFacebook) GetName() string {
	return fmt.Sprintf("%T", c)
}

// OnRegisterWithFacebook creates command handler
func OnRegisterWithFacebook(repository Repository, db *sql.DB) commandbus.CommandHandler {
	fn := func(ctx context.Context, c RegisterWithFacebook, out chan<- error) {
		// this goroutine runs independently to request's goroutine,
		// therefore recover middleware will not recover from panic to prevent crash
		defer recoverCommandHandler(out)

		var id, emailAddress, facebookID string

		row := db.QueryRowContext(ctx, `SELECT id, emailAddress, facebookId FROM users WHERE emailAddress=? OR facebookId=? LIMIT 1`, c.Email.String(), c.FacebookID)
		if err := row.Scan(&id, &emailAddress, &facebookID); err != nil && !systemErrors.Is(err, sql.ErrNoRows) {
			out <- errors.Wrap(err)
			return
		}

		if facebookID == c.FacebookID {
			out <- errors.Wrap(application.ErrInvalid)
			return
		}

		var u User
		if emailAddress == string(c.Email) {
			userID, err := uuid.Parse(id)
			if err != nil {
				out <- errors.Wrap(err)
				return
			}

			u, err = repository.Get(ctx, userID)
			if err != nil {
				out <- errors.Wrap(err)
				return
			}

			if err := u.ConnectWithFacebook(c.FacebookID); err != nil {
				out <- errors.Wrap(err)
				return
			}
		} else {
			id, err := uuid.NewRandom()
			if err != nil {
				out <- errors.Wrap(err)
				return
			}

			u = New()
			if err := u.RegisterWithFacebook(id, c.Email, c.FacebookID); err != nil {
				out <- errors.Wrap(err)
				return
			}
		}

		out <- repository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), u)
	}

	return commandbus.CommandHandler(fn)
}

// RegisterWithGoogle command
type RegisterWithGoogle struct {
	Email    EmailAddress `json:"email"`
	GoogleID string       `json:"google_id"`
}

// GetName returns command name
func (c RegisterWithGoogle) GetName() string {
	return fmt.Sprintf("%T", c)
}

// OnRegisterWithGoogle creates command handler
func OnRegisterWithGoogle(repository Repository, db *sql.DB) commandbus.CommandHandler {
	fn := func(ctx context.Context, c RegisterWithGoogle, out chan<- error) {
		// this goroutine runs independently to request's goroutine,
		// therefore recover middleware will not recover from panic to prevent crash
		defer recoverCommandHandler(out)

		var id, emailAddress, googleID string

		row := db.QueryRowContext(ctx, `SELECT id, emailAddress, googleId FROM users WHERE emailAddress=? OR googleId=? LIMIT 1`, c.Email.String(), c.GoogleID)
		if err := row.Scan(&id, &emailAddress, &googleID); err != nil && !systemErrors.Is(err, sql.ErrNoRows) {
			out <- errors.Wrap(err)
			return
		}

		if googleID == c.GoogleID {
			out <- errors.Wrap(application.ErrInvalid)
			return
		}

		var u User
		if emailAddress == string(c.Email) {
			userID, err := uuid.Parse(id)
			if err != nil {
				out <- errors.Wrap(err)
				return
			}

			u, err = repository.Get(ctx, userID)
			if err != nil {
				out <- errors.Wrap(err)
				return
			}

			if err := u.ConnectWithGoogle(c.GoogleID); err != nil {
				out <- errors.Wrap(err)
				return
			}
		} else {
			id, err := uuid.NewRandom()
			if err != nil {
				out <- errors.Wrap(err)
				return
			}

			u = New()
			if err := u.RegisterWithGoogle(id, c.Email, c.GoogleID); err != nil {
				out <- errors.Wrap(err)
				return
			}
		}

		out <- repository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), u)
	}

	return commandbus.CommandHandler(fn)
}

func recoverCommandHandler(out chan<- error) {
	if r := recover(); r != nil {
		out <- errors.New(fmt.Sprintf("[CommandHandler] Recovered in %v", r))

		// Log the Go stack trace for this panic'd goroutine.
		log.Printf("%s\n", debug.Stack())
	}
}
