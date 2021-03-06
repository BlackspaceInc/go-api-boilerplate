package eventhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/token"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/log"
)

// WhenTokenWasRemoved handles event
func WhenTokenWasRemoved(db *sql.DB, repository persistence.TokenRepository) eventbus.EventHandler {
	fn := func(parentCtx context.Context, event domain.Event) {
		ctx, cancel := context.WithTimeout(parentCtx, time.Second*120)
		defer cancel()

		logger := log.New(config.Env.App.Environment)
		logger.Info(ctx, "[EventHandler] %s\n", event.Payload)

		e := token.WasRemoved{}
		if err := json.Unmarshal(event.Payload, &e); err != nil {
			logger.Error(ctx, "[EventHandler] Error: %v\n", errors.Wrap(err))
			return
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			logger.Error(ctx, "[EventHandler] Error: %v\n", errors.Wrap(err))
			return
		}
		defer tx.Rollback()

		if err := repository.Delete(ctx, e.ID.String()); err != nil {
			logger.Error(ctx, "[EventHandler] Error: %v\n", errors.Wrap(err))
			return
		}

		if err := tx.Commit(); err != nil {
			logger.Error(ctx, "[EventHandler] Error: %v\n", errors.Wrap(err))
			return
		}
	}

	return fn
}
