module github.com/acroca/dapr-example-app

go 1.24.4

require github.com/dapr/go-sdk v1.12.0

replace github.com/dapr/go-sdk => ../../../go-sdk

replace github.com/dapr/durabletask-go => ../../../durabletask-go

require (
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/dapr/dapr v1.15.5 // indirect
	github.com/dapr/durabletask-go v0.7.2 // indirect
	github.com/dapr/kit v0.15.4 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel v1.36.0 // indirect
	go.opentelemetry.io/otel/metric v1.36.0 // indirect
	go.opentelemetry.io/otel/trace v1.36.0 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/grpc v1.73.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/utils v0.0.0-20250502105355-0f33e8f1c979 // indirect
)
