#!/usr/bin/env bash

set -euo pipefail

BASE_URL="http://localhost:3000/api/auth"

usage() {
  echo "Usage: $0 <user_id> [device_id]" >&2
  echo "Example: $0 d3lid38h1fg5gg8up810 'curl/8.7.1|::1'" >&2
}

if [[ ${1:-} == "" ]]; then
  usage
  exit 1
fi

USER_ID="$1"
DEVICE_ID="${2:-curl/8.7.1|::1}"

if ! command -v curl >/dev/null 2>&1; then
  echo "Error: curl is required" >&2
  exit 1
fi

echo "[1/4] Sending OTP for user_id=${USER_ID}"
SEND_RESP=$(curl -sS -X POST "${BASE_URL}/2fa/send-otp" \
  -H 'Content-Type: application/json' \
  -d "{\"user_id\":\"${USER_ID}\"}")
echo "${SEND_RESP}" > /tmp/otp_send.json

# Extract OTP code (dev mode returns inline)
if command -v jq >/dev/null 2>&1; then
  CODE=$(jq -r '.code // empty' /tmp/otp_send.json)
else
  CODE=$(sed -nE 's/.*"code":"?([0-9]{4,8})"?.*/\1/p' /tmp/otp_send.json)
fi

if [[ -z "${CODE:-}" ]]; then
  echo "Error: Could not extract OTP code from response:" >&2
  cat /tmp/otp_send.json >&2
  echo "Note: In production, code is delivered via email/SMS and not returned inline." >&2
  exit 1
fi

echo "[2/4] Verifying OTP and remembering device_id=${DEVICE_ID}"
VERIFY_RESP=$(curl -sS -X POST "${BASE_URL}/2fa/verify" \
  -H 'Content-Type: application/json' \
  -d "{\"user_id\":\"${USER_ID}\",\"code\":\"${CODE}\",\"remember_device\":true,\"device_id\":\"${DEVICE_ID}\"}")
echo "${VERIFY_RESP}" > /tmp/otp_verify.json
echo "${VERIFY_RESP}"

echo "[3/4] Checking 2FA status (without device)"
STATUS_NO_DEV=$(curl -sS -X POST "${BASE_URL}/2fa/status" \
  -H 'Content-Type: application/json' \
  -d "{\"user_id\":\"${USER_ID}\"}")
echo "${STATUS_NO_DEV}" > /tmp/otp_status_no_device.json
echo "${STATUS_NO_DEV}"

echo "[4/4] Checking 2FA status (with device)"
STATUS_WITH_DEV=$(curl -sS -X POST "${BASE_URL}/2fa/status" \
  -H 'Content-Type: application/json' \
  -d "{\"user_id\":\"${USER_ID}\",\"device_id\":\"${DEVICE_ID}\"}")
echo "${STATUS_WITH_DEV}" > /tmp/otp_status_with_device.json
echo "${STATUS_WITH_DEV}"

echo "---"
echo "Summary:"
if command -v jq >/dev/null 2>&1; then
  VERIFIED=$(jq -r '.status // empty' /tmp/otp_verify.json)
  TRUSTED=$(jq -r '.trusted // empty' /tmp/otp_status_with_device.json)
else
  VERIFIED=$(sed -nE 's/.*"status":"?([a-z_]+)"?.*/\1/p' /tmp/otp_verify.json)
  TRUSTED=$(sed -nE 's/.*"trusted":(true|false).*/\1/p' /tmp/otp_status_with_device.json)
fi
echo "Verified: ${VERIFIED:-unknown}"
echo "Trusted (with device): ${TRUSTED:-unknown}"
set -euo pipefail

# Simple end-to-end script for Username + 2FA flows against the dev server
# Requirements: jq

if ! command -v jq >/dev/null 2>&1; then
  echo "This script requires 'jq'. Please install jq and rerun." >&2
  exit 1
fi

BASE="http://localhost:3000"
AUTH="$BASE/api/auth"

# You can override these via environment variables
USERNAME="${USERNAME:-alice}"
EMAIL="${EMAIL:-alice@example.com}"
PASSWORD="${PASSWORD:-P@ssw0rd-Example-123}"
REMEMBER="${REMEMBER:-true}"

post_json() {
  local body="$1"; shift
  local path="$1"; shift || true
  curl -s -S -H "Content-Type: application/json" -X POST "$AUTH$path" -d "$body"
}

echo "== Health check =="
if ! curl -s "$BASE/health" | jq .; then
  echo "Dev server not reachable at $BASE. Start it via: go run ./cmd/dev" >&2
  exit 1
fi

echo
echo "== Core signup ($EMAIL) =="
signup_resp=$(post_json "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"name\":\"$USERNAME\"}" "/signup")
echo "$signup_resp" | jq .

echo
echo "== First sign-in (expect no 2FA yet) =="
signin_resp=$(post_json "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"remember\":$REMEMBER}" "/signin")
echo "$signin_resp" | jq .
# Support both lowercase map keys (plugin challenge) and capitalized struct keys (core auth response)
user_id=$(jq -r '.user.id // .User.ID // empty' <<<"$signin_resp")
require_twofa=$(jq -r '.require_twofa // .RequireTwoFA // false' <<<"$signin_resp")
device_id=$(jq -r '.device_id // empty' <<<"$signin_resp")
token_initial=$(jq -r '.token // .Token // empty' <<<"$signin_resp")

if [[ -z "$user_id" ]]; then
  echo "Could not extract user.id from sign-in response." >&2
  exit 1
fi

if [[ "$require_twofa" != "true" ]]; then
  echo "No 2FA required yet. Proceeding to enable 2FA (otp)."
fi

echo
echo "== Enable 2FA (otp) for user $user_id =="
enable_resp=$(post_json "{\"user_id\":\"$user_id\",\"method\":\"otp\"}" "/2fa/enable")
echo "$enable_resp" | jq .
totp_uri=$(jq -r '.totp_uri // empty' <<<"$enable_resp")
if [[ -n "$totp_uri" ]]; then
  echo "Provision this TOTP in an authenticator (URI shown above)."
fi

echo
echo "== Send OTP code (dev) =="
otp_resp=$(post_json "{\"user_id\":\"$user_id\"}" "/2fa/send-otp")
echo "$otp_resp" | jq .
otp_code=$(jq -r '.code // empty' <<<"$otp_resp")
if [[ -z "$otp_code" ]]; then
  echo "Could not retrieve OTP code from send-otp response." >&2
  exit 1
fi

# If device_id from first sign-in is empty, fetch a fresh challenge to get it
if [[ -z "$device_id" ]]; then
  echo
  echo "== Fetch 2FA challenge to get device_id =="
  challenge_resp=$(post_json "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"remember\":$REMEMBER}" "/signin")
  echo "$challenge_resp" | jq .
  device_id=$(jq -r '.device_id // empty' <<<"$challenge_resp")
  if [[ -z "$device_id" ]]; then
    echo "device_id not present in challenge; cannot remember device." >&2
  fi
fi

echo
echo "== Verify 2FA with remember_device =="
verify_body="{\"user_id\":\"$user_id\",\"code\":\"$otp_code\",\"remember_device\":true,\"device_id\":\"$device_id\"}"
verify_resp=$(post_json "$verify_body" "/2fa/verify")
echo "$verify_resp" | jq .

echo
echo "== Second sign-in (should be trusted, no 2FA) =="
signin2_resp=$(post_json "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"remember\":$REMEMBER}" "/signin")
echo "$signin2_resp" | jq .
token_final=$(jq -r '.token // .Token // empty' <<<"$signin2_resp")
if [[ -n "$token_final" ]]; then
  echo "Obtained session token: $token_final"
fi

echo
echo "== Anonymous sign-in =="
anon_resp=$(post_json "{}" "/anonymous/signin")
echo "$anon_resp" | jq .

echo
echo "E2E flow complete."