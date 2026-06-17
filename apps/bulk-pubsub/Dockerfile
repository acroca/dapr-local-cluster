FROM golang:1.26.0-alpine AS builder

WORKDIR /app

COPY . .
RUN go build -o bulk-pubsub main.go

FROM alpine:3.19.0
COPY --from=builder /app/bulk-pubsub /app/bulk-pubsub

# Set the default port
ENV APP_PORT=6006

# Expose the port
EXPOSE 6006

CMD ["/app/bulk-pubsub"]
