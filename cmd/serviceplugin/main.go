package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/opsorch/opsorch-core/schema"
	"github.com/opsorch/opsorch-core/service"
	"github.com/opsorch/opsorch-mock-adapters/internal/pluginrpc"
	"github.com/opsorch/opsorch-mock-adapters/servicemock"
)

func main() {
	var (
		prov     service.Provider
		provOnce sync.Once
		provErr  error
	)

	pluginrpc.Run(func(req pluginrpc.Request) (any, error) {
		provOnce.Do(func() {
			prov, provErr = servicemock.New(req.Config)
		})
		if provErr != nil {
			return nil, provErr
		}

		switch req.Method {
		case "service.query":
			var q schema.ServiceQuery
			if err := json.Unmarshal(req.Payload, &q); err != nil {
				return nil, err
			}
			return prov.Query(context.Background(), q)
		default:
			return nil, errUnknownMethod(req.Method)
		}
	})
}

func errUnknownMethod(method string) error {
	return fmt.Errorf("unknown method %s", method)
}
