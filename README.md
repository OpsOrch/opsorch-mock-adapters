# OpsOrch Mock Adapters

Mock implementations of the OpsOrch provider interfaces that keep everything in-memory and return deterministic, "realistic enough" demo data. The packages under `*mock/` can be imported directly into an OpsOrch Core binary or compiled into the standalone plugin binaries shipped with this repo.

These adapters intentionally stay simple: data is seeded from Go structs, scoped filtering happens in-process, and a handful of scenario-labeled records make it easy to demonstrate cascading failures without talking to any real systems.

## Using the mock adapters

### Embed directly inside OpsOrch Core

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

### Build plugin binaries

Every folder in `cmd/*plugin` exposes the same providers over OpsOrch’s plugin RPC (stdin/stdout JSON). Build all binaries with:

```bash
make plugin         # outputs bin/<capability>plugin
```

Set `PLUGINS="alertplugin metricplugin"` to limit the build. Wire a plugin into OpsOrch Core via `OPSORCH_<CAPABILITY>_PLUGIN=/full/path/to/bin/<capability>plugin`.

### Demo Docker image

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

## Provider behavior

### Alerts (`alertmock`)
- Seeds ~30 alerts that cover checkout, search, database, payments, ML, and infra services with critical/error/warning/info severities.
- `Query` supports free-text search, scope filters (service/environment/team), severity/status filters, limits, and context-aware scope merging via `alertmock.WithScope`.
- Alerts are enriched with runbooks, dashboards, escalation policies, Slack channels, deployment context, multi-region hints, and user impact metadata.
- A few IDs participate in scripted lifecycle plans so statuses/severities change over time (firing → acknowledged → resolved) without external writes.
- `mockutil.PublishAlerts` keeps a snapshot so metrics and logs can correlate active alerts.
- Config: `source` (stamps `Metadata["source"]`, defaults to `mock-alert`).

### Incidents (`incidentmock`)
- Seeds an in-memory list of incidents plus timelines; six extra incidents are tagged with scenario metadata (`scenario_id`, `scenario_name`, `Metadata["is_scenario"]`).
- `Query` and `List` filter by scope, severity, status, and search terms. Results inherit any `WithScope` context.
- Supports `Get`, `Create`, `Update`, `GetTimeline`, and `AppendTimeline`. `Create` infers the service from fields when missing and stamps `Metadata["source"]`.
- Config: `source` and `defaultSeverity` (defaults to `sev2`).

### Logs (`logmock`)
- Generates synthetic log entries inside the requested time window using the query search string, expression filters, and severity hints.
- Applies caller-provided `LogFilter`s to every generated entry so queries can simulate equality/inequality/contains matching.
- Incorporates alert snapshots: if a matching alert is active during the requested window, the log severity, latency, and metadata are adjusted and the alert summary is embedded.
- Emits additional static scenario logs flagged by `Fields["is_scenario"]` that describe cascading failures, rollbacks, and dependency outages.
- Config: `defaultLimit` (default 50) and `source` for metadata.

### Metrics (`metricmock`)
- Describes 40+ metric definitions (counters, gauges, histograms) and builds deterministic waveforms with daily patterns, noise, and growth trends.
- `Query` returns an "active" series plus a computed baseline clone for each requested descriptor. Metadata calls out service, query window, sampling step, alert summaries, and `variant`.
- Static scenario anomalies (IDs `scenario-001`–`scenario-006`) inject spikes, drops, or plateaus over specific services/metrics and surface `scenario_effects` metadata with the window that was modified.
- `Describe` returns the full metric catalog so UIs can populate dropdowns without hardcoding names.
- Config: `source` for metadata annotations.

### Tickets (`ticketmock`)
- Maintains an in-memory ticket store (seeded work items + queried scenario tickets with `Fields["is_scenario"]`).
- `Query` filters by service/team/status/priority, `Get` retrieves individual tickets, `Create`/`Update` mutate records for demos.
- Every ticket automatically gains helpful metadata such as runbook links, checklists, dependency hints, due dates, and `Metadata["relatedIncidents"]`.
- Config: `source` (stamped on `Metadata["source"]`).

### Messaging (`messagingmock`)
- `Send` simulates message delivery, records the request in-memory, and returns latency, retry history, throttling info, and failure reasons in `Metadata`.
- Delivery characteristics vary with the channel (chat/email/SMS/etc.) so playbooks can demonstrate success, delayed delivery, and permanent failures.
- `History()` exposes previously sent messages for demos/tests.
- Config: `provider` (defaults to `mock`) so results can look like `mock-sms-0001`, etc.

### Services (`servicemock`)
- Serves a static service catalog that covers frontend, backend, and data tiers. Each service includes tags (`env`, `tier`, `owner`) and metadata (runbooks, dashboards, repos, on-call info).
- `Query` supports filtering by IDs, name substrings, tags, and query scope.
- Config: `environment` which stamps the `env` tag for every record (default `prod`).

### Secrets (`secretmock`)
- Extremely small key/value secret store meant for demos. Defaults include DB passwords, Stripe keys, Slack webhooks, etc.
- Supports `Get` and `Put` so flows that need to rotate a secret can still succeed locally.
- Config: `secrets` map for pre-seeding custom values.

## Shared utilities & scenario data

- `internal/mockutil` hosts helpers for cloning maps, mapping services to teams/channels, alert metadata enrichment, and a lightweight alert store used by the log and metric providers.
- `internal/pluginrpc` implements the tiny JSON-over-stdio harness that all `cmd/*plugin` binaries use; every command lazily creates a provider instance and dispatches methods like `alert.query` or `incident.timeline.append`.
- Scenario fixtures are implemented as static Go slices inside each provider (e.g. `getScenarioLogs`, `getScenarioMetricAnomalies`, `getScenarioTickets`). There is no runtime scenario engine or toggle—scenario records are always available and can be identified via `Metadata["is_scenario"]`/`Fields["scenario_*"]`.

## Repository layout

```
.
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
└── go.mod            # go 1.22, depends on github.com/opsorch/opsorch-core v0.0.4
```

## Development

- Requires Go 1.22+.
- `go test ./...` or `make test` runs the suite; the Makefile version isolates `GOCACHE` so runs are self-contained.
- `make fmt` runs `gofmt -w` across every Go file.
- `make plugin` and `make docker` are the supported build targets for binaries and container images respectively.
- The repo does not depend on external services—tests only exercise local logic and the seeded data sets.
