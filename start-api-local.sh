#!/bin/bash
cd /home/steven/source/melodee-next/src/api
export MELODEE_DATABASE_HOST=127.0.0.1
export MELODEE_DATABASE_PASSWORD=admin123
export MELODEE_DATABASE_USER=melodee_user
export MELODEE_DATABASE_DBNAME=melodee
export MELODEE_JWT_SECRET=my-local-dev-secret-key-12345
export GO111MODULE=on
go run main.go
