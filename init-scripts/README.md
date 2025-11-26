# Database Management

## Local Development

The database schema is managed via SQL files in `init-scripts/` instead of migrations.

### Reset Database (Podman/Docker)

To apply schema changes or reset your local database:

```bash
# Stop and remove containers
podman-compose down -v

# Start fresh with new schema
podman-compose up -d
```

The `init-scripts/` directory contains:
- `init_db.sh` - Creates extensions and roles
- `001_schema.sql` - Main database schema with all tables

### Schema Changes

1. Edit `init-scripts/001_schema.sql`
2. Down and up the containers: `podman-compose down -v && podman-compose up -d`
3. That's it - no migrations to manage!

### Manual Database Access

```bash
# Connect to the running database
podman exec -it melodee-db psql -U melodee_user -d melodee

# Or from host (if port 5432 is exposed)
psql -h localhost -U melodee_user -d melodee
```

## Production Deployment

For production, you'll want to:
1. Use proper database backups before schema changes
2. Consider a migration system (sql-migrate, golang-migrate)
3. Never use `down -v` (it deletes all data!)

For now, local development keeps it simple with container resets.
