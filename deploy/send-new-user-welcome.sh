#!/usr/bin/env bash
set -euo pipefail

# Runs the one-shot Sub2API new-user welcome email command inside the app
# container. Install with cron on the host, for example:
#
#   0 22 * * * TZ=Asia/Shanghai SUB2API_CONTAINER=sub2api /path/to/deploy/send-new-user-welcome.sh >> /var/log/sub2api-new-user-welcome.log 2>&1
#
# For Dokploy or Compose deployments without a stable container name, run this
# script from the compose project directory or set SUB2API_PROJECT_DIR and,
# optionally, SUB2API_COMPOSE_FILE.

command=(/app/sub2api --send-new-user-welcome)

if [[ "${NEW_USER_WELCOME_DRY_RUN:-}" == "1" ]]; then
  command+=(--new-user-welcome-dry-run)
fi
if [[ -n "${NEW_USER_WELCOME_SINCE:-}" ]]; then
  command+=(--new-user-welcome-since "$NEW_USER_WELCOME_SINCE")
fi
if [[ -n "${NEW_USER_WELCOME_UNTIL:-}" ]]; then
  command+=(--new-user-welcome-until "$NEW_USER_WELCOME_UNTIL")
fi
if [[ -n "${NEW_USER_WELCOME_LIMIT:-}" ]]; then
  command+=(--new-user-welcome-limit "$NEW_USER_WELCOME_LIMIT")
fi
if [[ -n "${NEW_USER_WELCOME_SUBJECT:-}" ]]; then
  command+=(--new-user-welcome-subject "$NEW_USER_WELCOME_SUBJECT")
fi
if [[ -n "${NEW_USER_WELCOME_BODY:-}" ]]; then
  command+=(--new-user-welcome-body "$NEW_USER_WELCOME_BODY")
fi

if [[ -n "${SUB2API_CONTAINER:-}" ]]; then
  exec docker exec "$SUB2API_CONTAINER" "${command[@]}"
fi

if [[ -n "${SUB2API_PROJECT_DIR:-}" ]]; then
  cd "$SUB2API_PROJECT_DIR"
fi

compose=(docker compose)
if [[ -n "${SUB2API_COMPOSE_FILE:-}" ]]; then
  compose+=(-f "$SUB2API_COMPOSE_FILE")
fi

exec "${compose[@]}" exec -T "${SUB2API_SERVICE:-sub2api}" "${command[@]}"
