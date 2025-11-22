#!/bin/bash
#
# Database Backup Automation Script for Melodee
# Creates automated database backups with rotation
#

set -euo pipefail

# Configuration
BACKUP_DIR="${MELODEE_BACKUP_DIR:-/backup/melodee}"
RETENTION_DAYS="${MELODEE_BACKUP_RETENTION_DAYS:-7}"
DB_HOST="${MELODEE_DATABASE_HOST:-localhost}"
DB_PORT="${MELODEE_DATABASE_PORT:-5432}"
DB_NAME="${MELODEE_DATABASE_DBNAME:-melodee}"
DB_USER="${MELODEE_DATABASE_USER:-melodee_user}"
DATE_FORMAT="${MELODEE_DATE_FORMAT:-%Y%m%d_%H%M%S}"
ENCRYPTION_KEY="${MELODEE_BACKUP_ENCRYPTION_KEY:-}"

# Logging
LOG_FILE="${MELODEE_LOG_DIR:-/var/log}/melodee-backup.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]$(date '+%Y-%m-%d %H:%M:%S')${NC} $1" | tee -a "$LOG_FILE"
}

log_warn() {
    echo -e "${YELLOW}[WARN]$(date '+%Y-%m-%d %H:%M:%S')${NC} $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]$(date '+%Y-%m-%d %H:%M:%S')${NC} $1" | tee -a "$LOG_FILE"
}

# Create backup directory if it doesn't exist
create_backup_dir() {
    if [ ! -d "$BACKUP_DIR" ]; then
        log_info "Creating backup directory: $BACKUP_DIR"
        mkdir -p "$BACKUP_DIR" || {
            log_error "Failed to create backup directory: $BACKUP_DIR"
            exit 1
        }
    fi
}

# Validate database connection
test_db_connection() {
    log_info "Testing database connection..."
    
    if ! pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME"; then
        log_error "Database connection test failed"
        exit 1
    fi
    
    log_info "Database connection test successful"
}

# Create encrypted backup with compression
create_backup() {
    local timestamp
    timestamp=$(date +"$DATE_FORMAT")
    local backup_file="$BACKUP_DIR/melodee_$timestamp.sql.gz"
    
    log_info "Starting backup to: $backup_file"
    
    # Perform the backup with pg_dump and compress with gzip
    if PGPASSWORD="$MELODEE_DATABASE_PASSWORD" pg_dump \
        --host="$DB_HOST" \
        --port="$DB_PORT" \
        --username="$DB_USER" \
        --dbname="$DB_NAME" \
        --verbose \
        --clean \
        --no-owner \
        --no-privileges \
        --format=custom \
        --jobs=4 \
        | gzip > "$backup_file"; then
        
        log_info "Backup completed successfully: $backup_file"
        
        # Verify backup integrity
        if verify_backup "$backup_file"; then
            log_info "Backup integrity verified"
        else
            log_error "Backup integrity check failed"
            rm "$backup_file"  # Remove corrupted backup
            return 1
        fi
        
        # Optional encryption if encryption key is provided
        if [ -n "$ENCRYPTION_KEY" ]; then
            encrypt_backup "$backup_file"
        fi
        
        return 0
    else
        log_error "Backup failed"
        return 1
    fi
}

# Verify backup integrity by checking if it can be extracted
verify_backup() {
    local backup_file="$1"
    
    log_info "Verifying backup integrity for: $backup_file"
    
    # For custom format dumps, use pg_restore to test
    if [ "${backup_file##*.}" = "gz" ]; then
        gunzip -c "$backup_file" | pg_restore --list > /dev/null 2>&1
    else
        pg_restore --list "$backup_file" > /dev/null 2>&1
    fi
    
    return $?
}

# Encrypt backup file using openssl
encrypt_backup() {
    local backup_file="$1"
    local encrypted_file="${backup_file}.enc"
    
    log_info "Encrypting backup: $encrypted_file"
    
    # Create encrypted backup using AES-256-CBC
    openssl enc -aes-256-cbc -salt -in "$backup_file" -out "$encrypted_file" -k "$ENCRYPTION_KEY"
    
    if [ $? -eq 0 ]; then
        # Remove the original unencrypted backup
        rm "$backup_file"
        log_info "Backup encrypted successfully: $encrypted_file"
    else
        log_error "Encryption failed for: $backup_file"
        return 1
    fi
}

# Decrypt backup file for restoration
decrypt_backup() {
    local encrypted_file="$1"
    local decrypted_file="${encrypted_file%.enc}"
    
    log_info "Decrypting backup: $encrypted_file"
    
    openssl enc -aes-256-cbc -d -in "$encrypted_file" -out "$decrypted_file" -k "$ENCRYPTION_KEY"
    
    if [ $? -eq 0 ]; then
        log_info "Backup decrypted successfully: $decrypted_file"
        echo "$decrypted_file"
    else
        log_error "Decryption failed"
        return 1
    fi
}

# Clean up old backups based on retention policy
cleanup_old_backups() {
    log_info "Cleaning up backups older than $RETENTION_DAYS days"
    
    # Find and delete old backup files
    find "$BACKUP_DIR" -type f -name "melodee_*.sql.gz*" -mtime +$RETENTION_DAYS -exec rm {} \;
    
    log_info "Cleanup completed"
}

# Restore from backup
restore_from_backup() {
    local backup_file="$1"
    
    if [ ! -f "$backup_file" ]; then
        log_error "Backup file not found: $backup_file"
        exit 1
    fi
    
    log_info "Restoring from backup: $backup_file"
    
    # If file is encrypted, decrypt first
    if [[ "$backup_file" == *.enc ]]; then
        backup_file=$(decrypt_backup "$backup_file")
        if [ $? -ne 0 ]; then
            log_error "Failed to decrypt backup file"
            exit 1
        fi
    fi
    
    # If file is compressed, decompress first
    if [[ "$backup_file" == *.gz ]]; then
        log_info "Restoring compressed backup..."
        gunzip -c "$backup_file" | PGPASSWORD="$MELODEE_DATABASE_PASSWORD" pg_restore \
            --clean \
            --if-exists \
            --host="$DB_HOST" \
            --port="$DB_PORT" \
            --username="$DB_USER" \
            --dbname="$DB_NAME" \
            --verbose
    else
        log_info "Restoring uncompressed backup..."
        PGPASSWORD="$MELODEE_DATABASE_PASSWORD" pg_restore \
            --clean \
            --if-exists \
            --host="$DB_HOST" \
            --port="$DB_PORT" \
            --username="$DB_USER" \
            --dbname="$DB_NAME" \
            --verbose \
            "$backup_file"
    fi
    
    if [ $? -eq 0 ]; then
        log_info "Restore completed successfully"
    else
        log_error "Restore failed"
        exit 1
    fi
}

# Generate backup report
generate_report() {
    local report_file="$BACKUP_DIR/backup_report_$(date +%Y%m%d).txt"
    
    {
        echo "Melodee Database Backup Report - $(date)"
        echo "====================================="
        echo
        echo "Configuration:"
        echo "  Backup Directory: $BACKUP_DIR"
        echo "  Database: $DB_NAME@$DB_HOST:$DB_PORT"
        echo "  Retention Days: $RETENTION_DAYS"
        echo
        echo "Backup Statistics:"
        echo "  Total Backups: $(find "$BACKUP_DIR" -name "melodee_*.sql.gz*" -type f | wc -l)"
        echo "  Total Size: $(du -sh "$BACKUP_DIR" | cut -f1)"
        echo
        echo "Recent Backups:"
        ls -lh "$BACKUP_DIR" | grep "melodee_" | head -10
    } > "$report_file"
    
    log_info "Backup report generated: $report_file"
}

# Main execution
main() {
    local action="${1:-backup}"
    
    case "$action" in
        backup)
            log_info "Starting automated backup process"
            create_backup_dir
            test_db_connection
            if create_backup; then
                cleanup_old_backups
                generate_report
                log_info "Backup process completed successfully"
            else
                log_error "Backup process failed"
                exit 1
            fi
            ;;
        restore)
            local backup_to_restore="${2?Backup file path required for restore}"
            create_backup_dir
            test_db_connection
            restore_from_backup "$backup_to_restore"
            ;;
        verify)
            local backup_to_verify="${2?Backup file path required for verification}"
            verify_backup "$backup_to_verify"
            ;;
        clean)
            create_backup_dir
            cleanup_old_backups
            ;;
        report)
            create_backup_dir
            generate_report
            ;;
        *)
            echo "Usage: $0 {backup|restore|verify|clean|report} [backup_file]"
            echo "  backup (default) - Create a new backup"
            echo "  restore <file>   - Restore from specified backup file"
            echo "  verify <file>    - Verify integrity of specified backup file"
            echo "  clean           - Remove old backup files based on retention policy"
            echo "  report          - Generate backup statistics report"
            exit 1
            ;;
    esac
}

main "$@"