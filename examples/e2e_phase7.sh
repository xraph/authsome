#!/usr/bin/env bash

set -euo pipefail

BASE="http://localhost:3001"
AUTH="$BASE/api/auth"

# You can override these via environment variables
EMAIL="${EMAIL:-phase7.user@example.com}"
PHONE="${PHONE:-+15550001111}"
REMEMBER="${REMEMBER:-true}"
# Generate a unique credential ID for passkey registration to avoid duplicates
CRED_ID="${CRED_ID:-cred-${RANDOM}-$(date +%s)}"

if ! command -v jq >/dev/null 2>&1; then
  echo "This script requires 'jq'. Please install jq and rerun." >&2
  exit 1
fi

post_json() {
  local body="$1"; shift
  local path="$1"; shift || true
  curl -s -S -H "Content-Type: application/json" -X POST "$AUTH$path" -d "$body"
}

get() {
  local path="$1"; shift || true
  curl -s -S "$AUTH$path"
}

echo "== Health check =="
if ! curl -s "$BASE/health" | jq .; then
  echo "Dev server not reachable at $BASE. Start it via: go run ./cmd/dev" >&2
  exit 1
fi

echo
echo "== Email OTP: send and verify (implicit signup enabled) =="
send_otp_resp=$(post_json "{\"email\":\"$EMAIL\"}" "/email-otp/send")
echo "$send_otp_resp" | jq .
otp_val=$(jq -r '.dev_otp // empty' <<<"$send_otp_resp")
if [[ -z "$otp_val" ]]; then
  echo "Could not retrieve dev_otp from send response." >&2
  exit 1
fi

verify_otp_resp=$(post_json "{\"email\":\"$EMAIL\",\"otp\":\"$otp_val\",\"remember\":$REMEMBER}" "/email-otp/verify")
echo "$verify_otp_resp" | jq .
user_id=$(jq -r '.user.ID // .user.id // empty' <<<"$verify_otp_resp")
token_otp=$(jq -r '.token // empty' <<<"$verify_otp_resp")
if [[ -z "$user_id" ]]; then
  echo "Could not extract user.ID from Email OTP verify response." >&2
  exit 1
fi

echo
echo "== Magic Link: send and verify =="
ml_send_resp=$(post_json "{\"email\":\"$EMAIL\"}" "/magic-link/send")
echo "$ml_send_resp" | jq .
dev_url=$(jq -r '.dev_url // empty' <<<"$ml_send_resp")
if [[ -z "$dev_url" ]]; then
  echo "Could not retrieve dev_url from magic link send response." >&2
  exit 1
fi
token_param=$(sed -n 's/.*token=\([^&]*\).*/\1/p' <<<"$dev_url")
ml_verify_resp=$(curl -s -S "$AUTH/magic-link/verify?token=$token_param")
echo "$ml_verify_resp" | jq .

echo
echo "== Phone: send code and verify (associate with email) =="
phone_send_resp=$(post_json "{\"phone\":\"$PHONE\"}" "/phone/send-code")
echo "$phone_send_resp" | jq .
dev_code=$(jq -r '.dev_code // empty' <<<"$phone_send_resp")
if [[ -z "$dev_code" ]]; then
  echo "Could not retrieve dev_code from phone send response." >&2
  exit 1
fi
phone_verify_resp=$(post_json "{\"phone\":\"$PHONE\",\"code\":\"$dev_code\",\"email\":\"$EMAIL\",\"remember\":$REMEMBER}" "/phone/verify")
echo "$phone_verify_resp" | jq .

echo
echo "== Passkey: register, list, delete, login =="
pk_begin_resp=$(post_json "{\"user_id\":\"$user_id\"}" "/passkey/register/begin")
echo "$pk_begin_resp" | jq .
pk_finish_resp=$(post_json "{\"user_id\":\"$user_id\",\"credential_id\":\"$CRED_ID\"}" "/passkey/register/finish")
echo "$pk_finish_resp" | jq .

pk_list_resp=$(get "/passkey/list?user_id=$user_id")
echo "$pk_list_resp" | jq .
# Handle lowercase JSON field names from schema tags
pk_id=$(jq -r '.[0].id // .[0].ID // empty' <<<"$pk_list_resp")
if [[ -z "$pk_id" ]]; then
  echo "No passkey id found in list response; registration may have failed." >&2
  exit 1
fi

pk_delete_resp=$(curl -s -S -X POST "$AUTH/passkey/delete/$pk_id")
echo "$pk_delete_resp" | jq .

pk_login_begin=$(post_json "{\"user_id\":\"$user_id\"}" "/passkey/login/begin")
echo "$pk_login_begin" | jq .
pk_login_finish=$(post_json "{\"user_id\":\"$user_id\",\"remember\":$REMEMBER}" "/passkey/login/finish")
echo "$pk_login_finish" | jq .
token_pk=$(jq -r '.token // empty' <<<"$pk_login_finish")

echo
echo "== Summary =="
echo "Email: $EMAIL"
echo "User ID: $user_id"
echo "OTP token: ${token_otp:-<none>}"
echo "Passkey token: ${token_pk:-<none>}"
echo "Done."