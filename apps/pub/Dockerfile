FROM golang:1.24.2-alpine AS builder

WORKDIR /app

COPY . .
RUN go build -o pub main.go

FROM alpine:3.19.0
COPY --from=builder /app/pub /app/pub

CMD ["/app/pub"]
