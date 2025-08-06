# Build stage
FROM golang:1.22.5-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o simple-kv .

# Final stage
FROM alpine:latest

# Install ca-certificates and netcat for health checks
RUN apk --no-cache add ca-certificates netcat-openbsd

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/simple-kv .

# Create a directory for data persistence (for future use)
RUN mkdir -p /data

# Expose port 8080 (default port)
EXPOSE 8080

# Command to run the application
# PORT and PEERS will be set via environment variables
CMD ["sh", "-c", "./simple-kv ${PORT:-8080} ${PEERS:-}"]
