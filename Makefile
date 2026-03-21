.PHONY: agent
agent:
	@$(MAKE) agent-$(shell go env GOARCH)

agent-%: clean-agent
	CGO_ENABLED=0 GOARCH=$* go build -o agent ./cmd/agent

clean-agent:
	rm -f agent

.PHONY: pico-mac-sender
pico-mac-sender:
	@$(MAKE) pico-mac-sender-$(shell go env GOARCH)

pico-mac-sender-%: clean-pico-mac-sender
	CGO_ENABLED=0 GOARCH=$* go build -o pico-mac-sender ./cmd/pico-mac-sender

clean-pico-mac-sender:
	rm -f pico-mac-sender

.PHONY: build lint test docs docs-live ci

build:
	go build ./cmd/manager
	go build ./cmd/plugin
	go build ./cmd/pico-mac-sender

lint:
	golangci-lint run

test:
	go test -v -race ./...

docs:
	uv run sphinx-build -W --keep-going docs docs/_build/html

docs-live:
	uv run sphinx-autobuild docs docs/_build/html

ci:
	$(MAKE) -j lint test docs

