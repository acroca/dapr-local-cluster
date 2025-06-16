package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	dapr "github.com/dapr/go-sdk/client"
)

func main() {
	// Create a new client for Dapr using the SDK
	client, err := dapr.NewClient()
	if err != nil {
		panic(err)
	}
	defer client.Close()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	i := 0
	for {
		select {
		case <-ticker.C:
			i++
			err := client.PublishEvent(context.Background(), "pubsub", "numbers", []byte(`{"number":`+strconv.Itoa(i)+`}`))
			if err != nil {
				panic(err)
			}
			fmt.Println("Published event", i)
		}
	}
}
