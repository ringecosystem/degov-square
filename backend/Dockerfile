FROM golang:1.25-rc-alpine AS builder
ARG APP_VERSION

WORKDIR /code

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go get github.com/99designs/gqlgen \
  && go generate ./... \
  && go build -ldflags="-X main.Version=${APP_VERSION}" -o bin/degov-server cmd/main.go

# Final stage
FROM alpine:3.22 AS runner

# Install ca-certificates for TLS
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /code/bin/degov-server .

# Copy migrations
COPY --from=builder /code/migrations ./migrations

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./degov-server"]
