#!/usr/bin/env bash
set -euo pipefail

# build-linux.sh
#
# Convenience script to install prerequisites and build Melodee
# on common Linux distros (Ubuntu/Debian, Arch/Manjaro).
#
# This is intentionally conservative: it installs system packages
# if the relevant tools are missing, then builds the Go services
# and (optionally) the frontend.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SRC_DIR="$ROOT_DIR/src"
FRONTEND_DIR="$SRC_DIR/api/frontend"

bold() { printf '\033[1m%s\033[0m\n' "$*"; }
info() { printf '[INFO] %s\n' "$*"; }
warn() { printf '[WARN] %s\n' "$*" >&2; }
err()  { printf '[ERROR] %s\n' "$*" >&2; }

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    return 1
  fi
}

# Detect basic distro family
DETECTED_DISTRIB="unknown"
if [[ -f /etc/os-release ]]; then
  # shellcheck disable=SC1091
  . /etc/os-release
  case "${ID_LIKE:-$ID}" in
    *debian*|*ubuntu*) DETECTED_DISTRIB="debian" ;;
    *arch*)            DETECTED_DISTRIB="arch"   ;;
  esac
fi

install_deps_debian() {
  bold "Installing system packages via apt (Debian/Ubuntu)..."
  sudo apt-get update
  sudo apt-get install -y \
    build-essential \
    curl git \
    golang \
    nodejs npm \
    postgresql postgresql-contrib \
    redis-server \
    ffmpeg
}

install_deps_arch() {
  bold "Installing system packages via pacman (Arch/Manjaro)..."
  sudo pacman -Syu --noconfirm
  sudo pacman -S --needed --noconfirm \
    base-devel \
    curl git \
    go \
    nodejs npm \
    postgresql \
    redis \
    ffmpeg
}

install_prereqs() {
  missing=()
  for c in go node npm psql redis-server ffmpeg; do
    if ! require_cmd "$c"; then
      missing+=("$c")
    fi
  done

  if ((${#missing[@]} == 0)); then
    info "All required tools already present: go, node, npm, psql, redis, ffmpeg."
    return 0
  fi

  info "Missing tools: ${missing[*]}"

  case "$DETECTED_DISTRIB" in
    debian)
      install_deps_debian
      ;;
    arch)
      install_deps_arch
      ;;
    *)
      warn "Unknown distro; please install these tools manually: ${missing[*]}"
      return 1
      ;;
  esac
}

create_postgres_db() {
  local db_name="melodee_dev"
  local db_user="melodee"

  if ! command -v psql >/dev/null 2>&1; then
    warn "psql not found; skipping automatic database creation."
    return 0
  fi

  bold "Ensuring PostgreSQL user/database exist (user=$db_user, db=$db_name)..."
  if sudo -u postgres psql -tAc "SELECT 1 FROM pg_roles WHERE rolname='$db_user'" | grep -q 1; then
    info "PostgreSQL role '$db_user' already exists."
  else
    sudo -u postgres createuser --superuser "$db_user" || warn "Could not create role '$db_user' (it may already exist)."
  fi

  if sudo -u postgres psql -tAc "SELECT 1 FROM pg_database WHERE datname='$db_name'" | grep -q 1; then
    info "Database '$db_name' already exists."
  else
    sudo -u postgres createdb "$db_name" -O "$db_user" || warn "Could not create database '$db_name' (it may already exist)."
  fi
}

setup_env_file() {
  local env_file="$ROOT_DIR/.env.dev"
  if [[ -f "$env_file" ]]; then
    info "Found existing $env_file; not overwriting."
    return 0
  fi

  bold "Creating $env_file with sensible local defaults..."
  cat >"$env_file" <<'EOF'
MELODEE_DB_HOST=127.0.0.1
MELODEE_DB_PORT=5432
MELODEE_DB_USER=melodee
MELODEE_DB_PASSWORD=melodee
MELODEE_DB_NAME=melodee_dev
MELODEE_DB_SSLMODE=disable

MELODEE_REDIS_ADDR=127.0.0.1
MELODEE_REDIS_PORT=6379

MELODEE_SERVER_HOST=127.0.0.1
MELODEE_SERVER_PORT=8080

MELODEE_JWT_SECRET=dev-secret-change-me
EOF

  info "You can adjust settings in $env_file as needed."
}

export_env() {
  local env_file="$ROOT_DIR/.env.dev"
  if [[ -f "$env_file" ]]; then
    # shellcheck disable=SC2046
    export $(grep -v '^#' "$env_file" | xargs -d '\n' || true)
    info "Loaded environment from $env_file."
  else
    warn "No $env_file found; relying on existing environment variables."
  fi
}

build_go_services() {
  bold "Building Go services..."
  pushd "$SRC_DIR" >/dev/null

  info "Synchronizing go.work (if present)..."
  if [[ -f "$SRC_DIR/go.work" ]]; then
    go work sync || true
  fi

  info "Tidying Go modules..."
  go mod tidy

  info "Building root service (melodee)..."
  go build -o "$ROOT_DIR/bin/melodee" .

  info "Building API service..."
  go build -o "$ROOT_DIR/bin/melodee-api" ./api

  info "Building Web service..."
  go build -o "$ROOT_DIR/bin/melodee-web" ./web

  info "Building Worker service..."
  go build -o "$ROOT_DIR/bin/melodee-worker" ./worker

  popd >/dev/null
}

build_frontend() {
  if [[ ! -d "$FRONTEND_DIR" ]]; then
    warn "Frontend directory $FRONTEND_DIR not found; skipping frontend build."
    return 0
  fi

  bold "Building frontend (Vite + React)..."
  pushd "$FRONTEND_DIR" >/dev/null

  if [[ ! -d node_modules ]]; then
    info "Installing frontend dependencies via npm..."
    npm install
  fi

  info "Running npm run build..."
  npm run build

  popd >/dev/null
}

main() {
  bold "Melodee Linux build script"
  info "Root directory: $ROOT_DIR"
  info "Detected distro family: $DETECTED_DISTRIB"

  install_prereqs || warn "Prereq installation failed or skipped; ensure Go, Node, Postgres, Redis, FFmpeg are installed."
  create_postgres_db || warn "Database setup failed or was skipped."
  setup_env_file
  export_env
  build_go_services
  build_frontend

  bold "Build complete. Binaries should be in $ROOT_DIR/bin."
  info "Typical usage (from repo root):"
  info "  ./bin/melodee-api"
  info "  ./bin/melodee-web"
  info "  ./bin/melodee-worker"
}

main "$@"
