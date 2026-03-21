# Contribute to the project

Contributions and issues are most welcome! All issues and pull requests are
handled through [GitHub](https://github.com/DiamondLightSource/dra-usbip-driver/issues).

## Developer environment

It is recommended to use the [VSCode devcontainer](https://code.visualstudio.com/docs/devcontainers/containers).
This repository contains configuration to set up a containerized development
environment with all required Go tools.

Alternatively, ensure you have:

- Go 1.24+
- [golangci-lint](https://golangci-lint.run/)

## Running tests

```bash
go test ./...
```

## Running the linter

```bash
golangci-lint run
```

## Building

Build the manager and plugin:

```bash
$ make build
```

The agent cross-compiles for the target architecture (e.g. Raspberry Pi):

```bash
$ make agent
```
