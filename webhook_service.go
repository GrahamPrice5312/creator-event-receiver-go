package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type PlatformEvent struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Subscriber string `json:"subscriber_id"`
	AssetID    string `json:"asset_id"`
	Content    string `json:"content"`
}

type Delivery struct {
	Subscriber string
	AssetID    string
	Content    string
}

func decide(event PlatformEvent) (Delivery, error) {
	switch event.Kind {
	case "subscription.activated", "subscription.renewed":
		if event.Subscriber == "" {
			return Delivery{}, errors.New("subscriber_id is required")
		}
		return Delivery{Subscriber: event.Subscriber}, nil
	case "image.job.completed", "video.job.completed":
		if event.Subscriber == "" || event.AssetID == "" {
			return Delivery{}, errors.New("subscriber_id and asset_id are required")
		}
		return Delivery{Subscriber: event.Subscriber, AssetID: event.AssetID}, nil
	case "email.delivered":
		if event.Content == "" {
			return Delivery{}, errors.New("content is required")
		}
		return Delivery{Content: event.Content}, nil
	default:
		return Delivery{}, errors.New("event is not handled")
	}
}

func verifySignature(body []byte, supplied, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(supplied))
}

func webhookHandler(secret string, enqueue func(PlatformEvent) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		if !verifySignature(body, r.Header.Get("X-Infrai-Signature"), secret) {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}
		var event PlatformEvent
		if err := json.Unmarshal(body, &event); err != nil || event.ID == "" {
			http.Error(w, "invalid event", http.StatusBadRequest)
			return
		}
		if err := enqueue(event); err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	})
}
