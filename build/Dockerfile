# Stage 1: Build
FROM golang:1.23.2 AS builder

WORKDIR /app

# Copy go.mod and go.sum first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy only necessary folders
COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg

# Build main binary
RUN go build -o /bin/server ./cmd

# Stage 2: Runtime
FROM golang:1.23.2

WORKDIR /app

# Copy only what's needed at runtime
COPY --from=builder /bin/server /usr/local/bin/
COPY --from=builder /app/cmd ./cmd
COPY --from=builder /app/internal ./internal
COPY --from=builder /app/pkg ./pkg

HEALTHCHECK --interval=30s --timeout=5s CMD curl --fail http://localhost:2000/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/server"]
CMD ["http"]