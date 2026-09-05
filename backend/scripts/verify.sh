#!/usr/bin/env bash
#
# verify.sh — on iki gereksinimi mekanik olarak doğrular.
#
# Bağımlılıkları başlatır, migration'ları uygular, tohum verisini yazar,
# sahte LLM sağlayıcısını ve sunucuyu ayağa kaldırır, sonra cmd/verify'ı
# çalıştırır. Herhangi bir gereksinim FAIL ise çıkış kodu 1'dir.
#
# Kullanım:
#   ./scripts/verify.sh            # her şeyi başlat, doğrula, kapat
#   KEEP_RUNNING=1 ./scripts/verify.sh   # doğrulamadan sonra servisleri bırak
#   ONLY=5 ./scripts/verify.sh     # yalnızca 5. gereksinimi çalıştır
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_DIR="$(cd "$BACKEND_DIR/.." && pwd)"
RUN_DIR="${RUN_DIR:-$BACKEND_DIR/tmp/verify}"

cd "$BACKEND_DIR"
mkdir -p "$RUN_DIR"

# Doğrulama ortamı.
#
# Bekleme süreleri ve hız limitleri bilerek gevşetiliyor: doğrulama zincirin
# tamamını dakikalar içinde görmek zorunda, ve üretim varsayılanlarıyla
# çalışan bir silme kontrolü otuz gün beklerdi. Değerler burada, .env'de
# değil — doğrulamanın kendi ayarları geliştiricinin ayarlarına sızmamalı.
export APP_ENV=development
export SERVER_PORT="${SERVER_PORT:-8080}"
export CORS_ALLOWED_ORIGINS="${CORS_ALLOWED_ORIGINS:-http://localhost:3000}"
export WEB_BASE_URL="${WEB_BASE_URL:-http://localhost:3000}"
export JWT_SECRET="${JWT_SECRET:-verify-only-jwt-secret}"
export AUTH_CODE_PEPPER="${AUTH_CODE_PEPPER:-verify-only-pepper}"
export EMAIL_VERIFICATION_OTP_PEPPER="${EMAIL_VERIFICATION_OTP_PEPPER:-verify-only-email-verification-pepper}"
export EMAIL_PROVIDER=smtp
export EMAIL_FROM_ADDRESS="${EMAIL_FROM_ADDRESS:-noreply@mail.prova.local}"
export SMTP_HOST="${SMTP_HOST:-localhost}"
export SMTP_PORT="${SMTP_PORT:-1025}"
export LOG_FORMAT=text
export LOG_LEVEL="${LOG_LEVEL:-warn}"
# Art arda kod istenebilsin diye limitler kapalı.
export AUTH_CODE_RESEND_COOLDOWN_SECONDS=0
export AUTH_CODE_MAX_REQUESTS_PER_EMAIL=1000
export AUTH_CODE_MAX_REQUESTS_PER_IP=10000
export AUTH_MIN_RESPONSE_TIME_MS=0
# Silme zinciri gözlemlenebilsin diye pencere sıfır ve iş sık çalışıyor.
export LIFECYCLE_DELETION_GRACE_SECONDS=0
export LIFECYCLE_PURGE_INTERVAL_SECONDS=5
export LIFECYCLE_PURGE_ENABLED=true
# Devre kesici doğrulama sırasında failover'ı kolayca göstersin.
export LLM_CIRCUIT_THRESHOLD=1
export LLM_CIRCUIT_COOLDOWN_SECONDS=2

MOCKLLM_PORT="${MOCKLLM_PORT:-8099}"
GRAPHQL_URL="http://localhost:${SERVER_PORT}/graphql"
MAILPIT_URL="${MAILPIT_URL:-http://localhost:8025}"
MOCKLLM_URL="http://localhost:${MOCKLLM_PORT}"

log()  { printf '\033[1;34m▸\033[0m %s\n' "$*"; }
fail() { printf '\033[1;31m✗\033[0m %s\n' "$*" >&2; exit 1; }

cleanup() {
  if [[ "${KEEP_RUNNING:-0}" == "1" ]]; then
    log "servisler açık bırakıldı (KEEP_RUNNING=1)"
    return
  fi
  for name in server mockllm; do
    if [[ -f "$RUN_DIR/$name.pid" ]]; then
      kill "$(cat "$RUN_DIR/$name.pid")" 2>/dev/null || true
      rm -f "$RUN_DIR/$name.pid"
    fi
  done
}
trap cleanup EXIT

wait_for() {
  local url="$1" name="$2" attempts="${3:-40}"
  for ((i = 0; i < attempts; i++)); do
    if curl -fsS -o /dev/null "$url" 2>/dev/null; then
      return 0
    fi
    sleep 0.5
  done
  return 1
}

# --- 1. Altyapı ---
log "altyapı başlatılıyor (postgres, mongo, redis, mailpit)"
docker compose -f deployments/docker-compose.yml up -d postgres mongo redis mailpit >/dev/null
for ((i = 0; i < 60; i++)); do
  ready=$(docker compose -f deployments/docker-compose.yml ps --format json 2>/dev/null \
    | grep -c '"Health":"healthy"' || true)
  [[ "$ready" -ge 3 ]] && break
  sleep 1
done
wait_for "$MAILPIT_URL/api/v1/messages" "mailpit" || fail "mailpit ayağa kalkmadı"

# --- 2. Migration ve tohum verisi ---
log "migration'lar uygulanıyor"
./scripts/migrate.sh up >/dev/null 2>&1 || fail "migration başarısız"

log "sahte LLM sağlayıcısı başlatılıyor (:$MOCKLLM_PORT)"
go build -o "$RUN_DIR/mockllm" ./cmd/mockllm || fail "mockllm derlenemedi"
"$RUN_DIR/mockllm" -addr ":$MOCKLLM_PORT" > "$RUN_DIR/mockllm.log" 2>&1 &
echo $! > "$RUN_DIR/mockllm.pid"
disown
wait_for "$MOCKLLM_URL/v1/models" "mockllm" || fail "sahte sağlayıcı ayağa kalkmadı"

log "tohum verisi yazılıyor"
go run ./cmd/seed >/dev/null 2>&1 || fail "tohumlama başarısız"

# Tohum profilleri sahte sağlayıcıya yönlendiriliyor. Gerçek bir API anahtarı
# olmadan uçtan uca akışı çalıştırmanın ve hataları kasıtlı üretmenin başka
# yolu yok.
log "LLM profilleri sahte sağlayıcıya yönlendiriliyor"
docker exec prova-mongo mongosh --quiet prova --eval \
  "db.llm_profiles.updateMany({}, {\$set: {base_url: '$MOCKLLM_URL/v1'}})" >/dev/null \
  || fail "profiller güncellenemedi"

# --- 3. Şema dışa aktarımı ---
log "GraphQL şeması SDL olarak dışa aktarılıyor"
go run ./cmd/schema -out "$REPO_DIR/schema.graphql" >/dev/null || fail "şema dışa aktarılamadı"

# --- 4. Sunucu ---
log "sunucu başlatılıyor (:$SERVER_PORT)"
go build -o "$RUN_DIR/server" ./cmd/server || fail "sunucu derlenemedi"
"$RUN_DIR/server" > "$RUN_DIR/server.log" 2>&1 &
echo $! > "$RUN_DIR/server.pid"
disown
wait_for "http://localhost:${SERVER_PORT}/health/ready" "server" || {
  tail -20 "$RUN_DIR/server.log" >&2
  fail "sunucu ayağa kalkmadı"
}

# --- 5. Doğrulama ---
log "gereksinim doğrulaması çalıştırılıyor"
echo
VERIFY_GRAPHQL_URL="$GRAPHQL_URL" \
VERIFY_MAILPIT_URL="$MAILPIT_URL" \
VERIFY_MOCKLLM_URL="$MOCKLLM_URL" \
  go run ./cmd/verify ${ONLY:+-only "$ONLY"}
