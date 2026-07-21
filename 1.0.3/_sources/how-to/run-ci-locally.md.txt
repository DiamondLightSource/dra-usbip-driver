# Run CI checks locally

You can run the same checks that CI runs before pushing, to catch issues early.

## Run all checks in parallel

To run linting, tests, and docs build simultaneously (like `tox -p`):

```bash
$ make ci
```

## Run checks individually

### Linting

```bash
$ make lint
```

This runs [golangci-lint](https://golangci-lint.run/) against the codebase.

### Tests

```bash
$ make test
```

This runs all Go tests with the race detector enabled.

### Documentation

```bash
$ make docs
```

This builds the Sphinx documentation with warnings treated as errors, matching
the CI configuration. See {doc}`/how-to/run-sphinx` for more options including
live preview.
