
FROM golang:1.26.10-alpine AS build

WORKDIR /src

COPY backend/go.mod backend/go.sum* ./

RUN go mod download

COPY backend/ ./

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/server ./cmd

FROM alpine:3.21

RUN apk add --no-cache ca-certificates \
    && adduser -D -H -u 10001 appuser

WORKDIR /app

COPY --from=build --chown=appuser:appuser /out/server ./server

ENV HTTP_ADDR=:8080

USER appuser

EXPOSE 8080

ENTRYPOINT ["./server"]
