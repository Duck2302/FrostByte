# Stage 1: Build
FROM golang:1.24.1 AS builder

WORKDIR /app

# Copy go.mod and go.sum first to leverage caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 go build -o main .

# Stage 2: Runtime
FROM debian:bookworm-slim

# Install curl for health checks
RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/main .

# Copy static files
COPY --from=builder /app/static ./static

# Expose the application port
EXPOSE 8080
EXPOSE 6060

# Command to run the application
CMD ["./main"]