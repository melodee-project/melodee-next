# Use official Golang image to build the binary
FROM golang:1.25.1-alpine AS builder

# Install git (needed for go modules)
RUN apk add --no-cache git ca-certificates

# Set working directory inside the container
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code (entire repo so internal modules resolve)
COPY . .

# Build the API binary from its module path
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o melodee melodee/api

# Final stage: Use Alpine as a small base with ffmpeg available
FROM alpine:3.20

# Install ffmpeg and CA certificates
RUN apk add --no-cache ffmpeg ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/melodee .

# Expose port
EXPOSE 8080

# Command to run the executable
CMD ["./melodee"]