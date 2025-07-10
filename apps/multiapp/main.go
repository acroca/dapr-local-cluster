package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
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

func startWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req WorkflowRequest
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Error parsing request body: %v", err)
		}
	}

	// Use current timestamp as default input if none provided
	workflowInput := req.Input
	if workflowInput == "" {
		workflowInput = time.Now().Format(time.RFC3339)
	}

	log.Printf("Starting workflow with input: %s", workflowInput)

	// Start workflow
	id, err := wfClient.ScheduleNewWorkflow(context.Background(), "TestWorkflow",
		workflow.WithInput(workflowInput),
	)
	if err != nil {
		log.Printf("Error starting workflow: %v", err)
		response := WorkflowResponse{
			Status: "failed",
			Error:  fmt.Sprintf("Failed to start workflow: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Printf("Workflow started with instance ID: %s", id)

	// Wait for workflow completion with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	metadata, err := wfClient.WaitForWorkflowCompletion(ctx, id)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("Workflow timed out!")
			response := WorkflowResponse{
				Status:     "timeout",
				InstanceID: id,
				Message:    "Workflow execution timed out",
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestTimeout)
			json.NewEncoder(w).Encode(response)
			return
		}
		log.Printf("Error waiting for workflow completion: %v", err)
		response := WorkflowResponse{
			Status:     "failed",
			InstanceID: id,
			Error:      fmt.Sprintf("Failed to wait for workflow completion: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Printf("Workflow completed! Result: %s", string(metadata.SerializedOutput))
	response := WorkflowResponse{
		Status:     "completed",
		InstanceID: id,
		Result:     string(metadata.SerializedOutput),
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

	if err := w.RegisterWorkflow(TestWorkflow); err != nil {
		log.Fatal(err)
	}
	if err := w.RegisterWorkflow(TestChildWorkflow); err != nil {
		log.Fatal(err)
	}
	if err := w.RegisterActivity(TestActivity); err != nil {
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
	http.HandleFunc("/start", startWorkflowHandler)

	// Get port from environment variable or use default
	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "6006"
	}

	// Start HTTP server
	log.Printf("Starting HTTP server on port %s", appPort)
	log.Fatal(http.ListenAndServe(":"+appPort, nil))
}

func TestWorkflow(ctx *workflow.WorkflowContext) (any, error) {
	log.Printf("TestWorkflow called")
	var number int
	err := ctx.CallChildWorkflow(TestChildWorkflow, workflow.ChildWorkflowAppID("multiapp-3")).Await(&number)
	if err != nil {
		return nil, err
	}
	return number, nil
}

func TestChildWorkflow(ctx *workflow.WorkflowContext) (any, error) {
	log.Printf("TestChildWorkflow called")
	var number int
	err := ctx.CallActivity(TestActivity, workflow.ActivityAppID("multiapp-2")).Await(&number)
	if err != nil {
		return nil, err
	}
	return number, nil
}

func TestActivity(ctx workflow.ActivityContext) (any, error) {
	log.Printf("TestActivity called")
	return rand.Intn(100000), nil
}
