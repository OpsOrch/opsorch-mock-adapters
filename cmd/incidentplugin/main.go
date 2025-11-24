package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/opsorch/opsorch-core/incident"
	"github.com/opsorch/opsorch-core/schema"
	"github.com/opsorch/opsorch-mock-adapters/incidentmock"
	"github.com/opsorch/opsorch-mock-adapters/internal/pluginrpc"
)

func main() {
	var (
		prov     incident.Provider
		provOnce sync.Once
		provErr  error
	)

	pluginrpc.Run(func(req pluginrpc.Request) (any, error) {
		provOnce.Do(func() {
			prov, provErr = incidentmock.New(req.Config)
		})
		if provErr != nil {
			return nil, provErr
		}

		switch req.Method {
		case "incident.query":
			var q schema.IncidentQuery
			if err := json.Unmarshal(req.Payload, &q); err != nil {
				return nil, err
			}
			return prov.Query(context.Background(), q)
		case "incident.list":
			return prov.Query(context.Background(), schema.IncidentQuery{})
		case "incident.get":
			var payload struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(req.Payload, &payload); err != nil {
				return nil, err
			}
			return prov.Get(context.Background(), payload.ID)
		case "incident.create":
			var in schema.CreateIncidentInput
			if err := json.Unmarshal(req.Payload, &in); err != nil {
				return nil, err
			}
			return prov.Create(context.Background(), in)
		case "incident.update":
			var payload struct {
				ID    string                     `json:"id"`
				Input schema.UpdateIncidentInput `json:"input"`
			}
			if err := json.Unmarshal(req.Payload, &payload); err != nil {
				return nil, err
			}
			return prov.Update(context.Background(), payload.ID, payload.Input)
		case "incident.timeline.get":
			var payload struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(req.Payload, &payload); err != nil {
				return nil, err
			}
			return prov.GetTimeline(context.Background(), payload.ID)
		case "incident.timeline.append":
			var payload struct {
				ID    string                     `json:"id"`
				Entry schema.TimelineAppendInput `json:"entry"`
			}
			if err := json.Unmarshal(req.Payload, &payload); err != nil {
				return nil, err
			}
			return nil, prov.AppendTimeline(context.Background(), payload.ID, payload.Entry)
		default:
			return nil, errUnknownMethod(req.Method)
		}
	})
}

func errUnknownMethod(method string) error {
	return fmt.Errorf("unknown method %s", method)
}
