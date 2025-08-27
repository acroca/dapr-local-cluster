package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/gorilla/mux"
)

var actorType = "testActorType"
var numReminders = 1000

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func registerActorReminders(client dapr.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// fraction := time.Second / time.Duration(numReminders)
		wg := sync.WaitGroup{}
		wg.Add(numReminders)
		for i := range numReminders {
			go func(i int) {
				defer wg.Done()
				err := client.RegisterActorReminder(context.Background(), &dapr.RegisterActorReminderRequest{
					ActorType: actorType,
					ActorID:   fmt.Sprintf("my-actor-id-%d", i),
					Name:      fmt.Sprintf("my-reminder-%d", i),
					DueTime:   "1s",
					Period:    "1s",
				})
				if err != nil {
					log.Printf("Error registering reminder: %s", err.Error())
				}
			}(i)
		}
		wg.Wait()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Reminders registered"))
	}
}

func unregisterActorReminder(client dapr.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wg := sync.WaitGroup{}
		for i := 0; i < numReminders; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				err := client.UnregisterActorReminder(context.Background(), &dapr.UnregisterActorReminderRequest{
					ActorType: actorType,
					ActorID:   fmt.Sprintf("my-actor-id-%d", i),
					Name:      fmt.Sprintf("my-reminder-%d", i),
				})
				if err != nil {
					log.Printf("Error unregistering reminder: %s", err.Error())
				}
			}(i)
		}
		wg.Wait()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Reminders unregistered"))
	}
}

type Counts struct {
	Calls map[string]int
	mu    sync.Mutex
}

func NewCounts() *Counts {
	return &Counts{
		Calls: make(map[string]int),
		mu:    sync.Mutex{},
	}
}
func (c *Counts) clone() *Counts {
	c.mu.Lock()
	defer c.mu.Unlock()
	clone := NewCounts()
	for k, v := range c.Calls {
		clone.Calls[k] = v
	}
	return clone
}

func (c *Counts) actorMethodHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	// actorType := vars["actorType"]
	actorID := vars["id"]
	// reminderOrTimer := vars["reminderOrTimer"]
	// method := vars["method"]
	c.mu.Lock()
	c.Calls[actorID]++
	// c.Calls[fmt.Sprintf("%s/%s/%s/%s", actorType, actorID, reminderOrTimer, method)]++
	c.mu.Unlock()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Actor method called"))
}

func configHandler(w http.ResponseWriter, r *http.Request) {
	response := []byte(`{
		"entities": ["` + actorType + `"]
	}`)
	log.Printf("Processing dapr request for %s, responding with %#v\n", r.URL.RequestURI(), response)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func shutdownSidecarHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Processing %s test request for %s", r.Method, r.URL.RequestURI())

	shutdownURL := fmt.Sprintf("http://localhost:3500/v1.0/shutdown")
	_, err := http.Post(shutdownURL, "application/json", nil)
	if err != nil {
		log.Printf("Could not shutdown sidecar: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Sidecar shutdown"))
}

func printStats(counts *Counts) {
	sums := map[int]int{}
	counts.mu.Lock()
	for i := range numReminders {
		key := fmt.Sprintf("my-actor-id-%d", i)
		count, ok := counts.Calls[key]
		if !ok {
			count = 0
		}
		sums[count]++
	}
	counts.mu.Unlock()
	log.Printf("Summarized stats: %#v", sums)
}

func clearStatsHandler(counts *Counts) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		counts.mu.Lock()
		counts.Calls = make(map[string]int)
		counts.mu.Unlock()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Stats cleared"))
	}
}

func main() {
	// Create Dapr client
	client, err := dapr.NewClient()
	if err != nil {
		panic(err)
	}
	defer client.Close()

	counts := NewCounts()

	// Setup HTTP routes
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/healthz", healthHandler).Methods(http.MethodGet)
	router.HandleFunc("/dapr/config", configHandler).Methods(http.MethodGet)
	router.HandleFunc("/register-reminder", registerActorReminders(client)).Methods(http.MethodPost)
	router.HandleFunc("/unregister-reminder", unregisterActorReminder(client)).Methods(http.MethodPost)
	router.HandleFunc("/clear-stats", clearStatsHandler(counts)).Methods(http.MethodPost)
	router.HandleFunc("/shutdown", shutdownSidecarHandler).Methods(http.MethodPost)
	router.HandleFunc("/actors/{actorType}/{id}/method/{reminderOrTimer}/{method}", counts.actorMethodHandler).Methods(http.MethodPut)
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Not found: %s\n", r.URL.RequestURI())
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	})

	go func() {
		for {
			time.Sleep(1 * time.Second)
			printStats(counts)
		}
	}()

	// Get port from environment variable or use default
	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "6010"
	}
	// Start HTTP server
	log.Printf("Starting HTTP server on port %s\n", appPort)
	log.Fatal(http.ListenAndServe(":"+appPort, router))
}
