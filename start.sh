#!/usr/bin/env bash
#
# start.sh — Prova full-stack runner
#
# Brings the whole project up with a single command and takes it all down again
# when you stop it (Ctrl+C or ./start.sh stop).
#
#   Infrastructure : Postgres, Redis, Mongo, Mailpit (+ optional Kafka / Kafka UI)
#   Backend        : Go API with hot-reload (air)  → http://localhost:8080
#   Frontend (web) : Next.js dev server            → http://localhost:3000
#   Desktop        : Electron + Next.js renderer  → http://127.0.0.1:3100
#
# Usage:
#   ./start.sh            Full stack: infra + migrations + backend + frontend + desktop
#   ./start.sh infra      Infrastructure + migrations only
#   ./start.sh backend    Backend only (infra must already be running)
#   ./start.sh frontend   Web frontend only
#   ./start.sh desktop    Desktop (Electron) only
#   ./start.sh stop       Stop everything (apps + Docker services)
#   ./start.sh restart    Stop, then start the full stack
#   ./start.sh status     Show what is currently running
#   ./start.sh migrate    Run database migrations only
#   ./start.sh logs       Tail backend + frontend + desktop logs
#   ./start.sh clean      Stop everything, drop volumes, remove build artifacts
#   ./start.sh help       Show this help
#
set -euo pipefail

# ─── Configuration ────────────────────────────────────────────────────────────
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
FRONTEND_ROOT="$ROOT_DIR/frontend"
FRONTEND_DIR="$ROOT_DIR/frontend/web"
DESKTOP_DIR="$ROOT_DIR/frontend/desktop"
COMPOSE_FILE="$BACKEND_DIR/deployments/docker-compose.yml"
MIGRATION_DIR="$BACKEND_DIR/internal/infrastructure/postgres/migrations"

RUN_DIR="$ROOT_DIR/.run"
BACKEND_LOG="$RUN_DIR/backend.log"
FRONTEND_LOG="$RUN_DIR/frontend.log"
DESKTOP_LOG="$RUN_DIR/desktop.log"
STACK_PID_FILE="$RUN_DIR/stack.pid"
BACKEND_PID_FILE="$RUN_DIR/backend.pid"
FRONTEND_PID_FILE="$RUN_DIR/frontend.pid"
DESKTOP_PID_FILE="$RUN_DIR/desktop.pid"

DB_USER="masterfabric"
DB_NAME="masterfabric"
DB_CONTAINER="prova-postgres"

# Our own process group — used as a guard when signalling child groups.
SELF_PGID="$(ps -o pgid= -p $$ | tr -d ' ')"

BACKEND_PORT="${SERVER_PORT:-8080}"
FRONTEND_PORT="${FRONTEND_PORT:-3000}"
DESKTOP_PORT="${DESKTOP_PORT:-3100}"

# Build env — workaround for macOS sandbox permissions on /var/folders
export GOTMPDIR="$BACKEND_DIR/tmp"
export GOCACHE="$BACKEND_DIR/tmp/go-cache"
export TMPDIR="$BACKEND_DIR/tmp"
export CGO_ENABLED=0
export KAFKA_ENABLED="${KAFKA_ENABLED:-false}"
export PROVA_EMAIL="${PROVA_EMAIL:-mailpit}"

# Docker Compose project name — keeps Prova's containers, network and volumes
# grouped under "prova" instead of the compose file's parent directory.
export COMPOSE_PROJECT_NAME="prova"

# ─── Colors ───────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

log_info()  { echo -e "${CYAN}[INFO]${NC}  $*"; }
log_ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }
log_step()  { echo -e "\n${BOLD}━━━ $* ━━━${NC}"; }

# ─── Helpers ──────────────────────────────────────────────────────────────────

ensure_dirs() {
    mkdir -p "$RUN_DIR" "$BACKEND_DIR/tmp" "$BACKEND_DIR/bin"
}

# Load backend/.env into the environment (the Go app reads plain env vars),
# then fill in the local-dev defaults the server refuses to boot without.
load_env() {
    local env_file="$BACKEND_DIR/.env"
    if [[ -f "$env_file" ]]; then
        log_info "Loading environment from backend/.env"
        set -a
        # shellcheck disable=SC1090
        source "$env_file"
        set +a
    else
        log_warn "backend/.env not found — using local defaults (see backend/.env.example)"
    fi

    # Re-apply CLI/env overrides after sourcing .env (caller defaults win).
    export PROVA_EMAIL="${PROVA_EMAIL:-mailpit}"
    export KAFKA_ENABLED="${KAFKA_ENABLED:-false}"

    case "${PROVA_EMAIL}" in
        mailpit|smtp|local)
            export EMAIL_PROVIDER=smtp
            export SMTP_HOST="${SMTP_HOST:-localhost}"
            export SMTP_PORT="${SMTP_PORT:-1025}"
            export SMTP_USE_TLS="${SMTP_USE_TLS:-false}"
            # Ensure a From address for local SMTP / Mailpit delivery.
            if [[ -z "${EMAIL_FROM_ADDRESS:-}" && -z "${RESEND_FROM_EMAIL:-}" ]]; then
                export EMAIL_FROM_ADDRESS="${EMAIL_FROM_ADDRESS:-Prova <prova@localhost>}"
            fi
            if [[ -z "${EMAIL_FROM_NAME:-}" ]]; then
                export EMAIL_FROM_NAME=Prova
            fi
            log_info "Email mode: PROVA_EMAIL=${PROVA_EMAIL} → EMAIL_PROVIDER=smtp (${SMTP_HOST}:${SMTP_PORT})"
            ;;
        resend)
            export EMAIL_PROVIDER=resend
            if [[ -z "${RESEND_API_KEY:-}" ]]; then
                log_warn "RESEND_API_KEY is empty — starting with EMAIL_PROVIDER=none (login codes are not delivered)"
                export EMAIL_PROVIDER=none
            else
                log_info "Email mode: PROVA_EMAIL=resend → Resend provider"
            fi
            ;;
        none)
            export EMAIL_PROVIDER=none
            log_info "Email mode: PROVA_EMAIL=none → EMAIL_PROVIDER=none"
            ;;
        *)
            log_warn "Unknown PROVA_EMAIL=${PROVA_EMAIL} — leaving EMAIL_PROVIDER=${EMAIL_PROVIDER:-unset}"
            if [[ "${EMAIL_PROVIDER:-resend}" == "resend" && -z "${RESEND_API_KEY:-}" ]]; then
                log_warn "RESEND_API_KEY is empty — starting with EMAIL_PROVIDER=none"
                export EMAIL_PROVIDER=none
            fi
            ;;
    esac

    if [[ -n "${LLM_API_KEY:-}" || -n "${LLM_API_KEYS:-}" ]]; then
        log_ok "LLM API key: present"
    else
        log_warn "LLM API key: missing (LLM_API_KEY / LLM_API_KEYS empty)"
    fi
}

check_docker() {
    if ! docker info &>/dev/null; then
        log_warn "Docker is not running. Attempting to start Docker Desktop..."
        open -a Docker 2>/dev/null || true
        local retries=30
        for i in $(seq 1 $retries); do
            docker info &>/dev/null && break
            printf "  waiting for Docker daemon... (%d/%d)\r" "$i" "$retries"
            sleep 2
        done
        echo ""
        if ! docker info &>/dev/null; then
            log_error "Docker daemon failed to start. Please start Docker manually."
            exit 1
        fi
        log_ok "Docker is ready"
    fi
}

port_pids() {
    lsof -ti:"$1" 2>/dev/null || true
}

kill_port() {
    local port="$1" pids
    pids=$(port_pids "$port")
    [[ -n "$pids" ]] && echo "$pids" | xargs kill -9 2>/dev/null || true
}

# Terminate a pid file's process together with its whole process group,
# so children (air's built binary, next-server, electron) do not survive.
stop_pidfile() {
    local pid_file="$1" label="$2" pid pgid
    [[ -f "$pid_file" ]] || return 0
    pid=$(cat "$pid_file" 2>/dev/null || echo "")
    rm -f "$pid_file"
    [[ -n "$pid" ]] || return 0
    kill -0 "$pid" 2>/dev/null || return 0

    pgid=$(ps -o pgid= -p "$pid" 2>/dev/null | tr -d ' ' || echo "")
    # Never signal our own process group — that would kill this script too.
    [[ "$pgid" == "$SELF_PGID" ]] && pgid=""
    if [[ -n "$pgid" && "$pgid" != "0" ]]; then
        kill -TERM -"$pgid" 2>/dev/null || kill -TERM "$pid" 2>/dev/null || true
    else
        kill -TERM "$pid" 2>/dev/null || true
    fi

    for _ in $(seq 1 20); do
        kill -0 "$pid" 2>/dev/null || break
        sleep 0.25
    done
    if kill -0 "$pid" 2>/dev/null; then
        if [[ -n "$pgid" && "$pgid" != "0" ]]; then
            kill -9 -"$pgid" 2>/dev/null || kill -9 "$pid" 2>/dev/null || true
        else
            kill -9 "$pid" 2>/dev/null || true
        fi
    fi
    log_ok "$label stopped"
}

install_air() {
    export PATH="$BACKEND_DIR/bin:$PATH"
    if ! command -v air &>/dev/null; then
        log_info "Installing air (hot-reload tool)..."
        GOBIN="$BACKEND_DIR/bin" go install github.com/air-verse/air@latest 2>/dev/null \
            || go install github.com/air-verse/air@latest 2>/dev/null || true
    fi
    if ! command -v air &>/dev/null; then
        log_error "Failed to install air. Install manually: go install github.com/air-verse/air@latest"
        exit 1
    fi
    log_ok "air is available: $(command -v air)"
}

install_frontend_deps() {
    if [[ ! -d "$FRONTEND_ROOT/node_modules" ]]; then
        log_info "Installing frontend dependencies (npm install in frontend/)..."
        (cd "$FRONTEND_ROOT" && npm install)
    fi
}

wait_for_http() {
    local url="$1" label="$2" retries="${3:-60}" pid="${4:-}"
    for i in $(seq 1 "$retries"); do
        if curl -fsS -o /dev/null --max-time 2 "$url" 2>/dev/null; then
            log_ok "$label is up ($url)"
            return 0
        fi
        if [[ -n "$pid" ]] && ! kill -0 "$pid" 2>/dev/null; then
            log_error "$label exited before becoming ready"
            return 1
        fi
        sleep 1
    done
    log_warn "$label did not answer at $url yet — check the logs"
    return 1
}

wait_healthy() {
    local svc="$1" retries="${2:-30}"
    local i health
    for i in $(seq 1 "$retries"); do
        health=$(docker inspect --format='{{.State.Health.Status}}' "$svc" 2>/dev/null || echo "missing")
        if [[ "$health" == "healthy" ]]; then
            log_ok "$svc is healthy"
            return 0
        fi
        if [[ $i -eq $retries ]]; then
            log_warn "$svc did not become healthy (status: $health)"
            return 1
        fi
        sleep 2
    done
}

# ─── Infrastructure ───────────────────────────────────────────────────────────

start_infra() {
    log_step "Starting infrastructure"
    check_docker

    log_info "Starting Docker Compose services..."
    docker compose -f "$COMPOSE_FILE" up -d

    log_info "Waiting for core services to become healthy..."
    local services=("prova-postgres" "prova-redis" "prova-mongo" "prova-mailpit")
    local svc
    for svc in "${services[@]}"; do
        wait_healthy "$svc" || true
    done

    if [[ "${KAFKA_ENABLED}" == "true" ]]; then
        log_info "KAFKA_ENABLED=true — starting Kafka profile..."
        docker compose -f "$COMPOSE_FILE" --profile kafka up -d
        wait_healthy "prova-kafka" || true
    else
        log_info "KAFKA_ENABLED=false — skipping Kafka / Kafka UI"
    fi

    echo ""
    docker compose -f "$COMPOSE_FILE" ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}"
    echo ""
}

stop_infra() {
    log_step "Stopping infrastructure"
    if docker info &>/dev/null; then
        # Include kafka profile so profiled services are torn down when present.
        docker compose -f "$COMPOSE_FILE" --profile kafka down
        log_ok "Docker services stopped"
    else
        log_warn "Docker is not running — nothing to stop"
    fi
}

# ─── Migrations ───────────────────────────────────────────────────────────────

run_migrations() {
    log_step "Running database migrations"

    if ! docker ps --format '{{.Names}}' 2>/dev/null | grep -q "^${DB_CONTAINER}$"; then
        log_warn "$DB_CONTAINER is not running — skipping migrations"
        return 0
    fi

    local count=0
    for f in "$MIGRATION_DIR"/0*.sql; do
        [[ -f "$f" ]] || continue
        local fname sql
        fname=$(basename "$f")
        # Extract only the UP part (between "-- +goose Up" and "-- +goose Down")
        sql=$(sed -n '/^-- +goose Up$/,/^-- +goose Down$/p' "$f" | sed '1d;$d')
        if [[ -n "$sql" ]]; then
            echo "$sql" | docker exec -i "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -q 2>/dev/null \
                && log_ok "  $fname" \
                || log_warn "  $fname (may already exist)"
            count=$((count + 1))
        fi
    done

    log_ok "Processed $count migration files"
}

# ─── Applications ─────────────────────────────────────────────────────────────

start_backend() {
    log_step "Starting backend (hot-reload)"
    ensure_dirs
    load_env
    install_air

    stop_pidfile "$BACKEND_PID_FILE" "Previous backend"
    kill_port "$BACKEND_PORT"

    : > "$BACKEND_LOG"
    set -m   # own process group, so the whole tree can be stopped later
    (
        cd "$BACKEND_DIR"
        exec air -c .air.toml
    ) >>"$BACKEND_LOG" 2>&1 &
    echo $! > "$BACKEND_PID_FILE"
    set +m

    log_ok "Backend started (pid $(cat "$BACKEND_PID_FILE")) → log: .run/backend.log"
    wait_for_http "http://localhost:$BACKEND_PORT/health/live" "Backend" 90 || true
}

start_frontend() {
    log_step "Starting frontend (Next.js web)"
    ensure_dirs
    install_frontend_deps

    stop_pidfile "$FRONTEND_PID_FILE" "Previous frontend"
    kill_port "$FRONTEND_PORT"

    : > "$FRONTEND_LOG"
    set -m   # own process group, so the whole tree can be stopped later
    (
        cd "$FRONTEND_DIR"
        exec npm run dev -- --port "$FRONTEND_PORT"
    ) >>"$FRONTEND_LOG" 2>&1 &
    echo $! > "$FRONTEND_PID_FILE"
    set +m

    log_ok "Frontend started (pid $(cat "$FRONTEND_PID_FILE")) → log: .run/frontend.log"
    wait_for_http "http://localhost:$FRONTEND_PORT" "Frontend" 90 || true
}

start_desktop() {
    log_step "Starting desktop (Electron + Next.js renderer)"
    ensure_dirs
    install_frontend_deps

    stop_pidfile "$DESKTOP_PID_FILE" "Previous desktop"
    kill_port "$DESKTOP_PORT"

    : > "$DESKTOP_LOG"
    set -m
    (
        cd "$DESKTOP_DIR"
        export PROVA_GRAPHQL_URL="http://127.0.0.1:${BACKEND_PORT}/graphql"
        export NEXT_PUBLIC_GRAPHQL_URL="http://127.0.0.1:${BACKEND_PORT}/graphql"
        exec npm run dev
    ) >>"$DESKTOP_LOG" 2>&1 &
    echo $! > "$DESKTOP_PID_FILE"
    set +m

    local desktop_pid
    desktop_pid=$(cat "$DESKTOP_PID_FILE")
    log_ok "Desktop started (pid $desktop_pid) → log: .run/desktop.log"
    wait_for_http "http://127.0.0.1:$DESKTOP_PORT" "Desktop" 90 "$desktop_pid"
}

stop_apps() {
    log_step "Stopping applications"
    stop_pidfile "$DESKTOP_PID_FILE" "Desktop"
    stop_pidfile "$FRONTEND_PID_FILE" "Frontend"
    stop_pidfile "$BACKEND_PID_FILE" "Backend"
    kill_port "$DESKTOP_PORT"
    kill_port "$BACKEND_PORT"
    kill_port "$FRONTEND_PORT"
}

print_endpoints() {
    echo ""
    echo -e "${BOLD}Prova is running (full stack)${NC}"
    echo -e "  Frontend:   ${GREEN}http://localhost:$FRONTEND_PORT${NC}"
    echo -e "  Desktop:    ${GREEN}http://127.0.0.1:$DESKTOP_PORT${NC}  (Electron loads this)"
    echo -e "  API:        ${GREEN}http://localhost:$BACKEND_PORT${NC}"
    echo -e "  GraphQL:    http://127.0.0.1:$BACKEND_PORT/graphql"
    echo -e "  Health:     http://localhost:$BACKEND_PORT/health/ready"
    echo -e "  Metrics:    http://localhost:$BACKEND_PORT/metrics"
    echo -e "  Mailpit:    ${GREEN}http://localhost:8025${NC}  (SMTP → localhost:1025)"
    if [[ "${KAFKA_ENABLED}" == "true" ]]; then
        echo -e "  Kafka UI:   http://localhost:8090"
    else
        echo -e "  Kafka:      disabled (set KAFKA_ENABLED=true to enable)"
    fi
    if [[ -n "${LLM_API_KEY:-}" || -n "${LLM_API_KEYS:-}" ]]; then
        echo -e "  LLM:        key present"
    else
        echo -e "  LLM:        ${YELLOW}key missing${NC} — set LLM_API_KEY in backend/.env"
    fi
    echo ""
    echo -e "  Logs:       .run/backend.log · .run/frontend.log · .run/desktop.log"
    echo -e "  Stop:       ${BOLD}Ctrl+C${NC} (or ./start.sh stop from another shell)"
    echo ""
}

# ─── Shutdown ─────────────────────────────────────────────────────────────────

SHUTTING_DOWN=0

shutdown_all() {
    [[ "$SHUTTING_DOWN" -eq 1 ]] && return 0
    SHUTTING_DOWN=1
    trap - INT TERM EXIT
    echo ""
    log_step "Shutting down Prova"
    [[ -n "${TAIL_PID:-}" ]] && kill "$TAIL_PID" 2>/dev/null || true
    rm -f "$STACK_PID_FILE"
    stop_apps
    stop_infra
    log_ok "Everything is down"
}

# ─── Commands ─────────────────────────────────────────────────────────────────

cmd_up() {
    ensure_dirs
    trap shutdown_all INT TERM EXIT
    echo $$ > "$STACK_PID_FILE"

    start_infra
    run_migrations
    start_backend
    start_frontend
    start_desktop
    print_endpoints

    log_info "Streaming logs (Ctrl+C stops the whole stack)..."
    echo ""
    tail -n 0 -f "$BACKEND_LOG" "$FRONTEND_LOG" "$DESKTOP_LOG" &
    TAIL_PID=$!
    wait "$TAIL_PID" 2>/dev/null || true
    shutdown_all
}

cmd_infra() {
    ensure_dirs
    start_infra
    run_migrations
    log_ok "Infrastructure is ready."
}

cmd_backend() {
    ensure_dirs
    trap shutdown_apps_only INT TERM EXIT
    start_backend
    log_info "Streaming backend log (Ctrl+C stops it)..."
    tail -n 0 -f "$BACKEND_LOG" &
    TAIL_PID=$!
    wait "$TAIL_PID" 2>/dev/null || true
}

cmd_frontend() {
    ensure_dirs
    trap shutdown_apps_only INT TERM EXIT
    start_frontend
    log_info "Streaming frontend log (Ctrl+C stops it)..."
    tail -n 0 -f "$FRONTEND_LOG" &
    TAIL_PID=$!
    wait "$TAIL_PID" 2>/dev/null || true
}

cmd_desktop() {
    ensure_dirs
    trap shutdown_apps_only INT TERM EXIT
    start_desktop
    log_info "Streaming desktop log (Ctrl+C stops it)..."
    tail -n 0 -f "$DESKTOP_LOG" &
    TAIL_PID=$!
    wait "$TAIL_PID" 2>/dev/null || true
}

shutdown_apps_only() {
    [[ "$SHUTTING_DOWN" -eq 1 ]] && return 0
    SHUTTING_DOWN=1
    trap - INT TERM EXIT
    echo ""
    [[ -n "${TAIL_PID:-}" ]] && kill "$TAIL_PID" 2>/dev/null || true
    stop_apps
}

cmd_stop() {
    stop_supervisor
    stop_apps
    stop_infra
    log_ok "Everything is down"
}

# A foreground ./start.sh is tailing logs in another shell: ask it to shut the
# stack down itself, so it does not keep running against a torn-down stack.
stop_supervisor() {
    local pid
    pid=$(cat "$STACK_PID_FILE" 2>/dev/null || echo "")
    [[ -n "$pid" && "$pid" != "$$" ]] || return 0
    kill -0 "$pid" 2>/dev/null || { rm -f "$STACK_PID_FILE"; return 0; }

    log_info "Signalling running ./start.sh (pid $pid) to shut down..."
    kill -TERM "$pid" 2>/dev/null || true
    for _ in $(seq 1 40); do
        kill -0 "$pid" 2>/dev/null || break
        sleep 0.5
    done
    kill -0 "$pid" 2>/dev/null && kill -9 "$pid" 2>/dev/null || true
    rm -f "$STACK_PID_FILE"
}

cmd_restart() {
    cmd_stop
    cmd_up
}

cmd_status() {
    log_step "Application processes"
    local shown=0
    local entry name pid_file port pid
    for entry in \
        "backend:$BACKEND_PID_FILE:$BACKEND_PORT" \
        "frontend:$FRONTEND_PID_FILE:$FRONTEND_PORT" \
        "desktop:$DESKTOP_PID_FILE:$DESKTOP_PORT"
    do
        IFS=':' read -r name pid_file port <<< "$entry"
        pid=$(cat "$pid_file" 2>/dev/null || echo "")
        if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
            log_ok "$name running (pid $pid, port $port)"
        elif [[ -n "$(port_pids "$port")" ]]; then
            log_warn "$name not tracked, but port $port is in use by pid(s): $(port_pids "$port" | tr '\n' ' ')"
        else
            log_warn "$name is not running"
        fi
        shown=1
    done
    [[ $shown -eq 1 ]] || log_warn "no application processes tracked"

    log_step "Docker services"
    if docker info &>/dev/null; then
        docker compose -f "$COMPOSE_FILE" ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}"
    else
        log_warn "Docker is not running"
    fi
}

cmd_migrate() {
    run_migrations
}

cmd_logs() {
    local files=()
    [[ -f "$BACKEND_LOG" ]] && files+=("$BACKEND_LOG")
    [[ -f "$FRONTEND_LOG" ]] && files+=("$FRONTEND_LOG")
    [[ -f "$DESKTOP_LOG" ]] && files+=("$DESKTOP_LOG")
    if [[ ${#files[@]} -eq 0 ]]; then
        log_warn "No application logs yet. Start the stack with ./start.sh"
        return 0
    fi
    tail -n 50 -f "${files[@]}"
}

cmd_clean() {
    log_step "Cleaning up"
    stop_apps
    if docker info &>/dev/null; then
        docker compose -f "$COMPOSE_FILE" --profile kafka down -v 2>/dev/null || true
    fi
    rm -rf "$BACKEND_DIR/tmp" "$BACKEND_DIR/bin/server" "$BACKEND_DIR/.tmp" "$RUN_DIR"
    rm -rf "$FRONTEND_DIR/.next" "$DESKTOP_DIR/.next"
    log_ok "Cleaned: Docker volumes, backend tmp/, frontend+desktop .next/, .run/"
}

cmd_help() {
    echo -e "${BOLD}Prova full-stack runner${NC}"
    echo ""
    echo "Starts the FULL stack by default: infra + migrations + backend + web + desktop."
    echo ""
    echo "Usage: ./start.sh [command]"
    echo ""
    echo "Commands:"
    echo -e "  ${GREEN}(default)${NC}         Start everything: infra + migrations + backend + frontend + desktop"
    echo -e "  ${GREEN}infra${NC}             Start infrastructure + migrations only"
    echo -e "  ${GREEN}backend${NC}           Start backend only (hot-reload)"
    echo -e "  ${GREEN}frontend${NC}          Start web frontend only"
    echo -e "  ${GREEN}desktop${NC}|electron  Start desktop (Electron) only"
    echo -e "  ${GREEN}stop${NC}              Stop applications and Docker services"
    echo -e "  ${GREEN}restart${NC}           Stop everything, then start the full stack"
    echo -e "  ${GREEN}status${NC}            Show what is running"
    echo -e "  ${GREEN}migrate${NC}           Run database migrations"
    echo -e "  ${GREEN}logs${NC}              Tail backend + frontend + desktop logs"
    echo -e "  ${GREEN}clean${NC}             Stop everything, drop volumes, remove build artifacts"
    echo -e "  ${GREEN}help${NC}              Show this help message"
    echo ""
    echo "Environment:"
    echo "  KAFKA_ENABLED=false     (default: false; set true for Kafka + Kafka UI)"
    echo "  PROVA_EMAIL=mailpit     (default: mailpit; also: smtp|local|resend|none)"
    echo "  SERVER_PORT=8080        backend port"
    echo "  FRONTEND_PORT=3000      web frontend port"
    echo "  DESKTOP_PORT=3100       desktop renderer port"
    echo "  backend/.env is sourced automatically when present."
    echo ""
    echo "Endpoints (when running):"
    echo "  Frontend:   http://localhost:3000"
    echo "  Desktop:    http://127.0.0.1:3100"
    echo "  API:        http://localhost:8080"
    echo "  GraphQL:    http://127.0.0.1:8080/graphql"
    echo "  Health:     http://localhost:8080/health/ready"
    echo "  Metrics:    http://localhost:8080/metrics"
    echo "  Mailpit:    http://localhost:8025"
    echo "  Kafka UI:   http://localhost:8090  (only if KAFKA_ENABLED=true)"
}

# ─── Main ─────────────────────────────────────────────────────────────────────

case "${1:-}" in
    infra)              cmd_infra    ;;
    backend|server)     cmd_backend  ;;
    frontend|web)       cmd_frontend ;;
    desktop|electron)   cmd_desktop  ;;
    stop|down)          cmd_stop     ;;
    restart)            cmd_restart  ;;
    status|ps)          cmd_status   ;;
    migrate)            cmd_migrate  ;;
    logs)               cmd_logs     ;;
    clean)              cmd_clean    ;;
    help|-h|--help)     cmd_help     ;;
    up|start|"")        cmd_up       ;;
    *)                  log_error "Unknown command: $1"; echo ""; cmd_help; exit 1 ;;
esac
