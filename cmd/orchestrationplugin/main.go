package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/opsorch/opsorch-core/orchestration"
	"github.com/opsorch/opsorch-core/schema"
	"github.com/opsorch/opsorch-mock-adapters/internal/pluginrpc"
	"github.com/opsorch/opsorch-mock-adapters/orchestrationmock"
)

func main() {
	var (
		prov     orchestration.Provider
		provOnce sync.Once
		provErr  error
	)

	pluginrpc.Run(func(req pluginrpc.Request) (any, error) {
		provOnce.Do(func() {
			prov, provErr = orchestrationmock.New(req.Config)
		})
		if provErr != nil {
			return nil, provErr
		}

		switch req.Method {
		case "orchestration.plans.query":
			var q schema.OrchestrationPlanQuery
			if err := json.Unmarshal(req.Payload, &q); err != nil {
				return nil, err
			}
			return prov.QueryPlans(context.Background(), q)

		case "orchestration.plans.get":
			var payload struct {
				PlanID string `json:"planId"`
			}
			if err := json.Unmarshal(req.Payload, &payload); err != nil {
				return nil, err
			}
			return prov.GetPlan(context.Background(), payload.PlanID)

		case "orchestration.runs.query":
			var q schema.OrchestrationRunQuery
			if err := json.Unmarshal(req.Payload, &q); err != nil {
				return nil, err
			}
			return prov.QueryRuns(context.Background(), q)

		case "orchestration.runs.get":
			var payload struct {
				RunID string `json:"runId"`
			}
			if err := json.Unmarshal(req.Payload, &payload); err != nil {
				return nil, err
			}
			return prov.GetRun(context.Background(), payload.RunID)

		case "orchestration.runs.start":
			var payload struct {
				PlanID string `json:"planId"`
			}
			if err := json.Unmarshal(req.Payload, &payload); err != nil {
				return nil, err
			}
			return prov.StartRun(context.Background(), payload.PlanID)

		case "orchestration.runs.steps.complete":
			var payload struct {
				RunID  string `json:"runId"`
				StepID string `json:"stepId"`
				Actor  string `json:"actor"`
				Note   string `json:"note"`
			}
			if err := json.Unmarshal(req.Payload, &payload); err != nil {
				return nil, err
			}
			err := prov.CompleteStep(context.Background(), payload.RunID, payload.StepID, payload.Actor, payload.Note)
			if err != nil {
				return nil, err
			}
			return nil, nil

		default:
			return nil, errUnknownMethod(req.Method)
		}
	})
}

func errUnknownMethod(method string) error {
	return fmt.Errorf("unknown method %s", method)
}
