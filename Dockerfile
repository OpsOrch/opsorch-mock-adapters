# syntax=docker/dockerfile:1
# Build a demo-ready OpsOrch image with all mock adapters bundled as plugins.
# Demo image now layers plugins on the opsorch-core-base image, which already
# contains the core binary. Build context must include both opsorch-core and
# opsorch-mock-adapters (e.g. run from repo root with -f opsorch-mock-adapters/Dockerfile . or ..).

ARG GO_VERSION=1.22
ARG BASE_IMAGE=opsorch-core-base:latest
ARG PLUGINS=""

FROM golang:${GO_VERSION} AS plugin-builder
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG PLUGINS
SHELL ["/bin/bash", "-c"]
WORKDIR /src

# Expect both repos in the build context (sibling directories under the context root).
COPY opsorch-core /src/opsorch-core
COPY opsorch-mock-adapters /src/opsorch-mock-adapters

ENV CGO_ENABLED=0

# Build all mock plugin binaries. If PLUGINS is empty, build every cmd/* entry.
WORKDIR /src/opsorch-mock-adapters
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build <<'EOF'
set -euo pipefail
mkdir -p /out/plugins
if [[ -n "${PLUGINS}" ]]; then
  set -- ${PLUGINS}
else
  set -- cmd/*
fi
for p in "$@"; do
  name=$(basename "$p")
  GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /out/plugins/"$name" "./cmd/$name"
done
EOF

FROM ${BASE_IMAGE}
WORKDIR /opt/opsorch

COPY --from=plugin-builder /out/plugins ./plugins

# Default to mock plugins for every capability; users can override at runtime.
ENV \
  OPSORCH_INCIDENT_PLUGIN=/opt/opsorch/plugins/incidentplugin \
  OPSORCH_LOG_PLUGIN=/opt/opsorch/plugins/logplugin \
  OPSORCH_METRIC_PLUGIN=/opt/opsorch/plugins/metricplugin \
  OPSORCH_TICKET_PLUGIN=/opt/opsorch/plugins/ticketplugin \
  OPSORCH_MESSAGING_PLUGIN=/opt/opsorch/plugins/messagingplugin \
  OPSORCH_SERVICE_PLUGIN=/opt/opsorch/plugins/serviceplugin \
  OPSORCH_SECRET_PLUGIN=/opt/opsorch/plugins/secretplugin \
  OPSORCH_BEARER_TOKEN=demo
 