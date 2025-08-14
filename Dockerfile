# Build stage
FROM golang:1.24.3-alpine3.22 AS builder

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o healthcheck ./cmd/healthcheck

# Final stage
FROM alpine:3.22.0

# Set working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/healthcheck .

# Create a non-root user and group 'executor'
RUN addgroup -S executor && adduser -S executor -G executor

# Change ownership of the app files to 'executor'
RUN chown -R executor:executor /app

# Switch to the 'executor' user
USER executor

# Run the application
CMD ["./healthcheck"] 