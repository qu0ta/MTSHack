FROM golang:1.24-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum* ./
RUN if [ -f go.mod ]; then go mod download; fi
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/api

FROM alpine:3.21
RUN adduser -D -H appuser
WORKDIR /app
COPY --from=build /out/server /app/server
USER appuser
EXPOSE 8080
ENTRYPOINT ["/app/server"]
