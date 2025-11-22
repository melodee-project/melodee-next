#!/bin/bash
# Database initialization script for Melodee

# This script initializes the database with the required schemas and tables
# It will be run once when the database container starts for the first time

echo "Initializing database schema..."

# Example: Create extensions, users, and tables
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- Create required extensions
    CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
    CREATE EXTENSION IF NOT EXISTS "pg_trgm";
    CREATE EXTENSION IF NOT EXISTS "btree_gin";

    -- Create roles if they don't exist
    DO \$\$
    BEGIN
       IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'melodee_user') THEN
          CREATE ROLE melodee_user LOGIN PASSWORD '${MELODEE_DB_PASSWORD}';
       END IF;
    END
    \$\$;

    -- Grant privileges
    GRANT ALL PRIVILEGES ON DATABASE melodee TO melodee_user;
EOSQL

echo "Database initialization completed."