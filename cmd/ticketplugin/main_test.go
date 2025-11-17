package main

import (
	"encoding/json"
	"testing"

	"github.com/opsorch/opsorch-core/schema"
	"github.com/opsorch/opsorch-mock-adapters/internal/pluginrpc"
	"github.com/opsorch/opsorch-mock-adapters/ticketmock"
)

func TestHandleRequestQuery(t *testing.T) {
	prov, err := ticketmock.New(map[string]any{})
	if err != nil {
		t.Fatalf("failed to init provider: %v", err)
	}

	payload, err := json.Marshal(schema.TicketQuery{Query: "checkout"})
	if err != nil {
		t.Fatalf("failed to marshal query: %v", err)
	}

	res, err := handleRequest(prov, pluginrpc.Request{Method: "ticket.query", Payload: payload})
	if err != nil {
		t.Fatalf("handleRequest returned error: %v", err)
	}

	tickets, ok := res.([]schema.Ticket)
	if !ok {
		t.Fatalf("expected []schema.Ticket response, got %T", res)
	}
	if len(tickets) == 0 {
		t.Fatalf("expected seeded tickets in response")
	}
}

func TestHandleRequestUnknownMethod(t *testing.T) {
	prov, err := ticketmock.New(map[string]any{})
	if err != nil {
		t.Fatalf("failed to init provider: %v", err)
	}

	if _, err := handleRequest(prov, pluginrpc.Request{Method: "ticket.invalid"}); err == nil {
		t.Fatalf("expected error for unknown method")
	}
}
