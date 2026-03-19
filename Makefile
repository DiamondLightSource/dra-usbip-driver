.PHONY: agent
agent:
	@$(MAKE) agent-$(shell go env GOARCH)

agent-%: clean-agent
	CGO_ENABLED=0 GOARCH=$* go build -o agent ./cmd/agent

clean-agent:
	rm -f agent

