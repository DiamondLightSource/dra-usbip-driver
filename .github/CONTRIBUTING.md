# Contribute to the project

Contributions and issues are most welcome! All issues and pull requests are
handled through [GitHub](https://github.com/DiamondLightSource/dra-usbip-driver/issues).
Please check for any existing issues before filing a new one. If you have a
great idea but it involves big changes, please file a ticket before making a
pull request!

## Issue or Discussion?

GitHub also offers [discussions](https://github.com/DiamondLightSource/dra-usbip-driver/discussions)
as a place to ask questions and share ideas. If your issue is open ended and it
is not obvious when it can be "closed", please raise it as a discussion instead.

## Developer Information

It is recommended that developers use a
[VSCode devcontainer](https://code.visualstudio.com/docs/devcontainers/containers).
This repository contains configuration to set up a containerized development
environment with all required Go tools.

### Prerequisites (without devcontainer)

- Go 1.24+
- [golangci-lint](https://golangci-lint.run/)

### Running tests

```bash
go test ./...
```

### Running the linter

```bash
golangci-lint run
```

### Building

```bash
make agent
go build ./cmd/manager
go build ./cmd/plugin
```
