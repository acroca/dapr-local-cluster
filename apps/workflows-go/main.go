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
	dapr "github.com/dapr/go-sdk/client"
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
	id, err := wfClient.ScheduleWorkflow(context.Background(), "TestWorkflow", workflow.WithInput(workflowInput))
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

	_, err = wfClient.WaitForWorkflowCompletion(ctx, id)
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

	// Fetch workflow result
	respFetch, err := wfClient.FetchWorkflowMetadata(context.Background(), id, workflow.WithFetchPayloads(true))
	if err != nil {
		log.Printf("Error fetching workflow metadata: %v", err)
		response := WorkflowResponse{
			Status:     "failed",
			InstanceID: id,
			Error:      fmt.Sprintf("Failed to get workflow result: %v", err),
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

	log.Printf("Workflow completed! Result: %s", string(respFetch.Output.GetValue()))
	response := WorkflowResponse{
		Status:     "completed",
		InstanceID: id,
		Result:     respFetch.Output.GetValue(),
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
	// Create and start workflow worker
	wfClient = workflow.NewClient(client.GrpcClientConn())

	reg := workflow.NewRegistry()

	if err := reg.AddWorkflow(TestWorkflow); err != nil {
		log.Fatal(err)
	}
	if err := reg.AddActivity(TestActivity); err != nil {
		log.Fatal(err)
	}
	if err := wfClient.StartWorker(context.Background(), reg); err != nil {
		log.Fatal(err)
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
	var number int
	sum := 0
	a1 := ctx.CallActivity(TestActivity)
	a2 := ctx.CallActivity(TestActivity)
	err := a1.Await(&number)
	if err != nil {
		return nil, err
	}
	sum += number

	err = a2.Await(&number)
	if err != nil {
		return nil, err
	}
	sum += number

	ctx.WaitForExternalEvent("foo", -1).Await(nil)

	return "Workflow completed with sum: " + strconv.Itoa(sum), nil
}

func TestActivity(ctx workflow.ActivityContext) (any, error) {
	time.Sleep(1 * time.Second)
	return rand.Intn(100000), nil
}
