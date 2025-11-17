package secretmock

import (
	"context"
	"testing"
)

func TestGetAndPut(t *testing.T) {
	provAny, err := New(map[string]any{"secrets": map[string]any{"token": "abc"}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	val, err := prov.Get(context.Background(), "token")
	if err != nil || val != "abc" {
		t.Fatalf("expected seeded secret, got %s (%v)", val, err)
	}
	if _, err := prov.Get(context.Background(), "missing"); err == nil {
		t.Fatalf("expected error when secret missing")
	}

	if err := prov.Put(context.Background(), "token", "updated"); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	val, _ = prov.Get(context.Background(), "token")
	if val != "updated" {
		t.Fatalf("expected updated secret, got %s", val)
	}
}
