package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/opsorch/opsorch-core/metric"
	"github.com/opsorch/opsorch-core/schema"
	"github.com/opsorch/opsorch-mock-adapters/internal/pluginrpc"
	"github.com/opsorch/opsorch-mock-adapters/metricmock"
)

func main() {
	var (
		prov     metric.Provider
		provOnce sync.Once
		provErr  error
	)

	pluginrpc.Run(func(req pluginrpc.Request) (any, error) {
		provOnce.Do(func() {
			prov, provErr = metricmock.New(req.Config)
		})
		if provErr != nil {
			return nil, provErr
		}

		switch req.Method {
		case "metric.query":
			var q schema.MetricQuery
			if err := json.Unmarshal(req.Payload, &q); err != nil {
				return nil, err
			}
			return prov.Query(context.Background(), q)
		case "metric.describe":
			var scope schema.QueryScope
			if err := json.Unmarshal(req.Payload, &scope); err != nil {
				return nil, err
			}
			return prov.Describe(context.Background(), scope)
		default:
			return nil, errUnknownMethod(req.Method)
		}
	})
}

func errUnknownMethod(method string) error {
	return fmt.Errorf("unknown method %s", method)
}
