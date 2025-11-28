#!/bin/bash
# Database initialization script for Melodee
# Runs once when the database container starts for the first time

set -e

echo "Initializing database schema..."

# Set application DB user password (defaults to admin123 to match config/.env)
APP_DB_PASSWORD="${MELODEE_DATABASE_PASSWORD:-admin123}"

# Create extensions
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- Create required extensions
    CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
    CREATE EXTENSION IF NOT EXISTS "pg_trgm";
    CREATE EXTENSION IF NOT EXISTS "btree_gin";

    -- Create roles if they don't exist
    DO \$\$
    BEGIN
       IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'melodee_user') THEN
          CREATE ROLE melodee_user LOGIN PASSWORD '${APP_DB_PASSWORD}';
       ELSE
          ALTER ROLE melodee_user PASSWORD '${APP_DB_PASSWORD}';
       END IF;
    END
    \$\$;

    -- Grant database privileges
    GRANT ALL PRIVILEGES ON DATABASE melodee TO melodee_user;
EOSQL

# Run schema creation (executed in order by filename)
echo "Creating tables from schema files..."

echo "Database initialization completed successfully."