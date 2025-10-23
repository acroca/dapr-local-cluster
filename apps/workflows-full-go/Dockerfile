FROM golang:1.24.6-alpine AS builder

WORKDIR /app

COPY . .
RUN go build -o app main.go

FROM alpine:3.19.0
COPY --from=builder /app/app /app/app

# Set the default port
ENV APP_PORT=6020

# Expose the port
EXPOSE 6020

CMD ["/app/app"]
