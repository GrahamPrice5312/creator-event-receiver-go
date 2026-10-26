package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	client, err := newClient()
	if err != nil {
		log.Fatal(err)
	}
	secret := os.Getenv("WEBHOOK_SECRET")
	if secret == "" {
		log.Fatal("WEBHOOK_SECRET is required")
	}
	webhookURL := os.Getenv("WEBHOOK_URL")
	if webhookURL == "" {
		log.Fatal("WEBHOOK_URL is required")
	}
	queue := "creator-events"
	if err := client.createQueue("creator-events-dlq", ""); err != nil {
		log.Fatal(err)
	}
	if err := client.createQueue(queue, "creator-events-dlq"); err != nil {
		log.Fatal(err)
	}
	webhookID, err := client.registerWebhook(webhookURL, secret)
	if err != nil {
		log.Fatal(err)
	}
	if err := client.deliveries(webhookID); err != nil {
		log.Fatal(err)
	}
	handler := webhookHandler(secret, func(event PlatformEvent) error {
		if _, err := decide(event); err != nil {
			return err
		}
		return client.publish(queue, event)
	})
	log.Printf("creator event receiver listening on :8080 (webhook %s)", webhookID)
	log.Fatal(http.ListenAndServe(":8080", handler))
}
