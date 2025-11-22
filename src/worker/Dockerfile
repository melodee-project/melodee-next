# Use official Golang image to build the binary
FROM golang:1.25.1-alpine AS builder

# Install git (needed for go modules) and common build tools
RUN apk add --no-cache git ca-certificates gcc musl-dev

# Set working directory inside the container
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary - specifically for worker functionality
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o melodee-worker .

# Final stage: Use scratch as a minimal base for smallest image
FROM alpine:latest

# Install common tools and FFmpeg for media processing
RUN apk add --no-cache ca-certificates ffmpeg tzdata

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/melodee-worker .

# Create necessary directories with proper permissions
RUN mkdir -p /storage /inbound /staging /quarantine /user_images && \
    chmod -R 777 /storage /inbound /staging /quarantine /user_images

# Command to run the executable as a worker
CMD ["./melodee-worker"]