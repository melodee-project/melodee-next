#!/bin/bash
# 
# Melodee Deployment Scripts
# Scripts for deploying Melodee to different environments
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
ENVIRONMENT="${1:-local}"
ACTION="${2:-deploy}"

# Load environment-specific configuration
CONFIG_FILE="$ROOT_DIR/config/deployment/$ENVIRONMENT.env"
if [ -f "$CONFIG_FILE" ]; then
    echo "Loading configuration from $CONFIG_FILE"
    export $(grep -v '^#' "$CONFIG_FILE" | xargs)
fi

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Validate prerequisites
validate_prerequisites() {
    log_info "Validating prerequisites..."
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi
    
    # Check Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose is not installed"
        exit 1
    fi
    
    # Check Git
    if ! command -v git &> /dev/null; then
        log_error "Git is not installed"
        exit 1
    fi
    
    log_info "Prerequisites validated"
}

# Deploy to environment
deploy() {
    local environment=$1
    
    case $environment in
        local|development)
            deploy_development
            ;;
        staging)
            deploy_staging
            ;;
        production)
            deploy_production
            ;;
        *)
            log_error "Unknown environment: $environment"
            exit 1
            ;;
    esac
}

# Development deployment
deploy_development() {
    log_info "Deploying to development environment..."
    
    # Set development-specific environment variables
    export COMPOSE_PROJECT_NAME="melodee-dev"
    
    # Bring down any existing containers
    docker-compose -f "$ROOT_DIR/docker-compose.yml" down --remove-orphans
    
    # Build and start services
    docker-compose -f "$ROOT_DIR/docker-compose.yml" up --build -d
    
    log_info "Development deployment completed"
    log_info "Services are starting, check status with: docker-compose -f $ROOT_DIR/docker-compose.yml ps"
}

# Staging deployment
deploy_staging() {
    log_info "Deploying to staging environment..."
    
    # Set staging-specific environment variables
    export COMPOSE_PROJECT_NAME="melodee-staging"
    
    # Pull latest images
    docker-compose -f "$ROOT_DIR/docker-compose.staging.yml" pull
    
    # Bring down existing containers
    docker-compose -f "$ROOT_DIR/docker-compose.staging.yml" down --remove-orphans
    
    # Start services
    docker-compose -f "$ROOT_DIR/docker-compose.staging.yml" up -d
    
    log_info "Staging deployment completed"
}

# Production deployment
deploy_production() {
    log_info "Deploying to production environment..."
    
    # Validate production deployment requirements
    if [ -z "${PRODUCTION_DEPLOY_ALLOWED:-}" ] || [ "$PRODUCTION_DEPLOY_ALLOWED" != "true" ]; then
        log_error "Production deployment is not allowed from this environment"
        log_error "Set PRODUCTION_DEPLOY_ALLOWED=true to enable production deployment"
        exit 1
    fi
    
    # Verify branch
    local current_branch
    current_branch=$(git rev-parse --abbrev-ref HEAD)
    if [ "$current_branch" != "main" ] && [[ ! "$current_branch" =~ ^release/ ]]; then
        log_error "Production deployment only allowed from main or release/* branches"
        log_error "Current branch: $current_branch"
        exit 1
    fi
    
    # Set production-specific environment variables
    export COMPOSE_PROJECT_NAME="melodee-prod"
    
    # Pull latest images
    docker-compose -f "$ROOT_DIR/docker-compose.prod.yml" pull
    
    # Create backup before deployment (optional)
    if [ -n "${CREATE_PRE_DEPLOY_BACKUP:-}" ] && [ "$CREATE_PRE_DEPLOY_BACKUP" = "true" ]; then
        log_info "Creating pre-deployment backup..."
        # Execute backup script
        sudo $ROOT_DIR/scripts/backup-db.sh
    fi
    
    # Bring down existing containers gracefully
    docker-compose -f "$ROOT_DIR/docker-compose.prod.yml" down --remove-orphans
    
    # Start services
    docker-compose -f "$ROOT_DIR/docker-compose.prod.yml" up -d
    
    # Run smoke tests
    run_smoke_tests || {
        log_error "Smoke tests failed, rolling back deployment..."
        rollback_production
        exit 1
    }
    
    # Run database migrations if needed
    if [ -n "${AUTO_MIGRATE_ON_DEPLOY:-}" ] && [ "$AUTO_MIGRATE_ON_DEPLOY" = "true" ]; then
        docker-compose -f "$ROOT_DIR/docker-compose.prod.yml" exec api go run migrate.go
    fi
    
    log_info "Production deployment completed"
}

# Rollback production deployment
rollback_production() {
    log_info "Rolling back production deployment..."
    
    # Implement rollback logic here
    # This might involve restoring from a backup or reverting to a previous image
    docker-compose -f "$ROOT_DIR/docker-compose.prod.yml" down
    log_warn "Manual intervention required to restore from backup or revert to previous version"
}

# Run smoke tests after deployment
run_smoke_tests() {
    log_info "Running smoke tests..."
    
    # Wait for services to start
    sleep 30
    
    # Test API health endpoint
    local api_response
    api_response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/healthz || echo "000")
    
    if [ "$api_response" = "200" ]; then
        log_info "API health check passed: $api_response"
    else
        log_error "API health check failed: $api_response"
        return 1
    fi
    
    # Test database connection through health endpoint
    if curl -sf http://localhost:8080/healthz | jq -e '.db.status' > /dev/null 2>&1; then
        local db_status
        db_status=$(curl -s http://localhost:8080/healthz | jq -r '.db.status')
        if [ "$db_status" = "ok" ]; then
            log_info "Database connection check passed"
        else
            log_error "Database connection check failed: $db_status"
            return 1
        fi
    else
        log_info "Database status check skipped (jq not available or API format changed)"
    fi
    
    # Test Redis connection through health endpoint
    if curl -sf http://localhost:8080/healthz | jq -e '.redis.status' > /dev/null 2>&1; then
        local redis_status
        redis_status=$(curl -s http://localhost:8080/healthz | jq -r '.redis.status')
        if [ "$redis_status" = "ok" ]; then
            log_info "Redis connection check passed"
        else
            log_error "Redis connection check failed: $redis_status"
            return 1
        fi
    else
        log_info "Redis status check skipped (jq not available or API format changed)"
    fi
    
    log_info "All smoke tests passed"
    return 0
}

# Health check function
health_check() {
    log_info "Checking service health..."
    
    local services_down=0
    local total_services=0
    
    # Get service status
    while IFS= read -r line; do
        if [[ $line =~ ^[a-z]+[a-zA-Z0-9_-]+[[:space:]]+[0-9a-f]+[[:space:]]+(Up|Exited).* ]]; then
            local service_name
            local service_status
            service_name=$(echo "$line" | awk '{print $1}')
            service_status=$(echo "$line" | awk '{print $3}')
            
            if [ "$service_status" = "Up" ]; then
                log_info "Service $service_name: UP"
            else
                log_warn "Service $service_name: DOWN"
                ((services_down++))
            fi
            ((total_services++))
        fi
    done <<< "$(docker-compose -f "$ROOT_DIR/docker-compose.yml" ps)"
    
    if [ $services_down -eq 0 ]; then
        log_info "All $total_services services are healthy"
    else
        log_error "$services_down out of $total_services services are down"
        return 1
    fi
}

# Status check
status() {
    docker-compose -f "$ROOT_DIR/docker-compose.yml" ps
}

# Stop services
stop() {
    case $ENVIRONMENT in
        local|development)
            docker-compose -f "$ROOT_DIR/docker-compose.yml" down
            ;;
        staging)
            docker-compose -f "$ROOT_DIR/docker-compose.staging.yml" down
            ;;
        production)
            log_warn "Stopping production services!"
            docker-compose -f "$ROOT_DIR/docker-compose.prod.yml" down
            ;;
        *)
            log_error "Unknown environment: $ENVIRONMENT"
            exit 1
            ;;
    esac
    
    log_info "Services stopped"
}

# Main execution
main() {
    validate_prerequisites
    
    case $ACTION in
        deploy)
            deploy "$ENVIRONMENT"
            ;;
        status)
            status
            ;;
        health)
            health_check
            ;;
        stop|down)
            stop
            ;;
        rollback)
            if [ "$ENVIRONMENT" = "production" ]; then
                rollback_production
            else
                log_error "Rollback only supported for production environment"
                exit 1
            fi
            ;;
        *)
            log_error "Unknown action: $ACTION"
            echo "Usage: $0 [environment] [action]"
            echo "Environments: local, development, staging, production"
            echo "Actions: deploy, status, health, stop, rollback"
            exit 1
            ;;
    esac
}

main