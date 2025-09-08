# Stage 1: Build
FROM golang:1.23.2 AS builder

WORKDIR /app

# Copy go.mod and go.sum first for caching
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build go mod download
RUN go mod verify

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

# Copy only necessary folders
COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg

# Build main binary
RUN --mount=type=cache,target="/root/.cache/go-build" go build -o ./server ./cmd

# Stage 2: Runtime
FROM alpine AS runtime

WORKDIR /app

RUN apk add --no-cache curl

# Copy only what's needed at runtime
COPY --from=builder /app/server /usr/local/bin/server
COPY --from=builder /app/pkg/database/migrations ./pkg/database/migrations
COPY --from=builder /app/pkg/config/dev.yaml ./pkg/config/dev.yaml


RUN chmod +x /usr/local/bin/server
ENV PATH="/usr/local/bin/server:${PATH}"

HEALTHCHECK --interval=10s --timeout=5s CMD curl --fail http://localhost:2000/healthz || exit 1

ENTRYPOINT ["server"]
CMD ["http"]