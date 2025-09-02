FROM golang:1.24.4-alpine AS builder

WORKDIR /app

COPY . .
RUN go build -o main main.go

FROM alpine:3.19.0
COPY --from=builder /app/main /app/main

# Set the default port
ENV APP_PORT=6006

# Expose the port
EXPOSE 6006

CMD ["/app/main"]
