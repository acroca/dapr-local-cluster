package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	rtv1 "github.com/dapr/dapr/pkg/proto/runtime/v1"
	dapr "github.com/dapr/go-sdk/client"
)

type WorkflowRequest struct {
	Input string `json:"input,omitempty"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

type WorkflowResponse struct {
	Status     string `json:"status"`
	InstanceID string `json:"instance_id"`
	Result     string `json:"result,omitempty"`
	Message    string `json:"message,omitempty"`
	Error      string `json:"error,omitempty"`
}

type Envelope struct {
	Entries []struct {
		Event struct {
			Data int `json:"data,omitempty"`
		} `json:"event,omitempty"`
	} `json:"entries,omitempty"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	client, err := dapr.NewClient()
	if err != nil {
		panic(err)
	}
	defer client.Close()

	grpc := client.GrpcClient()
	stream, err := grpc.SubscribeTopicEventsAlpha1(context.Background())
	if err != nil {
		log.Fatalf("failed to subscribe to topic: %v", err)
	}

	err = stream.Send(&rtv1.SubscribeTopicEventsRequestAlpha1{
		SubscribeTopicEventsRequestType: &rtv1.SubscribeTopicEventsRequestAlpha1_InitialRequest{
			InitialRequest: &rtv1.SubscribeTopicEventsRequestInitialAlpha1{
				PubsubName: "test", Topic: "test",
			},
		},
	})

	// go func() {
	// 	for {
	// 		event, err := stream.Recv()
	// 		if err != nil {
	// 			log.Printf("error receiving event: %v", err)
	// 			return
	// 		}
	// 		log.Printf("Received event: %s", string(event.GetEventMessage().GetData()))
	// 	}
	// }()

	// Setup HTTP routes
	http.HandleFunc("/healthz", healthHandler)
	http.HandleFunc("/batch", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		// all, err := io.ReadAll(r.Body)
		// if err != nil {
		// 	log.Printf("Error reading request body: %v", err)
		// 	http.Error(w, "Internal server error", http.StatusInternalServerError)
		// 	return
		// }
		// log.Printf("Received batch event with data: %s", string(all))

		data := Envelope{}
		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			log.Printf("Error reading request body: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		ids := make([]string, len(data.Entries))
		for i, entry := range data.Entries {
			ids[i] = fmt.Sprintf("%d", entry.Event.Data)
		}
		log.Printf("Received batch event with IDs: %s", ids)
		w.WriteHeader(http.StatusOK)
	})
	http.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		publishBatch(client, 8)
		time.Sleep(12 * time.Second)
		publishBatch(client, 10)
		publishBatch(client, 8)
	})

	// Get port from environment variable or use default
	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "6006"
	}

	// Start HTTP server
	log.Printf("Starting HTTP server on port %s", appPort)
	log.Fatal(http.ListenAndServe(":"+appPort, nil))
}

var publishedCount = 0

func publishBatch(client dapr.Client, size int) {
	log.Printf("Published batch event with IDs: %d-%d", publishedCount+1, publishedCount+size)
	batch := make([]any, 0, size)
	for i := publishedCount + 1; i <= publishedCount+size; i++ {
		batch = append(batch, i)
	}
	publishedCount += size
	if resp := client.PublishEvents(context.Background(), "pubsub", "test", batch); resp.Error != nil {
		log.Printf("Error publishing event: %v", resp.Error)
	}
}
