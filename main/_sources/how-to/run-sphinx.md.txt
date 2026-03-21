# Build the docs locally

This guide explains how to build and preview the project documentation locally
using Sphinx. Makefile targets are provided so you don't need to remember the
full commands.

## Prerequisites

You need [uv](https://docs.astral.sh/uv/) installed. If you are using the
devcontainer, everything is already set up.

## Install dependencies

Install the docs dependency group:

```bash
$ uv sync --group docs
```

## Build the documentation

Run a one-off build (warnings are treated as errors, matching CI):

```bash
$ make docs
```

The output is written to `docs/_build/html`.

## Live preview

For a live-reloading server that rebuilds on changes:

```bash
$ make docs-live
```

Then open the URL shown in the terminal (typically `http://127.0.0.1:8000`).

If port 8000 is already in use, specify a different port:

```bash
$ uv run sphinx-autobuild docs docs/_build/html --port 8001
```
