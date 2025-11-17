package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/opsorch/opsorch-core/schema"
	"github.com/opsorch/opsorch-core/ticket"
	"github.com/opsorch/opsorch-mock-adapters/internal/pluginrpc"
	"github.com/opsorch/opsorch-mock-adapters/ticketmock"
)

func main() {
	var (
		prov     ticket.Provider
		provOnce sync.Once
		provErr  error
	)

	pluginrpc.Run(func(req pluginrpc.Request) (any, error) {
		provOnce.Do(func() {
			prov, provErr = ticketmock.New(req.Config)
		})
		if provErr != nil {
			return nil, provErr
		}

		return handleRequest(prov, req)
	})
}

func handleRequest(prov ticket.Provider, req pluginrpc.Request) (any, error) {
	switch req.Method {
	case "ticket.query":
		var query schema.TicketQuery
		if err := json.Unmarshal(req.Payload, &query); err != nil {
			return nil, err
		}
		return prov.Query(context.Background(), query)
	case "ticket.get":
		var payload struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(req.Payload, &payload); err != nil {
			return nil, err
		}
		return prov.Get(context.Background(), payload.ID)
	case "ticket.create":
		var in schema.CreateTicketInput
		if err := json.Unmarshal(req.Payload, &in); err != nil {
			return nil, err
		}
		return prov.Create(context.Background(), in)
	case "ticket.update":
		var payload struct {
			ID    string                   `json:"id"`
			Input schema.UpdateTicketInput `json:"input"`
		}
		if err := json.Unmarshal(req.Payload, &payload); err != nil {
			return nil, err
		}
		return prov.Update(context.Background(), payload.ID, payload.Input)
	default:
		return nil, errUnknownMethod(req.Method)
	}
}

func errUnknownMethod(method string) error {
	return fmt.Errorf("unknown method %s", method)
}
