.PHONY: agent
agent:
	@$(MAKE) agent-$(shell go env GOARCH)

agent-%: clean-agent
	CGO_ENABLED=0 GOARCH=$* go build -o agent ./cmd/agent

clean-agent:
	rm -f agent

.PHONY: build lint test docs docs-live ci

build:
	go build ./cmd/manager
	go build ./cmd/plugin

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

