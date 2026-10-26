package main

import "testing"

func TestDecideBusinessWorkflow(t *testing.T) {
	tests := []struct {
		name  string
		event PlatformEvent
		want  Delivery
		ok    bool
	}{
		{"subscriber update", PlatformEvent{Kind: "subscription.renewed", Subscriber: "sub_7"}, Delivery{Subscriber: "sub_7"}, true},
		{"asset delivery", PlatformEvent{Kind: "video.job.completed", Subscriber: "sub_7", AssetID: "asset_9"}, Delivery{Subscriber: "sub_7", AssetID: "asset_9"}, true},
		{"content processing", PlatformEvent{Kind: "email.delivered", Content: "episode-12"}, Delivery{Content: "episode-12"}, true},
		{"unknown event", PlatformEvent{Kind: "wallet.low_balance"}, Delivery{}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := decide(test.event)
			if (err == nil) != test.ok || got != test.want {
				t.Fatalf("got %#v, err %v", got, err)
			}
		})
	}
}
