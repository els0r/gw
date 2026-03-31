#!/usr/bin/env bash
# ~/.gw/hooks/lib.sh — shared helpers for gw hooks
# Source this at the top of any hook script.

GW_STATE_DIR="${HOME}/.gw"

gw_config_get() {
  local key="$1"
  local cfg="${GW_STATE_DIR}/config"
  [[ -f "$cfg" ]] && grep -E "^${key}=" "$cfg" | cut -d= -f2-
}

# ── EARLY token management ───────────────────────────────────────────────────

EARLY_BASE="https://api.early.app/api/v4"

gw_early_configured() {
  local key secret
  key="$(gw_config_get EARLY_API_KEY)"
  secret="$(gw_config_get EARLY_API_SECRET)"
  [[ -n "$key" && -n "$secret" ]]
}

# Exchange API key+secret for a bearer token; cache for 23h
gw_early_token() {
  local cache="${GW_STATE_DIR}/early_token"
  if [[ -f "$cache" ]]; then
    local age=$(( $(date +%s) - $(stat -f %m "$cache") ))
    [[ $age -lt 82800 ]] && { cat "$cache"; return 0; }
  fi
  local api_key api_secret
  api_key="$(gw_config_get EARLY_API_KEY)"
  api_secret="$(gw_config_get EARLY_API_SECRET)"
  local token
  token=$(curl -s -X POST "${EARLY_BASE}/developer/sign-in" \
    -H "Content-Type: application/json" \
    -d "{\"apiKey\":\"${api_key}\",\"apiSecret\":\"${api_secret}\"}" \
    | jq -r '.token // empty')
  [[ -z "$token" ]] && { echo "  ⚠  EARLY auth failed" >&2; return 1; }
  mkdir -p "$GW_STATE_DIR"
  echo "$token" > "$cache"
  chmod 600 "$cache"
  echo "$token"
}
