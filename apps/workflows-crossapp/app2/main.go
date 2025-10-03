package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/dapr/durabletask-go/workflow"
	"github.com/dapr/go-sdk/client"
	dapr "github.com/dapr/go-sdk/client"
)

var wclient *workflow.Client

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
	id, err := wclient.ScheduleWorkflow(context.Background(), "TestWorkflow2", workflow.WithInput(workflowInput))
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

	respFetch, err := wclient.WaitForWorkflowCompletion(ctx, id)
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

	if respFetch.RuntimeStatus.String() != "ORCHESTRATION_STATUS_COMPLETED" {
		log.Printf("Workflow failed! Status: %s", respFetch.RuntimeStatus.String())
		response := WorkflowResponse{
			Status:     "failed",
			InstanceID: id,
			Error:      fmt.Sprintf("Workflow failed with status: %s", respFetch.RuntimeStatus.String()),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Printf("Workflow completed! Result: %s", respFetch.Output.GetValue())
	response := WorkflowResponse{
		Status:     "completed",
		InstanceID: id,
		Result:     respFetch.Output.GetValue(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	r := workflow.NewRegistry()
	r.AddWorkflow(TestWorkflow2)
	r.AddActivity(TestActivity2)

	var err error
	wclient, err = client.NewWorkflowClient()
	if err != nil {
		log.Fatalf("failed to create workflow client: %v", err)
	}
	wclient.StartWorker(context.Background(), r)

	// Create Dapr client
	client, err := dapr.NewClient()
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// Setup HTTP routes
	http.HandleFunc("/healthz", healthHandler)
	http.HandleFunc("/start", startWorkflowHandler)

	// Get port from environment variable or use default
	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "6009"
	}

	// Start HTTP server
	log.Printf("Starting HTTP server on port %s", appPort)
	log.Fatal(http.ListenAndServe(":"+appPort, nil))
}

func TestWorkflow2(ctx *workflow.WorkflowContext) (any, error) {
	fmt.Println("TestWorkflow2 called")
	var number int
	// err := ctx.CallActivity(TestActivity2).Await(&number)
	err := ctx.CallActivity("random_number_generator", workflow.WithActivityAppID("workflows-crossapp3"), workflow.WithActivityRetryPolicy(&workflow.RetryPolicy{
		MaxAttempts:          3,
		InitialRetryInterval: 100 * time.Millisecond,
		BackoffCoefficient:   2,
		MaxRetryInterval:     1 * time.Second,
	})).Await(&number)
	if err != nil {
		return nil, err
	}
	return "Workflow completed with number: " + strconv.Itoa(number), nil
}

func TestActivity2(ctx workflow.ActivityContext) (any, error) {
	fmt.Println("TestActivity2 called")
	return rand.Intn(100000), nil
}
