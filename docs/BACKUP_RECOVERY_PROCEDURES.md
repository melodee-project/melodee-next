# Backup and Disaster Recovery Procedures

## Database Backup

### Automated Backups
The system includes automated database backup capabilities that run daily and can be configured using the following approach:

```bash
# Daily backup script (add to crontab)
0 2 * * * /usr/local/bin/melodee-backup-db

# Weekly full backup with retention
0 3 * * 0 /usr/local/bin/melodee-backup-db --full --retention=30d

# Monthly archival backup
0 4 1 * * /usr/local/bin/melodee-backup-db --archive --retention=365d
```

### Backup Script
```bash
#!/bin/bash
# melodee-backup-db
# Database backup script for melodee application

BACKUP_DIR="/backup/melodee"
DATE=$(date +%Y%m%d_%H%M%S)
DB_NAME="melodee"
DB_USER="melodee_user"
DB_HOST="${MELODEE_DATABASE_HOST:-localhost}"
ARCHIVE_DAYS="${MELODEE_BACKUP_RETENTION_DAYS:-7}"
TEMP_DIR="/tmp"

# Create backup directory if it doesn't exist
mkdir -p $BACKUP_DIR

# Create backup filename
BACKUP_FILE="$BACKUP_DIR/melodee_$DATE.sql.gz"

# Perform backup with pg_dump
pg_dump -h $DB_HOST -U $DB_USER -d $DB_NAME | gzip > $BACKUP_FILE

# Verify backup integrity
if gunzip --test $BACKUP_FILE; then
    echo "Backup successful: $BACKUP_FILE"
    # Remove backups older than retention period
    find $BACKUP_DIR -name "melodee_*.sql.gz" -mtime +$ARCHIVE_DAYS -exec rm {} \;
else
    echo "Backup failed! Integrity check failed"
    rm $BACKUP_FILE
    exit 1
fi
```

## Media File Backup

### Backup Strategy
Media files should be backed up using a combination of:

1. **Incremental backups**: Using rsync to sync only changed files
2. **Snapshot-based backups**: Using LVM or filesystem snapshots for consistency
3. **Cloud replication**: Optional cloud backup for offsite storage

### Backup Script Example
```bash
#!/bin/bash
# melodee-backup-media
# Media library backup script

SOURCE_DIR="/melodee/storage"
BACKUP_DIR="/backup/melodee-media"
LOG_FILE="/var/log/melodee-backup-media.log"
DATE=$(date +%Y%m%d_%H%M%S)

# Create backup directory if it doesn't exist
mkdir -p $BACKUP_DIR

# Perform incremental backup using rsync
rsync -av --progress --partial --human-readable \
  --backup --backup-dir=$BACKUP_DIR/incremental_$DATE \
  --exclude='*.tmp' \
  --exclude='.DS_Store' \
  $SOURCE_DIR/ $BACKUP_DIR/current/

# Create a dated symlink for this backup
ln -sfn $BACKUP_DIR/current $BACKUP_DIR/latest_$DATE

# Cleanup old incremental backups
find $BACKUP_DIR -name "incremental_*" -mtime +7 -exec rm -rf {} \;

# Write log entry
echo "$(date): Media backup completed to $BACKUP_DIR" >> $LOG_FILE
```

## Disaster Recovery Procedures

### Data Recovery Steps
1. **Assessment**: Determine scope of data loss and affected components
2. **Database Recovery**:
   - Stop all services to prevent further data corruption
   - Restore latest database backup using `psql`
   - Validate data integrity

3. **Media Recovery**:
   - Restore media files from latest backup
   - Update file permissions and ownership
   - Optionally run consistency check to verify file integrity

4. **Verification**: Test basic functionality and data consistency
5. **Rollback Plan**: Have ready procedures to rollback if recovery fails

### Recovery Script
```bash
#!/bin/bash
# melodee-recover-db
# Database recovery script

RESTORE_DB_NAME="melodee"
RESTORE_DB_USER="melodee_user"
RESTORE_DB_HOST="${MELODEE_DATABASE_HOST:-localhost}"
BACKUP_FILE=$1

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup_file>"
    exit 1
fi

if [ ! -f "$BACKUP_FILE" ]; then
    echo "Backup file not found: $BACKUP_FILE"
    exit 1
fi

echo "Starting database recovery from: $BACKUP_FILE"
echo "WARNING: This will overwrite the current database!"

read -p "Continue? [y/N] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    exit 1
fi

# Create a safety backup of current database if possible
echo "Creating safety backup..."
pg_dump -h $RESTORE_DB_HOST -U $RESTORE_DB_USER -d $RESTORE_DB_NAME | gzip > /tmp/melodee_safety_backup_$(date +%Y%m%d_%H%M%S).sql.gz

# Stop services
sudo systemctl stop melodee-api
sudo systemctl stop melodee-worker

# Drop and recreate database
dropdb -h $RESTORE_DB_HOST -U $RESTORE_DB_USER $RESTORE_DB_NAME
createdb -h $RESTORE_DB_HOST -U $RESTORE_DB_USER $RESTORE_DB_NAME

# Restore database
gunzip -c $BACKUP_FILE | psql -h $RESTORE_DB_HOST -U $RESTORE_DB_USER -d $RESTORE_DB_NAME

# Start services
sudo systemctl start melodee-api
sudo systemctl start melodee-worker

echo "Database recovery completed."
```

## High Availability Configuration

### Database Replication
PostgreSQL streaming replication can be configured with:

1. **Primary server** configured with WAL archiving
2. **Standby server** receiving WAL logs
3. **Automatic failover** using Patroni or similar tools

### Application Scaling
Docker Compose configuration supports horizontal scaling:
```yaml
services:
  api:
    # ... existing config
    deploy:
      replicas: 3
      resources:
        limits:
          cpus: '0.5'
          memory: 512M
        reservations:
          cpus: '0.25'
          memory: 256M
  worker:
    # ... existing config
    deploy:
      replicas: 2  # Adjust based on workload
      resources:
        limits:
          cpus: '1.0'
          memory: 1G
        reservations:
          cpus: '0.5'
          memory: 512M
```

## Monitoring and Alerts

### Backup Verification
Automated backup verification using checksums:
```bash
# Verify backup integrity
cksum $BACKUP_FILE > $BACKUP_FILE.cksum

# Compare with stored checksum
diff $BACKUP_FILE.cksum $BACKUP_DIR/latest.cksum
```

### Alert Conditions
- Database backup failure
- Media storage filling up (>90%)
- Backup verification failure
- Service downtime exceeding threshold
- Data inconsistency detected