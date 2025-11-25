#!/bin/bash
cd /home/steven/source/melodee-next/src/api || exit 1

export MELODEE_DATABASE_HOST=127.0.0.1
export MELODEE_DATABASE_PASSWORD=admin123  
export MELODEE_DATABASE_USER=melodee_user
export MELODEE_DATABASE_NAME=melodee
export MELODEE_JWT_SECRET=my-local-dev-secret-key-12345
export GO111MODULE=on

echo "Starting Melodee API..."
exec go run main.go
