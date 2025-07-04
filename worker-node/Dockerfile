# Stage 1: Build
FROM golang:1.24.1 AS builder

WORKDIR /app

# Copy go.mod and go.sum first to leverage caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the source code
COPY . .

RUN CGO_ENABLED=1 go build -o main .

# Stage 2: Runtime
FROM debian:bookworm-slim

WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/main .

EXPOSE 8081
EXPOSE 6060

CMD ["./main"]