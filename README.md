# OpsOrch Mock Adapters

In-process adapters that provide realistic, self-contained demo data for OpsOrch capabilities: incidents, logs, metrics, tickets, messaging, services, and secrets.

The providers register themselves under the name `mock` for their respective capability registries. They keep data in-memory and expose helpful defaults so the API is interactive without touching real systems.

## Quick start with OpsOrch Core

Add the adapters to your Core binary and import them for side effects:

```go
import (
    _ "github.com/opsorch/opsorch-mock-adapters/incidentmock"
    _ "github.com/opsorch/opsorch-mock-adapters/logmock"
    _ "github.com/opsorch/opsorch-mock-adapters/metricmock"
    _ "github.com/opsorch/opsorch-mock-adapters/ticketmock"
    _ "github.com/opsorch/opsorch-mock-adapters/messagingmock"
    _ "github.com/opsorch/opsorch-mock-adapters/servicemock"
    _ "github.com/opsorch/opsorch-mock-adapters/secretmock"
)
```

Start Core with the mock providers enabled (adjust config as desired):

```bash
OPSORCH_INCIDENT_PROVIDER=mock \
OPSORCH_LOG_PROVIDER=mock \
OPSORCH_METRIC_PROVIDER=mock \
OPSORCH_TICKET_PROVIDER=mock \
OPSORCH_MESSAGING_PROVIDER=mock \
OPSORCH_SERVICE_PROVIDER=mock \
OPSORCH_SECRET_PROVIDER=mock \
go run ./cmd/opsorch
```

### Plugin binaries

If you prefer plugin loading, build the binaries:

```bash
make plugin
```

This produces `bin/incidentplugin`, `bin/logplugin`, `bin/metricplugin`, `bin/ticketplugin`, `bin/messagingplugin`, `bin/serviceplugin`, and `bin/secretplugin`.
Point Core at them via `OPSORCH_<CAP>_PLUGIN=/full/path/to/bin/<name>`.

## Mock behaviors

- **Incident**: seeded checkout/search incidents with timelines; supports create/update/timeline append; metadata stamped with `source` config.
- **Log**: generates query-aware log lines within the requested window; infers severity/service from the query and returns structured fields.
- **Metric**: builds deterministic waveforms for the requested expression, plus a "baseline" comparison series.
- **Ticket**: seeded work items with assignees; supports create/update flows.
- **Messaging**: records sent messages in-memory and returns delivery metadata.
- **Service**: filters static services by id/name/tags with optional environment tagging via config.
- **Secret**: in-memory key/value store with optional seed data.

## Development

Run the test suite:

```bash
go test ./...
```

## Demo Docker image

Build a demo-ready image that layers the mock plugins onto `opsorch-core-base:latest` (make sure that base image is available locally via `docker pull` or `make docker-build-base` from the `opsorch-core` repo). The build context must include both this repo and `opsorch-core` as siblings (run from the parent directory):

```bash
docker build -f opsorch-mock-adapters/Dockerfile -t opsorch-mock-demo:latest ..
```

Or use the helper target from this repo:

```bash
make docker-demo
```

Run the demo image (defaults wire every capability to the bundled mock plugin binaries):

```bash
docker run --rm -p 8080:8080 opsorch-mock-demo:latest
```

Override any capability by setting the corresponding `OPSORCH_<CAP>_PLUGIN` env to a different plugin path at runtime.

The module pulls `opsorch-core` from `../opsorch-core` via a local replace for development convenience.
