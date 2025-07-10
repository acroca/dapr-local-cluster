FROM golang:1.24.4-alpine AS builder

WORKDIR /app

COPY durabletask-go /app/durabletask-go
COPY go-sdk /app/go-sdk

# Create the apps/multiapp directory structure
RUN mkdir -p /app/local-cluster/apps/multiapp
WORKDIR /app/local-cluster/apps/multiapp

COPY local-cluster/apps/multiapp/go.mod local-cluster/apps/multiapp/go.sum .
RUN go mod download

# Copy the multiapp source code
COPY local-cluster/apps/multiapp/ .

# Build the application
RUN go build -o multiapp main.go

FROM alpine:3.19.0
COPY --from=builder /app/local-cluster/apps/multiapp/multiapp /app/multiapp

# Set the default port
ENV APP_PORT=6007

# Expose the port
EXPOSE 6007

CMD ["/app/multiapp"]
