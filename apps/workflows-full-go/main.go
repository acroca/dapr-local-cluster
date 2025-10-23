package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dapr/durabletask-go/workflow"
	"github.com/dapr/go-sdk/client"
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
	id, err := wfClient.ScheduleWorkflow(context.Background(), "RootWorkflow", workflow.WithInput(workflowInput))
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

	respFetch, err := wfClient.WaitForWorkflowCompletion(ctx, id)
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

	if respFetch.RuntimeStatus != workflow.StatusCompleted {
		errorMessage := fmt.Sprintf("Workflow failed with status: %s. Error: %v", respFetch.RuntimeStatus.String(), respFetch.FailureDetails.ErrorMessage)
		log.Println(errorMessage)
		response := WorkflowResponse{
			Status:     "failed",
			InstanceID: id,
			Error:      errorMessage,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Printf("Workflow completed!")
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

	if err := r.AddWorkflow(RootWorkflow); err != nil {
		log.Fatalf("failed to add workflow: %v", err)
	}
	if err := r.AddWorkflow(ChildWorkflowAsyncActivities); err != nil {
		log.Fatalf("failed to add workflow: %v", err)
	}
	if err := r.AddWorkflow(ChildWorkflowNTimes); err != nil {
		log.Fatalf("failed to add workflow: %v", err)
	}
	if err := r.AddActivity(DoubleActivity); err != nil {
		log.Fatalf("failed to add activity: %v", err)
	}

	var err error
	wfClient, err = client.NewWorkflowClient()
	if err != nil {
		log.Fatalf("failed to create workflow client: %v", err)
	}

	if err := wfClient.StartWorker(context.Background(), r); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}

	// Setup HTTP routes
	http.HandleFunc("/healthz", healthHandler)
	http.HandleFunc("/start", startWorkflowHandler)

	// Get port from environment variable or use default
	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "6020"
	}

	// Start HTTP server
	log.Printf("Starting HTTP server on port %s", appPort)
	log.Fatal(http.ListenAndServe(":"+appPort, nil))
}

func RootWorkflow(ctx *workflow.WorkflowContext) (any, error) {

	// Test simple activity call from root workflow
	var number int
	err := ctx.CallActivity(DoubleActivity, workflow.WithActivityInput(4)).Await(&number)
	if err != nil {
		return nil, err
	}
	if number != 8 {
		return nil, fmt.Errorf("number is not 8, is %d", number)
	}

	// Test child workflow call from root workflow, in the same app
	err = ctx.CallChildWorkflow(ChildWorkflowAsyncActivities, workflow.WithChildWorkflowInput(4)).Await(&number)
	if err != nil {
		return nil, err
	}
	if number != 16 {
		return nil, fmt.Errorf("number is not 16, is %d", number)
	}

	// Test child workflow call from root workflow, in a different app
	err = ctx.CallChildWorkflow(ChildWorkflowAsyncActivities, workflow.WithChildWorkflowInput(5), workflow.WithChildWorkflowAppID("workflows-full-go-2")).Await(&number)
	if err != nil {
		return nil, err
	}
	if number != 20 {
		return nil, fmt.Errorf("number is not 20, is %d", number)
	}

	// A child workflow with ContinueAsNew, in the same app
	err = ctx.CallChildWorkflow(ChildWorkflowNTimes, workflow.WithChildWorkflowInput(&ChildWorkflow2xNTimesInput{N: 4, Times: 3})).Await(&number)
	if err != nil {
		return nil, err
	}
	if number != 32 {
		return nil, fmt.Errorf("number is not 32, is %d", number)
	}

	// A child workflow with ContinueAsNew, in a different app
	err = ctx.CallChildWorkflow(ChildWorkflowNTimes, workflow.WithChildWorkflowInput(&ChildWorkflow2xNTimesInput{N: 5, Times: 3}), workflow.WithChildWorkflowAppID("workflows-full-go-2")).Await(&number)
	if err != nil {
		return nil, err
	}
	if number != 40 {
		return nil, fmt.Errorf("number is not 40, is %d", number)
	}

	return nil, nil
}

// ChildWorkflowAsyncActivities calls DoubleActivity twice asynchronously, returning 4x the input
func ChildWorkflowAsyncActivities(ctx *workflow.WorkflowContext) (any, error) {
	var n int
	ctx.GetInput(&n)

	now := time.Now()
	a1 := ctx.CallActivity(DoubleActivity, workflow.WithActivityInput(n))
	a2 := ctx.CallActivity(DoubleActivity, workflow.WithActivityInput(n))

	var n1, n2 int
	err := a1.Await(&n1)
	if err != nil {
		return nil, err
	}
	err = a2.Await(&n2)
	if err != nil {
		return nil, err
	}
	if time.Since(now) >= 2*time.Second {
		return nil, fmt.Errorf("activities didn't run in parallel")
	}
	return n1 + n2, nil
}

type ChildWorkflow2xNTimesInput struct {
	N     int `json:"n"`
	Times int `json:"times"`
}

// ChildWorkflowNTimes calls DoubleActivity N times, returning 2^N * input. It's done using ContinueAsNew until Times is 1.
func ChildWorkflowNTimes(ctx *workflow.WorkflowContext) (any, error) {
	var input ChildWorkflow2xNTimesInput
	ctx.GetInput(&input)
	var number int
	err := ctx.CallActivity(DoubleActivity, workflow.WithActivityInput(input.N)).Await(&number)
	if err != nil {
		return nil, err
	}
	if input.Times > 1 {
		ctx.ContinueAsNew(&ChildWorkflow2xNTimesInput{
			N:     number,
			Times: input.Times - 1,
		})
	}
	return number, nil
}

func DoubleActivity(ctx workflow.ActivityContext) (any, error) {
	time.Sleep(1 * time.Second)
	var n int
	ctx.GetInput(&n)
	return n * 2, nil
}
