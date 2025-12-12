# Stage 1: Build
FROM golang:1.25 AS builder

WORKDIR /app

# Copy go.mod and go.sum first for caching
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build go mod download
RUN go mod verify

# Use build args to detect target platform
ARG TARGETOS=linux
ARG TARGETARCH=amd64

ENV CGO_ENABLED=0
ENV GOOS=$TARGETOS
ENV GOARCH=$TARGETARCH

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

RUN chmod +x /usr/local/bin/server
ENV PATH="/usr/local/bin/server:${PATH}"

HEALTHCHECK --interval=10s --timeout=5s CMD curl --fail http://localhost:2000/healthz || exit 1

ENTRYPOINT ["server"]
CMD ["http"]