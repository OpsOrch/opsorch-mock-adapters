package pluginrpc

import (
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/opsorch/opsorch-core/orcherr"
)

// Request mirrors the JSON payload OpsOrch sends to plugins.
type Request struct {
	Method  string          `json:"method"`
	Config  map[string]any  `json:"config"`
	Payload json.RawMessage `json:"payload"`
}

// Response is emitted for every request.
type Response struct {
	Result any         `json:"result,omitempty"`
	Error  *errorValue `json:"error,omitempty"`
}

type errorValue struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

// Run decodes requests from stdin, dispatches to handler, and writes responses to stdout.
func Run(handler func(Request) (any, error)) {
	dec := json.NewDecoder(os.Stdin)
	enc := json.NewEncoder(os.Stdout)

	for {
		var req Request
		if err := dec.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			_ = enc.Encode(Response{Error: toErrorValue(err)})
			return
		}

		res, err := handler(req)
		if err != nil {
			_ = enc.Encode(Response{Error: toErrorValue(err)})
			continue
		}
		_ = enc.Encode(Response{Result: res})
	}
}

func toErrorValue(err error) *errorValue {
	if err == nil {
		return nil
	}
	var oe orcherr.OpsOrchError
	if errors.As(err, &oe) {
		return &errorValue{Code: oe.Code, Message: oe.Message}
	}
	return &errorValue{Message: err.Error()}
}
