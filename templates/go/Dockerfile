FROM golang:1.24.4-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -o app main.go

# ------------------------------------------------------------
FROM alpine:3.19.0
COPY --from=builder /app/app /app/app

# Set the default port
ENV APP_PORT=$APP_PORT$

# Expose the port
EXPOSE $APP_PORT$

CMD ["/app/app"]
