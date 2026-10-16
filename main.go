package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("set INFRAI_API_KEY")
	}
	client := InfraiClient{Key: key, HTTP: http.DefaultClient}
	agent := NewAgent(key, client, client)
	job, err := agent.Deliver(context.Background(), StreamJob{
		AssetID: "asset-demo-42", Creator: "mira", Transcript: "A short trailer for the new episode.",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("asset=%s delivery=%s\n", job.AssetID, job.DeliveryState)
}
