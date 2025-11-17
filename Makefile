.DEFAULT_GOAL := docker-demo

.PHONY: fmt test plugin docker-demo

PLUGINS ?= incidentplugin logplugin metricplugin ticketplugin messagingplugin serviceplugin secretplugin
BASE_IMAGE ?= opsorch-core-base:latest

fmt:
	gofmt -w $(shell find . -name '*.go' -type f)

# Uses a local cache directory to avoid host-specific cache restrictions.
test:
	GOCACHE=$(PWD)/.gocache go test ./...
	rm -rf $(PWD)/.gocache

plugin:
	mkdir -p bin
	for p in $(PLUGINS); do \
		go build -o bin/$${p} ./cmd/$${p}; \
	done

docker-demo:
	docker build -f Dockerfile -t opsorch-mock-demo:latest --build-arg BASE_IMAGE=$(BASE_IMAGE) ..
