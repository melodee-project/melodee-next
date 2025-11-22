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

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o melodee .

# Final stage: Use scratch as a minimal base for smallest image
FROM scratch

# Copy certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/melodee .

# Expose port
EXPOSE 8080

# Command to run the executable
CMD ["./melodee"]