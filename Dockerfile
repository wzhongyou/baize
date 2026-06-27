# ── Build Stage ──────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /baize ./cli/

# ── Runtime Stage ────────────────────────────────────────────
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata git curl

COPY --from=builder /baize /usr/local/bin/baize

ENV BAIZE_DATA_DIR=/data
VOLUME /data

EXPOSE 9779

ENTRYPOINT ["baize"]
CMD ["server", "--host", "0.0.0.0", "--port", "9779"]
