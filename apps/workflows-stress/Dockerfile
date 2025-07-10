FROM golang:1.24.4-alpine AS builder

WORKDIR /app

COPY . .
RUN go build -o workflows-stress main.go

FROM alpine:3.19.0
COPY --from=builder /app/workflows-stress /app/workflows-stress

# Set the default port
ENV APP_PORT=6006

# Expose the port
EXPOSE 6006

CMD ["/app/workflows-stress"]
