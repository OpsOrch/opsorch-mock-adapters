package secretmock

import (
	"context"
	"fmt"
	"sync"

	"github.com/opsorch/opsorch-core/orcherr"
	"github.com/opsorch/opsorch-core/secret"
)

// ProviderName can be referenced via OPSORCH_SECRET_PROVIDER.
const ProviderName = "mock"

// Config seeds the secret store.
type Config struct {
	Secrets map[string]string
}

// Provider stores secrets in-memory.
type Provider struct {
	store map[string]string
	mu    sync.Mutex
}

// New constructs the mock secret provider.
func New(cfg map[string]any) (secret.Provider, error) {
	parsed := parseConfig(cfg)
	if len(parsed.Secrets) == 0 {
		parsed.Secrets = defaultSecrets()
	}
	store := make(map[string]string, len(parsed.Secrets))
	for k, v := range parsed.Secrets {
		store[k] = v
	}
	return &Provider{store: store}, nil
}

func init() {
	_ = secret.RegisterProvider(ProviderName, New)
}

// Get returns a plaintext secret.
func (p *Provider) Get(ctx context.Context, key string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if val, ok := p.store[key]; ok {
		return val, nil
	}
	return "", orcherr.New("not_found", fmt.Sprintf("%s not found", key), nil)
}

// Put stores or updates a plaintext secret.
func (p *Provider) Put(ctx context.Context, key, value string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.store[key] = value
	return nil
}

func parseConfig(cfg map[string]any) Config {
	out := Config{Secrets: map[string]string{}}
	if raw, ok := cfg["secrets"].(map[string]any); ok {
		for k, v := range raw {
			if val, ok := v.(string); ok {
				out.Secrets[k] = val
			}
		}
	}
	if raw, ok := cfg["secrets"].(map[string]string); ok {
		for k, v := range raw {
			out.Secrets[k] = v
		}
	}
	return out
}

func defaultSecrets() map[string]string {
	return map[string]string{
		"db/checkout/password":  "ch3ck0ut-demo#2024",
		"slack/webhook/ops":     "https://hooks.slack.com/services/T00000000/B00000000/placeholder",
		"api/stripe/key":        "sk_test_mock123",
		"gcp/service-account":   "{\"type\":\"service_account\",\"project_id\":\"mock-demo\"}",
		"secrets/feature-flags": "enabled=true, cohorts=alpha",
	}
}

var _ secret.Provider = (*Provider)(nil)
