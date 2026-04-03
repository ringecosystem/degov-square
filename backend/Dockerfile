FROM golang:1.25.0-alpine AS builder
ARG APP_VERSION

WORKDIR /code

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download \
  && go get github.com/99designs/gqlgen@$(go list -m -f '{{.Version}}' github.com/99designs/gqlgen)

# Copy the build inputs without overwriting the bootstrapped module files.
COPY gqlgen.yml ./
COPY cmd ./cmd
COPY database ./database
COPY graph ./graph
COPY internal ./internal
COPY migrations ./migrations
COPY routes ./routes
COPY services ./services
COPY tasks ./tasks
COPY tests ./tests
COPY types ./types

# Build the application
RUN GOFLAGS=-mod=mod go generate ./... \
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
