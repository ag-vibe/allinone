FROM golang:1.25-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server ./cmd/main.go

FROM gcr.io/distroless/static-debian12

COPY --from=builder /app/server /server

ENTRYPOINT ["/server"]
