package messagingmock

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/opsorch/opsorch-core/messaging"
	"github.com/opsorch/opsorch-core/schema"
	"github.com/opsorch/opsorch-mock-adapters/internal/mockutil"
)

// ProviderName can be referenced via OPSORCH_MESSAGING_PROVIDER.
const ProviderName = "mock"

// Config controls message metadata.
type Config struct {
	Provider string
}

// Provider stores sent messages in-memory for demo feedback.
type Provider struct {
	cfg     Config
	mu      sync.Mutex
	nextID  int
	history []schema.MessageResult
}

// New constructs the mock messaging provider.
func New(cfg map[string]any) (messaging.Provider, error) {
	parsed := parseConfig(cfg)
	return &Provider{cfg: parsed}, nil
}

func init() {
	_ = messaging.RegisterProvider(ProviderName, New)
}

// Send records the message send and returns a synthetic provider response.
func (p *Provider) Send(ctx context.Context, msg schema.Message) (schema.MessageResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.nextID++
	id := fmt.Sprintf("msg-%04d", p.nextID)
	provider := p.cfg.Provider
	if msg.Metadata != nil {
		if v, ok := msg.Metadata["provider"].(string); ok && v != "" {
			provider = v
		}
	}

	metadata := mockutil.CloneMap(msg.Metadata)
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["provider"] = provider
	metadata["status"] = "delivered"

	result := schema.MessageResult{
		ID:       id,
		Channel:  msg.Channel,
		SentAt:   time.Now().UTC(),
		Metadata: metadata,
	}

	p.history = append(p.history, result)
	return result, nil
}

func parseConfig(cfg map[string]any) Config {
	out := Config{Provider: "mock"}
	if v, ok := cfg["provider"].(string); ok && v != "" {
		out.Provider = v
	}
	return out
}

// History returns a copy of delivered messages for demos/tests.
func (p *Provider) History() []schema.MessageResult {
	p.mu.Lock()
	defer p.mu.Unlock()

	out := make([]schema.MessageResult, len(p.history))
	copy(out, p.history)
	for i := range out {
		out[i].Metadata = mockutil.CloneMap(out[i].Metadata)
	}
	return out
}

var _ messaging.Provider = (*Provider)(nil)
