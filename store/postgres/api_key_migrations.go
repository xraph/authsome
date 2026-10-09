package postgres

import (
	"context"

	"github.com/xraph/grove/migrate"
)

func init() {
	Migrations.MustRegister(&migrate.Migration{
		Name: "api_key_machine_owners", Version: "20261009000001",
		Up: func(ctx context.Context, exec migrate.Executor) error {
			// One statement keeps validation and schema changes atomic, including when
			// the executor does not wrap individual migrations in a transaction.
			_, err := exec.Exec(ctx, `DO $$ BEGIN
    ALTER TABLE authsome_api_keys ADD COLUMN service_account_id TEXT;
    ALTER TABLE authsome_api_keys ALTER COLUMN user_id DROP NOT NULL;
    ALTER TABLE authsome_api_keys ADD CONSTRAINT authsome_api_keys_owner_check
        CHECK ((user_id IS NOT NULL AND user_id <> '' AND service_account_id IS NULL)
            OR (user_id IS NULL AND service_account_id IS NOT NULL AND service_account_id <> ''));
    ALTER TABLE authsome_users ADD CONSTRAINT authsome_api_key_users_app_unique UNIQUE (id, app_id);
    ALTER TABLE authsome_service_accounts ADD CONSTRAINT authsome_api_key_accounts_scope_unique UNIQUE (id, app_id, env_id);
    ALTER TABLE authsome_environments ADD CONSTRAINT authsome_api_key_env_app_unique UNIQUE (id, app_id);
    ALTER TABLE authsome_api_keys ADD CONSTRAINT authsome_api_keys_user_app_fk
        FOREIGN KEY (user_id, app_id) REFERENCES authsome_users(id, app_id);
    ALTER TABLE authsome_api_keys ADD CONSTRAINT authsome_api_keys_account_scope_fk
        FOREIGN KEY (service_account_id, app_id, env_id)
        REFERENCES authsome_service_accounts(id, app_id, env_id) ON DELETE CASCADE;
    ALTER TABLE authsome_api_keys ADD CONSTRAINT authsome_api_keys_env_app_fk
        FOREIGN KEY (env_id, app_id) REFERENCES authsome_environments(id, app_id);
    CREATE INDEX idx_authsome_api_keys_service_account ON authsome_api_keys (service_account_id)
        WHERE service_account_id IS NOT NULL;
END $$`)
			return err
		},
		Down: func(ctx context.Context, exec migrate.Executor) error {
			// Keep the guard and reversal under the same lock so an issuer cannot
			// insert a machine key between the check and removal of its owner column.
			_, err := exec.Exec(ctx, `DO $$ BEGIN
LOCK TABLE authsome_api_keys IN ACCESS EXCLUSIVE MODE;
IF EXISTS (SELECT 1 FROM authsome_api_keys WHERE service_account_id IS NOT NULL) THEN
    RAISE EXCEPTION 'cannot downgrade api key machine owners while machine keys exist';
END IF;
ALTER TABLE authsome_api_keys ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE authsome_api_keys DROP CONSTRAINT authsome_api_keys_owner_check;
ALTER TABLE authsome_api_keys DROP CONSTRAINT authsome_api_keys_user_app_fk;
ALTER TABLE authsome_api_keys DROP CONSTRAINT authsome_api_keys_account_scope_fk;
ALTER TABLE authsome_api_keys DROP CONSTRAINT authsome_api_keys_env_app_fk;
DROP INDEX idx_authsome_api_keys_service_account;
ALTER TABLE authsome_api_keys DROP COLUMN service_account_id;
ALTER TABLE authsome_users DROP CONSTRAINT authsome_api_key_users_app_unique;
ALTER TABLE authsome_service_accounts DROP CONSTRAINT authsome_api_key_accounts_scope_unique;
ALTER TABLE authsome_environments DROP CONSTRAINT authsome_api_key_env_app_unique;
END $$`)
			return err
		},
	})
}
