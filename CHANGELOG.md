# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-04-14

### Added

- GitFlow-aware semantic version generation from GitHub Actions environment variables.
- Branch mapping: `main`/`master` → `rc`, `develop` → `dev`, `feat/*` → minor bump + `feat`, `fix/*`/`bugfix/*` → `fix`, `hotfix/*` → `hotfix`, `release/*` → `beta`, other branches → `branch`.
- Tag support: clean version output with no pre-release suffix when running on a tag.
- Commit count since latest semver tag as the pre-release numeric component.
- `GIT_SEM_VER_BUMP` environment variable to override the bump type (`major`, `minor`, `patch`).
- Branch slug truncated to 20 characters, slugified to alphanumeric and hyphens.
- Pure Go git implementation via go-git (no git CLI dependency).
- GitHub Actions composite action with pre-built binary download.
- Multi-platform releases: Linux, macOS, Windows × amd64/arm64.
- CI pipeline: tests on Go 1.25, 1.26, stable with race detection.
- Lint pipeline: golangci-lint v2 with strict configuration.
- Security pipeline: Trivy Go module vulnerability scanning.

[Unreleased]: https://github.com/Vr00mm/git-sem-ver/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/Vr00mm/git-sem-ver/releases/tag/v1.0.0
