package messagingmock

import (
	"context"
	"fmt"
	"strings"
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
	channelType := detectChannelType(msg.Channel)
	metadata["provider"] = provider
	metadata["channelType"] = channelType
	metadata["preview"] = previewBody(msg.Body)
	metadata["providerMessageId"] = fmt.Sprintf("%s-%04d", provider, p.nextID)

	// Simulate realistic delivery patterns
	now := time.Now().UTC()
	deliveryPattern := p.simulateDeliveryPattern(p.nextID, channelType)

	metadata["status"] = deliveryPattern.Status
	metadata["latencyMs"] = deliveryPattern.LatencyMs
	metadata["deliveryState"] = deliveryPattern.State

	// Add retry information if applicable
	if deliveryPattern.RetryCount > 0 {
		metadata["retryCount"] = deliveryPattern.RetryCount
		metadata["retryHistory"] = deliveryPattern.RetryHistory
	}

	// Add failure reason if failed
	if deliveryPattern.Status == "failed" {
		metadata["failureReason"] = deliveryPattern.FailureReason
	}

	// Add throttling info if throttled
	if deliveryPattern.Throttled {
		metadata["throttled"] = true
		metadata["queueDepth"] = deliveryPattern.QueueDepth
	}

	result := schema.MessageResult{
		ID:       id,
		Channel:  msg.Channel,
		SentAt:   now,
		Metadata: metadata,
	}

	p.history = append(p.history, result)
	return result, nil
}

// DeliveryPattern represents the delivery characteristics of a message.
type DeliveryPattern struct {
	Status        string
	LatencyMs     int
	State         map[string]any
	RetryCount    int
	RetryHistory  []map[string]any
	FailureReason string
	Throttled     bool
	QueueDepth    int
}

// simulateDeliveryPattern simulates realistic delivery patterns including delays, retries, and failures.
func (p *Provider) simulateDeliveryPattern(msgID int, channelType string) DeliveryPattern {
	now := time.Now().UTC()

	// 5% of messages fail initially and require retries
	shouldRetry := (msgID % 20) == 0

	// 1% of messages fail permanently
	shouldFail := (msgID % 100) == 0

	// 10% of messages are throttled
	isThrottled := (msgID % 10) == 0

	pattern := DeliveryPattern{
		Status:    "delivered",
		LatencyMs: p.calculateLatency(channelType, isThrottled),
		State:     make(map[string]any),
	}

	queuedAt := now.Add(-time.Duration(pattern.LatencyMs) * time.Millisecond)
	pattern.State["queuedAt"] = queuedAt

	if shouldFail {
		// Permanent failure after retries
		pattern.Status = "failed"
		pattern.RetryCount = 3
		pattern.FailureReason = p.getFailureReason(channelType)
		pattern.RetryHistory = p.generateRetryHistory(queuedAt, 3, true)
		pattern.State["failedAt"] = now
	} else if shouldRetry {
		// Successful after retries
		pattern.Status = "delivered"
		pattern.RetryCount = 2
		pattern.RetryHistory = p.generateRetryHistory(queuedAt, 2, false)
		pattern.State["deliveredAt"] = now
	} else {
		// Successful on first attempt
		pattern.State["deliveredAt"] = now
	}

	if isThrottled {
		pattern.Throttled = true
		pattern.QueueDepth = 50 + (msgID % 200)
	}

	return pattern
}

// calculateLatency calculates realistic latency based on channel type and throttling.
func (p *Provider) calculateLatency(channelType string, throttled bool) int {
	baseLatency := map[string]int{
		"email": 500,
		"sms":   200,
		"slack": 150,
		"push":  100,
	}

	latency := baseLatency[channelType]
	if latency == 0 {
		latency = 300
	}

	// Add jitter
	latency += (p.nextID % 100)

	// Throttled messages take longer
	if throttled {
		latency += 1000
	}

	return latency
}

// getFailureReason returns a realistic failure reason based on channel type.
func (p *Provider) getFailureReason(channelType string) string {
	reasons := map[string][]string{
		"email": {
			"Recipient mailbox full",
			"SMTP connection timeout",
			"Invalid recipient address",
		},
		"sms": {
			"Invalid phone number",
			"Carrier rejected message",
			"Number not in service",
		},
		"slack": {
			"Channel not found",
			"Bot not in channel",
			"Rate limit exceeded",
		},
		"push": {
			"Device token invalid",
			"App not installed",
			"Push service unavailable",
		},
	}

	channelReasons := reasons[channelType]
	if len(channelReasons) == 0 {
		return "Delivery failed"
	}

	return channelReasons[p.nextID%len(channelReasons)]
}

// generateRetryHistory generates a realistic retry history with exponential backoff.
func (p *Provider) generateRetryHistory(queuedAt time.Time, retryCount int, failed bool) []map[string]any {
	history := make([]map[string]any, retryCount)

	currentTime := queuedAt
	backoff := 1 * time.Second

	for i := 0; i < retryCount; i++ {
		currentTime = currentTime.Add(backoff)

		attempt := map[string]any{
			"attempt":   i + 1,
			"attemptAt": currentTime,
		}

		if i < retryCount-1 || failed {
			attempt["status"] = "failed"
			attempt["reason"] = "Temporary failure"
		} else {
			attempt["status"] = "success"
		}

		history[i] = attempt

		// Exponential backoff: 1s, 2s, 4s, 8s, ...
		backoff *= 2
	}

	return history
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

func detectChannelType(channel string) string {
	switch {
	case strings.HasPrefix(channel, "#"):
		return "chat"
	case strings.HasPrefix(channel, "sms:"):
		return "sms"
	case strings.Contains(channel, "@"):
		return "email"
	default:
		return "generic"
	}
}

func previewBody(body string) string {
	trimmed := strings.TrimSpace(body)
	if len([]rune(trimmed)) > 80 {
		runes := []rune(trimmed)
		return string(runes[:77]) + "..."
	}
	return trimmed
}
