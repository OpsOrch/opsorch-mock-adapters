package messagingmock

import (
	"context"
	"strings"
	"testing"

	"github.com/opsorch/opsorch-core/schema"
)

func TestSendAndHistory(t *testing.T) {
	provAny, err := New(map[string]any{"provider": "demo"})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	result, err := prov.Send(context.Background(), schema.Message{Channel: "#ops", Body: "hello"})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if result.ID == "" {
		t.Fatalf("unexpected message result: %+v", result)
	}
	if result.Metadata["provider"] != "demo" {
		t.Fatalf("expected provider metadata: %+v", result.Metadata)
	}
	if result.Metadata["status"] != "delivered" {
		t.Fatalf("expected status metadata, got %+v", result.Metadata)
	}

	history := prov.History()
	if len(history) != 1 || history[0].ID != result.ID {
		t.Fatalf("history did not record message: %+v", history)
	}

	history[0].Metadata["status"] = "mutated"
	if prov.History()[0].Metadata["status"] == "mutated" {
		t.Fatalf("history should be cloned")
	}
}
func TestMessagingURLGeneration(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	prov := provAny.(*Provider)

	result, err := prov.Send(context.Background(), schema.Message{
		Channel: "#test-channel",
		Body:    "Test message",
	})
	if err != nil {
		t.Fatalf("failed to send message: %v", err)
	}

	if result.URL == "" {
		t.Errorf("message result has empty URL")
	}
	if !strings.HasPrefix(result.URL, "https://slack.demo.com/archives/") {
		t.Errorf("message result has invalid URL format: %s", result.URL)
	}
	if !strings.Contains(result.URL, "test-channel") {
		t.Errorf("message URL should contain channel name: %s", result.URL)
	}
	if !strings.Contains(result.URL, "/p") {
		t.Errorf("message URL should contain /p for message permalink: %s", result.URL)
	}
}
