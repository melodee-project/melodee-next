#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 3 ]; then
  echo "Usage: $0 <username> <email> <password>" >&2
  exit 1
fi

USERNAME="$1"
EMAIL="$2"
PASSWORD="$3"

: "${MELODEE_DB_HOST:=localhost}"
: "${MELODEE_DB_PORT:=5432}"
: "${MELODEE_DB_USER:=melodee_user}"
: "${MELODEE_DB_PASSWORD:=admin123}"
: "${MELODEE_DB_NAME:=melodee}"

# Generate bcrypt hash using the helper Go program
HASH=$(cd "$(dirname "$0")/cmd/bcrypt-hash" && GO111MODULE=on go run . "$PASSWORD")

export PGPASSWORD="$MELODEE_DB_PASSWORD"

psql -v ON_ERROR_STOP=1 \
  -h "$MELODEE_DB_HOST" -p "$MELODEE_DB_PORT" \
  -U "$MELODEE_DB_USER" -d "$MELODEE_DB_NAME" <<SQL
DO \$\$
BEGIN
  IF EXISTS (SELECT 1 FROM melodee_users WHERE username = '$USERNAME') THEN
    RAISE EXCEPTION 'User % already exists', '$USERNAME';
  END IF;

  INSERT INTO melodee_users (username, email, password_hash, is_admin, created_at)
  VALUES ('$USERNAME', '$EMAIL', '$HASH', TRUE, NOW());
END
\$\$;
SQL

echo "Admin user '$USERNAME' created."
