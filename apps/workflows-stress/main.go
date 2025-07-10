package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/dapr/go-sdk/workflow"
)

var count atomic.Uint64

func main() {
	// Create and start workflow worker
	w, err := workflow.NewWorker()
	if err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}

	if err := w.RegisterWorkflow(TestWorkflow); err != nil {
		log.Fatal(err)
	}
	if err := w.RegisterActivity(TestActivity); err != nil {
		log.Fatal(err)
	}

	if err := w.Start(); err != nil {
		log.Fatal(err)
	}

	log.Printf("Starting continuous workflow execution...")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	concurrentWorkflowRuns := 100
	workers := 200

	wg := sync.WaitGroup{}
	wg.Add(workers)

	sem := make(chan struct{}, concurrentWorkflowRuns)
	for range workers {
		go func() {
			defer wg.Done()
			createWorkflowWorker(ctx, sem)
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(1 * time.Second)
		prevCount := count.Load()
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				currCount := count.Load()
				log.Printf("Workflows completed: %d (about %d/s)", currCount, currCount-prevCount)
				prevCount = currCount
			}
		}
	}()

	wg.Wait()
}

func createWorkflowWorker(ctx context.Context, sem chan struct{}) error {
	client, err := dapr.NewClient()
	if err != nil {
		return err
	}
	defer client.Close()

	wfClient, err := workflow.NewClient(workflow.WithDaprClient(client))
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case sem <- struct{}{}:
			if err := RunWorkflow(wfClient); err != nil {
				log.Printf("Error running workflow: %v", err)
			}
			<-sem
		}
	}
}

func RunWorkflow(wfClient *workflow.Client) error {
	// Use current timestamp as workflow input
	workflowInput := time.Now().Format(time.RFC3339)

	// Start workflow
	id, err := wfClient.ScheduleNewWorkflow(context.Background(), "TestWorkflow", workflow.WithInput(workflowInput))
	if err != nil {
		return err
	}

	_, err = wfClient.WaitForWorkflowCompletion(context.Background(), id)
	if err != nil {
		return err
	}

	// Fetch workflow result
	respFetch, err := wfClient.FetchWorkflowMetadata(context.Background(), id, workflow.WithFetchPayloads(true))
	if err != nil {
		return err
	}
	if respFetch.RuntimeStatus.String() != "COMPLETED" {
		return fmt.Errorf("workflow %s failed! Status: %s", id, respFetch.RuntimeStatus.String())
	}
	count.Add(1)
	return nil
}

func TestWorkflow(ctx *workflow.WorkflowContext) (any, error) {
	var number int
	err := ctx.CallActivity(TestActivity).Await(&number)
	if err != nil {
		return nil, err
	}
	return "Workflow completed with number: " + strconv.Itoa(number), nil
}

func TestActivity(ctx workflow.ActivityContext) (any, error) {
	return rand.Intn(100000), nil
}
