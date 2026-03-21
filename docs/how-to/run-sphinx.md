# Run Sphinx

This guide explains how to build and preview the project documentation locally using Sphinx.

## Prerequisites

You need [uv](https://docs.astral.sh/uv/) installed. If you are using the
devcontainer, everything is already set up.

## Install dependencies

Install the docs dependency group:

```bash
$ uv sync --group docs
```

## Build the documentation

Run a one-off build into `docs/_build/html`:

```bash
$ uv run sphinx-build docs docs/_build/html
```

To treat warnings as errors (as CI does):

```bash
$ uv run sphinx-build -W --keep-going docs docs/_build/html
```

## Live preview

For a live-reloading server that rebuilds on changes:

```bash
$ uv run sphinx-autobuild docs docs/_build/html
```

Then open the URL shown in the terminal (typically `http://127.0.0.1:8000`).

If port 8000 is already in use, specify a different port:

```bash
$ uv run sphinx-autobuild docs docs/_build/html --port 8001
```
