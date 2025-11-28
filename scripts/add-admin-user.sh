#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 3 ]; then
  echo "Usage: $0 <username> <email> <password>" >&2
  exit 1
fi

USERNAME="$1"
EMAIL="$2"
PASSWORD="$3"

# Load .env if present for consistency
if [ -f "$(dirname "$0")/../.env" ]; then
  set -a
  # shellcheck disable=SC1091
  source "$(dirname "$0")/../.env"
  set +a
fi

# Respect new standardized env names first, then legacy, then defaults
DB_HOST="${MELODEE_DATABASE_HOST:-${MELODEE_DB_HOST:-localhost}}"
DB_PORT="${MELODEE_DATABASE_PORT:-${MELODEE_DB_PORT:-5432}}"
DB_USER="${MELODEE_DATABASE_USER:-${MELODEE_DB_USER:-melodee_user}}"
DB_PASS="${MELODEE_DATABASE_PASSWORD:-${MELODEE_DB_PASSWORD:-admin123}}"
DB_NAME="${MELODEE_DATABASE_DBNAME:-${MELODEE_DB_NAME:-melodee}}"

# Generate bcrypt hash using the helper Go program
HASH=$(cd "$(dirname "$0")/cmd/bcrypt-hash" && GO111MODULE=on go run . "$PASSWORD")

export PGPASSWORD="$DB_PASS"

psql -v ON_ERROR_STOP=1 \
  -h "$DB_HOST" -p "$DB_PORT" \
  -U "$DB_USER" -d "$DB_NAME" <<SQL
DO \$\$
BEGIN
  IF EXISTS (SELECT 1 FROM users WHERE username = '$USERNAME') THEN
    RAISE EXCEPTION 'User % already exists', '$USERNAME';
  END IF;

  INSERT INTO users (username, email, password_hash, is_admin, created_at)
  VALUES ('$USERNAME', '$EMAIL', '$HASH', TRUE, NOW());
END
\$\$;
SQL

echo "Admin user '$USERNAME' created."
