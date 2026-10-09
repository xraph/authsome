package mongo

import (
	"context"
	"fmt"

	"github.com/xraph/grove/drivers/mongodriver/mongomigrate"
	"github.com/xraph/grove/migrate"
)

func init() {
	Migrations.MustRegister(&migrate.Migration{
		Name:    "webhook_relay_endpoint",
		Version: "20260922000005",
		Up: func(ctx context.Context, exec migrate.Executor) error {
			mexec, ok := exec.(*mongomigrate.Executor)
			if !ok {
				return fmt.Errorf("expected mongomigrate executor, got %T", exec)
			}
			// The collection's validator is generated from the model, so the
			// new fields have to be admitted before a webhook is written.
			return mexec.RefreshValidator(ctx, (*webhookModel)(nil))
		},
		Down: func(_ context.Context, _ migrate.Executor) error {
			return nil // forward-only on the validator, as the other field additions are
		},
	})
}
