#!/bin/bash
cd /home/steven/source/melodee-next/src || exit 1

export MELODEE_DATABASE_HOST=127.0.0.1
export MELODEE_DATABASE_PASSWORD=admin123
export MELODEE_DATABASE_USER=melodee_user
export MELODEE_DATABASE_DBNAME=melodee
export MELODEE_JWT_SECRET=my-local-dev-secret-key-12345
export MELODEE_SERVER_TLS_ENABLED=true
export MELODEE_SERVER_TLS_CERT_FILE="/home/steven/source/melodee-next/certs/localhost+3.pem"
export MELODEE_SERVER_TLS_KEY_FILE="/home/steven/source/melodee-next/certs/localhost+3-key.pem"
export GO111MODULE=on

echo "Starting Melodee API with TLS..."
echo "Building from source..."
go build -o melodee main.go || exit 1
echo "Starting server with TLS on https://0.0.0.0:8080..."
exec ./melodee
