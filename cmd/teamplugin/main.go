package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/opsorch/opsorch-core/schema"
	"github.com/opsorch/opsorch-core/team"
	"github.com/opsorch/opsorch-mock-adapters/internal/pluginrpc"
	"github.com/opsorch/opsorch-mock-adapters/teammock"
)

func main() {
	var (
		prov     team.Provider
		provOnce sync.Once
		provErr  error
	)

	pluginrpc.Run(func(req pluginrpc.Request) (any, error) {
		provOnce.Do(func() {
			prov, provErr = teammock.New(req.Config)
		})
		if provErr != nil {
			return nil, provErr
		}

		switch req.Method {
		case "team.query":
			var q schema.TeamQuery
			if err := json.Unmarshal(req.Payload, &q); err != nil {
				return nil, err
			}
			return prov.Query(context.Background(), q)
		case "team.get":
			var params struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(req.Payload, &params); err != nil {
				return nil, err
			}
			return prov.Get(context.Background(), params.ID)
		case "team.members":
			var params struct {
				TeamID string `json:"teamID"`
			}
			if err := json.Unmarshal(req.Payload, &params); err != nil {
				return nil, err
			}
			return prov.Members(context.Background(), params.TeamID)
		default:
			return nil, errUnknownMethod(req.Method)
		}
	})
}

func errUnknownMethod(method string) error {
	return fmt.Errorf("unknown method %s", method)
}
