# OpsOrch Mock Adapters

[![Version](https://img.shields.io/github/v/release/opsorch/opsorch-mock-adapters)](https://github.com/opsorch/opsorch-mock-adapters/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/opsorch/opsorch-mock-adapters)](https://github.com/opsorch/opsorch-mock-adapters/blob/main/go.mod)
[![License](https://img.shields.io/github/license/opsorch/opsorch-mock-adapters)](https://github.com/opsorch/opsorch-mock-adapters/blob/main/LICENSE)

Mock implementations of the OpsOrch provider interfaces that keep everything in-memory and return deterministic, "realistic enough" demo data. These adapters are designed for demos, testing, and development - not for production use.

## Capabilities

This repository provides mock implementations for all OpsOrch capabilities:

1. **Alert Provider**: In-memory alerts with scripted lifecycle transitions
2. **Incident Provider**: In-memory incidents with timeline support
3. **Log Provider**: Synthetic log generation with alert correlation
4. **Metric Provider**: Deterministic metric waveforms with anomaly injection
5. **Ticket Provider**: In-memory ticket store with scenario data
6. **Messaging Provider**: Simulated message delivery with metadata
7. **Service Provider**: Static service catalog
8. **Secret Provider**: Simple key/value secret store

## Features

These adapters intentionally stay simple: data is seeded from Go structs, scoped filtering happens in-process, and a handful of scenario-labeled records make it easy to demonstrate cascading failures without talking to any real systems.

### Alert Provider (`alertmock`)
- Seeds ~30 alerts covering checkout, search, database, payments, ML, and infra services
- Supports free-text search, scope filters (service/environment/team), severity/status filters
- Enriched with runbooks, dashboards, escalation policies, Slack channels, deployment context
- Scripted lifecycle: some alerts transition firing → acknowledged → resolved over time
- Alert snapshots available for correlation with logs and metrics

### Incident Provider (`incidentmock`)
- Seeds in-memory incidents plus timelines
- Six scenario incidents with `scenario_id`, `scenario_name`, `Metadata["is_scenario"]`
- Supports Query, Get, Create, Update, GetTimeline, AppendTimeline
- Filters by scope, severity, status, and search terms

### Log Provider (`logmock`)
- Generates synthetic log entries within requested time windows
- Incorporates alert snapshots: adjusts severity/latency when matching alerts are active
- Emits static scenario logs flagged by `Fields["is_scenario"]`
- Applies caller-provided filters for equality/inequality/contains matching

### Metric Provider (`metricmock`)
- Describes 40+ metric definitions (counters, gauges, histograms)
- Builds deterministic waveforms with daily patterns, noise, and growth trends
- Returns "active" series plus computed baseline for each descriptor
- Static scenario anomalies inject spikes, drops, or plateaus with `scenario_effects` metadata
- Describe returns full metric catalog for UI dropdowns

### Ticket Provider (`ticketmock`)
- Maintains in-memory ticket store with seeded work items
- Scenario tickets flagged with `Fields["is_scenario"]`
- Supports Query, Get, Create, Update
- Enriched with runbook links, checklists, dependency hints, due dates

### Messaging Provider (`messagingmock`)
- Simulates message delivery, records requests in-memory
- Returns latency, retry history, throttling info, failure reasons in metadata
- Delivery characteristics vary by channel (chat/email/SMS)
- History() exposes previously sent messages

### Service Provider (`servicemock`)
- Serves static service catalog (frontend, backend, data tiers)
- Each service includes tags (env, tier, owner) and metadata (runbooks, dashboards, repos)
- Supports filtering by IDs, name substrings, tags, and query scope

### Secret Provider (`secretmock`)
- Extremely small key/value secret store for demos
- Defaults include DB passwords, Stripe keys, Slack webhooks
- Supports Get and Put for secret rotation flows

## Configuration

Mock adapters have minimal configuration requirements since they don't connect to external systems.

### Alert Provider

| Field | Type | Required | Description | Default |
|-------|------|----------|-------------|---------|
| `source` | string | No | Source identifier stamped in `Metadata["source"]` | `mock-alert` |

### Incident Provider

| Field | Type | Required | Description | Default |
|-------|------|----------|-------------|---------|
| `source` | string | No | Source identifier | `mock` |
| `defaultSeverity` | string | No | Default severity for new incidents | `sev2` |

### Log Provider

| Field | Type | Required | Description | Default |
|-------|------|----------|-------------|---------|
| `defaultLimit` | int | No | Default number of logs to return | `50` |
| `source` | string | No | Source identifier | `mock` |

### Metric Provider

| Field | Type | Required | Description | Default |
|-------|------|----------|-------------|---------|
| `source` | string | No | Source identifier for metadata annotations | `mock` |

### Ticket Provider

| Field | Type | Required | Description | Default |
|-------|------|----------|-------------|---------|
| `source` | string | No | Source identifier stamped in `Metadata["source"]` | `mock` |

### Messaging Provider

| Field | Type | Required | Description | Default |
|-------|------|----------|-------------|---------|
| `provider` | string | No | Provider name for message IDs | `mock` |

### Service Provider

| Field | Type | Required | Description | Default |
|-------|------|----------|-------------|---------|
| `environment` | string | No | Environment tag stamped on all services | `prod` |

### Secret Provider

| Field | Type | Required | Description | Default |
|-------|------|----------|-------------|---------|
| `secrets` | map | No | Pre-seeded key/value pairs | Default secrets (DB passwords, API keys) |

## Usage

### Embed Directly Inside OpsOrch Core

Import the packages for side effects so the providers self-register under the `mock` name:

```go
import (
    _ "github.com/opsorch/opsorch-mock-adapters/alertmock"
    _ "github.com/opsorch/opsorch-mock-adapters/incidentmock"
    _ "github.com/opsorch/opsorch-mock-adapters/logmock"
    _ "github.com/opsorch/opsorch-mock-adapters/metricmock"
    _ "github.com/opsorch/opsorch-mock-adapters/ticketmock"
    _ "github.com/opsorch/opsorch-mock-adapters/messagingmock"
    _ "github.com/opsorch/opsorch-mock-adapters/servicemock"
    _ "github.com/opsorch/opsorch-mock-adapters/secretmock"
)
```

Point OpsOrch Core at the mocks by setting the provider names:

```bash
OPSORCH_INCIDENT_PROVIDER=mock \
OPSORCH_ALERT_PROVIDER=mock \
OPSORCH_LOG_PROVIDER=mock \
OPSORCH_METRIC_PROVIDER=mock \
OPSORCH_TICKET_PROVIDER=mock \
OPSORCH_MESSAGING_PROVIDER=mock \
OPSORCH_SERVICE_PROVIDER=mock \
OPSORCH_SECRET_PROVIDER=mock \
go run ./cmd/opsorch
```

### Build Plugin Binaries

Every folder in `cmd/*plugin` exposes the same providers over OpsOrch's plugin RPC (stdin/stdout JSON). Build all binaries with:

```bash
make plugin         # outputs bin/<capability>plugin
```

Set `PLUGINS="alertplugin metricplugin"` to limit the build. Wire a plugin into OpsOrch Core via `OPSORCH_<CAPABILITY>_PLUGIN=/full/path/to/bin/<capability>plugin`.

### Demo Docker Image

The provided Dockerfile layers the plugin binaries onto the published OpsOrch Core image and defaults every `OPSORCH_*_PLUGIN` env var to the bundled mocks. Build and run it locally with:

```bash
# Build (defaults to ghcr.io/opsorch/opsorch-core:latest as the base)
make docker

# Run the demo API on localhost:8080
make run
```

Override the base image or build without make:

```bash
make docker BASE_IMAGE=ghcr.io/...:v0.0.4
docker build -f Dockerfile -t opsorch-mock-adapters:latest --build-arg BASE_IMAGE=ghcr.io/opsorch/opsorch-core:latest .
```

## Scenario Data

Scenario fixtures are implemented as static Go slices inside each provider (e.g. `getScenarioLogs`, `getScenarioMetricAnomalies`, `getScenarioTickets`). There is no runtime scenario engine or toggle—scenario records are always available and can be identified via `Metadata["is_scenario"]`/`Fields["scenario_*"]`.

Scenario data demonstrates cascading failures across multiple services and capabilities, making it easy to show how OpsOrch correlates alerts, logs, metrics, and incidents.

## Development

### Prerequisites

- Go 1.22+
- No external services required (all data is in-memory)

### Building

```bash
# Run unit tests
make test

# Format code
make fmt

# Build all packages
make build

# Build plugin binaries
make plugin

# Build Docker image
make docker
```

### Testing

**Unit Tests:**
```bash
go test ./...
# or
make test
```

The Makefile version isolates `GOCACHE` so runs are self-contained.

**No Integration Tests:**
The repo does not depend on external services—tests only exercise local logic and the seeded data sets.

### Project Structure

```
opsorch-mock-adapters/
├── alertmock/        # Alert provider
├── incidentmock/     # Incident provider + timelines
├── logmock/          # Log generator
├── metricmock/       # Metric generator and descriptors
├── ticketmock/       # Ticket provider
├── messagingmock/    # Messaging provider
├── servicemock/      # Service catalog
├── secretmock/       # Secret store
├── internal/
│   ├── mockutil/     # Shared helpers + alert store
│   └── pluginrpc/    # JSON RPC harness for plugins
├── cmd/              # One plugin entrypoint per capability
├── Makefile
├── Dockerfile
└── go.mod            # go 1.22, depends on github.com/opsorch/opsorch-core
```

**Key Components:**

- **internal/mockutil**: Shared helpers for cloning maps, mapping services to teams/channels, alert metadata enrichment, and a lightweight alert store used by log and metric providers
- **internal/pluginrpc**: Tiny JSON-over-stdio harness that all `cmd/*plugin` binaries use; lazily creates provider instances and dispatches methods like `alert.query` or `incident.timeline.append`
- **Scenario fixtures**: Static Go slices in each provider (no runtime engine)

## Shared Utilities

### Alert Store (`internal/mockutil`)

The alert store keeps a snapshot of active alerts so metrics and logs can correlate with them:

- `PublishAlerts(alerts)`: Updates the alert snapshot
- `GetActiveAlerts()`: Returns current alert snapshot
- Used by log and metric providers to adjust generated data based on active alerts

### Service Mapping (`internal/mockutil`)

Helper functions for mapping services to teams and Slack channels:

- `GetTeamForService(service)`: Returns team name for a service
- `GetSlackChannelForService(service)`: Returns Slack channel for a service

## Plugin RPC Contract

OpsOrch Core communicates with plugins over stdin/stdout using JSON-RPC.

### Message Format

**Request:**
```json
{
  "method": "{capability}.{operation}",
  "config": { /* configuration */ },
  "payload": { /* method-specific body */ }
}
```

**Response:**
```json
{
  "result": { /* method-specific result */ },
  "error": "optional error message"
}
```

### Supported Methods

Each plugin supports the standard methods for its capability:

- **Alert Plugin**: `alert.query`, `alert.get`
- **Incident Plugin**: `incident.query`, `incident.get`, `incident.create`, `incident.update`, `incident.timeline.get`, `incident.timeline.append`
- **Log Plugin**: `log.query`
- **Metric Plugin**: `metric.query`, `metric.describe`
- **Ticket Plugin**: `ticket.query`, `ticket.get`, `ticket.create`, `ticket.update`
- **Messaging Plugin**: `messaging.send`
- **Service Plugin**: `service.query`
- **Secret Plugin**: `secret.get`, `secret.put`

## Use Cases

### Demos and Presentations

Mock adapters are perfect for demos because:
- No external dependencies or credentials required
- Deterministic data makes demos reproducible
- Scenario data demonstrates realistic failure scenarios
- Fast response times (no network calls)

### Development and Testing

Use mock adapters during development:
- Test OpsOrch Core without external services
- Develop UI components with realistic data
- Test correlation logic across capabilities
- Validate query filtering and scoping

### CI/CD Pipelines

Mock adapters enable testing in CI:
- No need to provision external services
- Fast test execution
- Consistent test data
- No flaky tests from external service issues

## Differences from Production Adapters

Mock adapters differ from production adapters in several ways:

1. **In-Memory Only**: All data is stored in memory, not persisted
2. **Deterministic**: Data generation is predictable and reproducible
3. **No External Calls**: No network requests to external systems
4. **Simplified Logic**: Filtering and querying happen in-process
5. **Scenario Data**: Pre-built scenario data for demos
6. **No Authentication**: No credentials or authentication required

## License

Apache 2.0

See LICENSE file for details.
