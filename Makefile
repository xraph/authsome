.PHONY: e2e-2fa e2e-phase7

# Default device_id used in examples; override with DEVICE_ID='curl/8.7.1|::1'
DEVICE_ID ?= curl/8.7.1|::1

e2e-2fa:
	@ if [ -z "$(USER_ID)" ]; then \
		echo "ERROR: USER_ID is required" >&2; \
		echo "Usage: make e2e-2fa USER_ID=<xid> DEVICE_ID='curl/8.7.1|::1'" >&2; \
		exit 1; \
	fi
	@ echo "Running e2e 2FA flow for USER_ID=$(USER_ID) DEVICE_ID=$(DEVICE_ID)";
	@ bash examples/e2e.sh "$(USER_ID)" "$(DEVICE_ID)"

e2e-phase7:
	@ echo "Running Phase 7 e2e flows (Email OTP, Magic Link, Phone, Passkey)";
	@ bash examples/e2e_phase7.sh

.PHONY: dev db-users

DB_PATH ?= authsome_dev.db

dev:
	@ echo "Starting dev server (Ctrl+C to stop)...";
	@ go run ./cmd/dev

db-users:
	@ if ! command -v sqlite3 >/dev/null 2>&1; then \
		echo "ERROR: sqlite3 is required" >&2; \
		exit 1; \
	fi
	@ if [ ! -f "$(DB_PATH)" ]; then \
		echo "ERROR: DB file $(DB_PATH) not found" >&2; \
		exit 1; \
	fi
	@ echo "Listing users (id | email | username) from $(DB_PATH):";
	@ sqlite3 "$(DB_PATH)" "select id||' | '||email||' | '||coalesce(username,'') from users;"