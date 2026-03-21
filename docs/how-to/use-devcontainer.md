# Use the devcontainer

The repository includes a [VSCode devcontainer](https://code.visualstudio.com/docs/devcontainers/containers)
that provides a ready-to-use development environment.

## What is included

The devcontainer is designed for use with **Podman** or other rootless container
runtimes. It runs as `root` inside the container, which is the standard approach
when the container runtime itself is unprivileged (rootless). This avoids
permission issues with mounted volumes and build tools.

The devcontainer is based on the Microsoft Go 1.24 image and comes with:

- **Go 1.24** and **golangci-lint** for building and linting the project
- **uv** for Python dependency management
- **Sphinx** and all documentation dependencies, installed automatically on
  container creation

No additional setup is needed after opening the project in the devcontainer.

## Getting started

1. Install the
   [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
   in VSCode.
2. Open this repository in VSCode.
3. When prompted, click **Reopen in Container** — or run the
   **Dev Containers: Reopen in Container** command from the command palette.
4. Wait for the container to build and the post-create setup to finish.

Once the container is ready you can immediately:

- `make build` — build the manager and plugin
- `make agent` — cross-compile the agent
- `make ci` — run linting, tests, and docs build in parallel
- `make docs` / `make docs-live` — build or live-preview the documentation

See {doc}`/how-to/run-ci-locally` and {doc}`/how-to/run-sphinx` for more details.
