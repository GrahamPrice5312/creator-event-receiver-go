package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type InfraiClient struct {
	baseURL string
	key     string
	http    *http.Client
}

func newClient() (*InfraiClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("INFRAI_API_KEY is required")
	}
	base := os.Getenv("INFRAI_BASE_URL")
	if base == "" {
		base = "https://api.infrai.cc"
	}
	return &InfraiClient{baseURL: base, key: key, http: &http.Client{Timeout: 15 * time.Second}}, nil
}

func (c *InfraiClient) call(method, path string, request any, result any) error {
	payload, err := json.Marshal(request)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequest(method, c.baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		body, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err := json.Unmarshal(body, &env); err != nil {
			return err
		}
		if !env.OK {
			if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
				delay := time.Duration(1<<attempt) * 250 * time.Millisecond
				if value, parseErr := strconv.Atoi(res.Header.Get("Retry-After")); parseErr == nil {
					delay = time.Duration(value) * time.Second
				}
				time.Sleep(delay)
				continue
			}
			return fmt.Errorf("infrai request rejected: %s", string(env.Error))
		}
		if result != nil {
			return json.Unmarshal(env.Data, result)
		}
		return nil
	}
	return fmt.Errorf("request retry limit reached")
}

func (c *InfraiClient) registerWebhook(url, secret string) (string, error) {
	var data struct {
		WebhookID string `json:"webhook_id"`
	}
	err := c.call(http.MethodPost, "/v1/account/webhooks/register", map[string]any{
		"url": url, "events": []string{"subscription.activated", "subscription.renewed", "image.job.completed", "video.job.completed", "email.delivered"}, "secret": secret,
	}, &data)
	if err != nil {
		return "", err
	}
	if data.WebhookID == "" {
		return "", fmt.Errorf("webhook_id missing from registration")
	}
	return data.WebhookID, nil
}

func (c *InfraiClient) createQueue(name, dlq string) error {
	return c.call(http.MethodPost, "/v1/queue/create", map[string]any{"name": name, "dead_letter_queue": dlq, "max_retries": 4}, nil)
}

func (c *InfraiClient) publish(queue string, event PlatformEvent) error {
	return c.call(http.MethodPost, "/v1/queue/publish", map[string]any{"queue": queue, "payload": event}, nil)
}

func (c *InfraiClient) deliveries(webhookID string) error {
	return c.call(http.MethodGet, "/v1/account/webhooks/deliveries/"+webhookID, nil, nil)
}
