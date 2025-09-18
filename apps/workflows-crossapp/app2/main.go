package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/dapr/go-sdk/workflow"
)

var wfClient *workflow.Client

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
	// Create and start workflow worker
	w, err := workflow.NewWorker()
	if err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}

	if err := w.RegisterWorkflow(TestWorkflow2); err != nil {
		log.Fatal(err)
	}
	if err := w.RegisterActivity(TestActivity2); err != nil {
		log.Fatal(err)
	}

	if err := w.Start(); err != nil {
		log.Fatal(err)
	}

	// Create Dapr client
	client, err := dapr.NewClient()
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// Create workflow client
	wfClient, err = workflow.NewClient(workflow.WithDaprClient(client))
	if err != nil {
		log.Fatalf("failed to initialise workflow client: %v", err)
	}

	// Setup HTTP routes
	http.HandleFunc("/healthz", healthHandler)

	// Get port from environment variable or use default
	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "6006"
	}

	// Start HTTP server
	log.Printf("Starting HTTP server on port %s", appPort)
	log.Fatal(http.ListenAndServe(":"+appPort, nil))
}

func TestWorkflow2(ctx *workflow.WorkflowContext) (any, error) {
	fmt.Println("TestWorkflow2 called")
	var number int
	err := ctx.CallActivity(TestActivity2).Await(&number)
	if err != nil {
		return nil, err
	}
	return "Workflow completed with number: " + strconv.Itoa(number), nil
}

func TestActivity2(ctx workflow.ActivityContext) (any, error) {
	fmt.Println("TestActivity2 called")
	return rand.Intn(100000), nil
}
