package messagingmock

import (
	"context"
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
