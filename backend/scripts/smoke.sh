#!/usr/bin/env bash
#
# smoke.sh — ürünü sıfırdan uçtan uca oynatır.
#
# verify.sh her gereksinimi ayrı ayrı kanıtlar; bu betik tek bir kullanıcının
# yolculuğunu baştan sona yürür. Bir gereksinim tek başına çalışırken zincirin
# tamamının çalışmaması mümkün: adımlar birbirinin çıktısına bağlı.
#
# Kullanım:
#   ./scripts/smoke.sh
#   KEEP_RUNNING=1 ./scripts/smoke.sh   # servisleri açık bırak
#   REUSE=1 ./scripts/smoke.sh          # zaten çalışan servislere bağlan
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
RUN_DIR="${RUN_DIR:-$BACKEND_DIR/tmp/smoke}"

cd "$BACKEND_DIR"
mkdir -p "$RUN_DIR"

export APP_ENV=development
export SERVER_PORT="${SERVER_PORT:-8080}"
export CORS_ALLOWED_ORIGINS="${CORS_ALLOWED_ORIGINS:-http://localhost:3000}"
export WEB_BASE_URL="${WEB_BASE_URL:-http://localhost:3000}"
export JWT_SECRET="${JWT_SECRET:-smoke-only-jwt-secret}"
export AUTH_CODE_PEPPER="${AUTH_CODE_PEPPER:-smoke-only-pepper}"
export EMAIL_VERIFICATION_OTP_PEPPER="${EMAIL_VERIFICATION_OTP_PEPPER:-smoke-only-email-verification-pepper}"
export EMAIL_PROVIDER=smtp
export EMAIL_FROM_ADDRESS="${EMAIL_FROM_ADDRESS:-noreply@mail.prova.local}"
export SMTP_HOST="${SMTP_HOST:-localhost}"
export SMTP_PORT="${SMTP_PORT:-1025}"
export LOG_FORMAT=text
export LOG_LEVEL="${LOG_LEVEL:-warn}"
export AUTH_CODE_RESEND_COOLDOWN_SECONDS=0
export AUTH_CODE_MAX_REQUESTS_PER_EMAIL=1000
export AUTH_CODE_MAX_REQUESTS_PER_IP=10000
export AUTH_MIN_RESPONSE_TIME_MS=0
export LIFECYCLE_DELETION_GRACE_SECONDS=0
export LIFECYCLE_PURGE_INTERVAL_SECONDS=5
export LIFECYCLE_PURGE_ENABLED=true

MOCKLLM_PORT="${MOCKLLM_PORT:-8099}"
MAILPIT_URL="${MAILPIT_URL:-http://localhost:8025}"
MOCKLLM_URL="http://localhost:${MOCKLLM_PORT}"

log()  { printf '\033[1;34m▸\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31m✗\033[0m %s\n' "$*" >&2; exit 1; }

cleanup() {
  [[ "${KEEP_RUNNING:-0}" == "1" ]] && return
  for name in server mockllm; do
    if [[ -f "$RUN_DIR/$name.pid" ]]; then
      kill "$(cat "$RUN_DIR/$name.pid")" 2>/dev/null || true
      rm -f "$RUN_DIR/$name.pid"
    fi
  done
}
trap cleanup EXIT

wait_for() {
  for ((i = 0; i < 40; i++)); do
    curl -fsS -o /dev/null "$1" 2>/dev/null && return 0
    sleep 0.5
  done
  return 1
}

if [[ "${REUSE:-0}" != "1" ]]; then
  log "altyapı başlatılıyor"
  docker compose -f deployments/docker-compose.yml up -d postgres mongo redis mailpit >/dev/null
  wait_for "$MAILPIT_URL/api/v1/messages" || die "mailpit ayağa kalkmadı"

  log "migration'lar ve tohum verisi"
  ./scripts/migrate.sh up >/dev/null 2>&1 || die "migration başarısız"

  log "sahte LLM sağlayıcısı"
  go build -o "$RUN_DIR/mockllm" ./cmd/mockllm || die "mockllm derlenemedi"
  "$RUN_DIR/mockllm" -addr ":$MOCKLLM_PORT" > "$RUN_DIR/mockllm.log" 2>&1 &
  echo $! > "$RUN_DIR/mockllm.pid"
  disown
  wait_for "$MOCKLLM_URL/v1/models" || die "sahte sağlayıcı ayağa kalkmadı"

  go run ./cmd/seed >/dev/null 2>&1 || die "tohumlama başarısız"
  docker exec prova-mongo mongosh --quiet prova --eval \
    "db.llm_profiles.updateMany({}, {\$set: {base_url: '$MOCKLLM_URL/v1'}})" >/dev/null

  log "sunucu başlatılıyor"
  go build -o "$RUN_DIR/server" ./cmd/server || die "sunucu derlenemedi"
  "$RUN_DIR/server" > "$RUN_DIR/server.log" 2>&1 &
  echo $! > "$RUN_DIR/server.pid"
  disown
  wait_for "http://localhost:${SERVER_PORT}/health/ready" || {
    tail -20 "$RUN_DIR/server.log" >&2
    die "sunucu ayağa kalkmadı"
  }
fi

echo
SMOKE_GRAPHQL_URL="http://localhost:${SERVER_PORT}/graphql" \
SMOKE_MAILPIT_URL="$MAILPIT_URL" \
SMOKE_MOCKLLM_URL="$MOCKLLM_URL" \
  go run ./cmd/smoke
