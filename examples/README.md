# Examples

## 2FA End-to-End Flow (`e2e.sh`)

This script drives a full 2FA OTP flow against the dev server:
- Sends an OTP for a given `user_id`
- Verifies the OTP with `remember_device=true` and a `device_id`
- Checks 2FA status without and with device to confirm trust

### Prerequisites
- `curl` installed
- Optional: `jq` for nicer JSON parsing (script falls back to `sed`)
- Dev server running: `go run ./cmd/dev` (Phase 7 scripts use `http://localhost:3001`)

### Getting a `user_id`
- From SQLite (dev DB):
  - `sqlite3 authsome_dev.db "select id,email from users;"`
- Or create a user via `/signup` and copy `User.ID`:
  - `curl -sS -X POST http://localhost:3000/api/auth/signup \
      -H 'Content-Type: application/json' \
      -d '{"email":"alice@example.com","password":"password123","name":"alice"}'`

### Usage
```
# Direct script
bash examples/e2e.sh <user_id> [device_id]

# Via Make (recommended)
make e2e-2fa USER_ID=<user_id> DEVICE_ID='curl/8.7.1|::1'
 
# Start dev server via Make
make dev

# List users from dev DB via Make
make db-users DB_PATH=authsome_dev.db
```
Examples:
```
bash examples/e2e.sh d3lid38h1fg5gg8up810 'curl/8.7.1|::1'
make e2e-2fa USER_ID=d3lid38h1fg5gg8up810 DEVICE_ID='curl/8.7.1|::1'
make db-users
```

### What you’ll see
- OTP send response (in dev, code is returned inline)
- Verification response (`{"status":"verified"}`)
- Status without device (`trusted:false`)
- Status with device (`trusted:true`)

### Dev vs Prod
- In development, `POST /2fa/send-otp` returns the OTP inline for convenience.
- In production, the OTP is delivered via configured channels (email/SMS), not in the response.

### Troubleshooting
- `{"error":"invalid user_id"}` indicates a malformed or non-xid `user_id`.
- If you see `UNIQUE constraint failed: users.username` during unrelated flows (e.g., anonymous plugin), it is separate from the 2FA flow.

## Phase 7 End-to-End Flow (`e2e_phase7.sh`)

This script validates the Phase 7 plugins against the dev server:
- Email OTP: send and verify (implicit signup in dev)
- Magic Link: send and verify
- Phone: send code and verify (associates phone with email)
- Passkey: register, list, delete, and login

### Prerequisites
- `curl` and `jq` installed
- Dev server running: `go run ./cmd/dev` (listens on `http://localhost:3001`)

### Usage
```
# Direct script
bash examples/e2e_phase7.sh

# Via Make (recommended)
make e2e-phase7 EMAIL=phase7.user@example.com PHONE=+15550001111 REMEMBER=true

# Start dev server via Make
make dev
```

Environment overrides (optional):
- `EMAIL` (default: `phase7.user@example.com`)
- `PHONE` (default: `+15550001111`)
- `REMEMBER` (default: `true`)

### What you’ll see
- Health check on `http://localhost:3001/health`
- Email OTP: dev OTP returned and verified, `user_id` extracted
- Magic Link: dev URL returned, token extracted, verified
- Phone: dev code returned, verified and bound to `email`
- Passkey: register, list (get ID), delete, and login flows succeed
- Summary with `Email`, `User ID`, and any generated tokens

### Troubleshooting
- Ensure the dev server is on port `3001` before running this script.
- If Passkey insert errors appear, restart the dev server to ensure latest plugin code is loaded.