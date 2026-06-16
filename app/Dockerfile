# syntax=docker/dockerfile:1

# ---------- builder stage ----------
FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags='-s -w' -o /quicknotes .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags='-s -w' -o /healthcheck ./cmd/healthcheck

# ---------- runtime stage ----------
FROM gcr.io/distroless/static:nonroot

WORKDIR /

COPY --from=builder /quicknotes /quicknotes
COPY --from=builder /healthcheck /healthcheck
COPY seed.json /seed.json
COPY --chown=nonroot:nonroot data/ /data/

EXPOSE 8080

USER nonroot:nonroot
ENTRYPOINT ["/quicknotes"]
