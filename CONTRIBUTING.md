# Contributing

## Prerequisites

- Go 1.25 or later
- Git

## Setup

```bash
git clone https://github.com/Vr00mm/git-sem-ver.git
cd git-sem-ver
go mod download
```

## Build

```bash
make build
# or
go build ./cmd/git-sem-ver
```

## Test

```bash
make test
# or
go test -race -shuffle=on ./...
```

## Lint

```bash
make lint
# or
golangci-lint run ./...
```

## Makefile targets

```
make build    — compile the binary
make test     — run tests with race detection
make lint     — run golangci-lint
make fmt      — format code
make clean    — remove build artifacts
```

## Pull request process

1. Fork the repository and create a branch from `develop` following GitFlow conventions:
   - `feat/<description>` for new features
   - `fix/<description>` for bug fixes
   - `hotfix/<description>` for critical production fixes
2. Write tests for any new behaviour.
3. Ensure `make test` and `make lint` pass.
4. Open a pull request against `develop`.

## Commit style

Use conventional commits: `feat:`, `fix:`, `docs:`, `ci:`, `refactor:`, `test:`.
