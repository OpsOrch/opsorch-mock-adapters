package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/opsorch/opsorch-core/secret"
	"github.com/opsorch/opsorch-mock-adapters/internal/pluginrpc"
	"github.com/opsorch/opsorch-mock-adapters/secretmock"
)

func main() {
	var (
		prov     secret.Provider
		provOnce sync.Once
		provErr  error
	)

	pluginrpc.Run(func(req pluginrpc.Request) (any, error) {
		provOnce.Do(func() {
			prov, provErr = secretmock.New(req.Config)
		})
		if provErr != nil {
			return nil, provErr
		}

		switch req.Method {
		case "secret.get":
			var payload struct {
				Key string `json:"key"`
			}
			if err := json.Unmarshal(req.Payload, &payload); err != nil {
				return nil, err
			}
			return prov.Get(context.Background(), payload.Key)
		case "secret.put":
			var payload struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			}
			if err := json.Unmarshal(req.Payload, &payload); err != nil {
				return nil, err
			}
			return nil, prov.Put(context.Background(), payload.Key, payload.Value)
		default:
			return nil, errUnknownMethod(req.Method)
		}
	})
}

func errUnknownMethod(method string) error {
	return fmt.Errorf("unknown method %s", method)
}
