package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/opsorch/opsorch-core/deployment"
	"github.com/opsorch/opsorch-core/schema"
	"github.com/opsorch/opsorch-mock-adapters/deploymentmock"
	"github.com/opsorch/opsorch-mock-adapters/internal/pluginrpc"
)

func main() {
	var (
		prov     deployment.Provider
		provOnce sync.Once
		provErr  error
	)

	pluginrpc.Run(func(req pluginrpc.Request) (any, error) {
		provOnce.Do(func() {
			prov, provErr = deploymentmock.New(req.Config)
		})
		if provErr != nil {
			return nil, provErr
		}

		return handleRequest(prov, req)
	})
}

func handleRequest(prov deployment.Provider, req pluginrpc.Request) (any, error) {
	switch req.Method {
	case "deployment.query":
		var query schema.DeploymentQuery
		if err := json.Unmarshal(req.Payload, &query); err != nil {
			return nil, err
		}
		return prov.Query(context.Background(), query)
	case "deployment.get":
		var payload struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(req.Payload, &payload); err != nil {
			return nil, err
		}
		return prov.Get(context.Background(), payload.ID)
	default:
		return nil, errUnknownMethod(req.Method)
	}
}

func errUnknownMethod(method string) error {
	return fmt.Errorf("unknown method %s", method)
}
