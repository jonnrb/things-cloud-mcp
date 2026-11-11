FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
COPY things-cloud-sdk/go.mod things-cloud-sdk/go.sum ./things-cloud-sdk/
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o things-mcp .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && mkdir -p /data \
    && chown 65532:65532 /data

COPY --from=builder /app/things-mcp /usr/local/bin/things-mcp

# Run as an unprivileged user; /data must be writable by uid 65532.
USER 65532:65532

EXPOSE 8080

ENV PORT=8080
ENV DATA_DIR=/data

VOLUME ["/data"]

CMD ["things-mcp"]
