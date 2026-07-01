# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS build

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/quicknotes .

FROM alpine:3.22

RUN addgroup -S quicknotes && adduser -S -G quicknotes quicknotes

WORKDIR /app

RUN mkdir -p /app/data && chown -R quicknotes:quicknotes /app

COPY --from=build /out/quicknotes /app/quicknotes
COPY --chown=quicknotes:quicknotes seed.json /app/seed.json

ENV ADDR=:8080
ENV DATA_PATH=/app/data/notes.json
ENV SEED_PATH=/app/seed.json

EXPOSE 8080

USER quicknotes

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/health >/dev/null || exit 1

ENTRYPOINT ["/app/quicknotes"]
