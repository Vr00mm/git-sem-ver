# git-sem-ver

[![Go Version](https://img.shields.io/github/go-mod/go-version/Vr00mm/git-sem-ver)](https://go.dev/)
[![License](https://img.shields.io/github/license/Vr00mm/git-sem-ver)](./LICENSE)
[![CI](https://img.shields.io/github/actions/workflow/status/Vr00mm/git-sem-ver/ci.yml?branch=main&label=CI)](https://github.com/Vr00mm/git-sem-ver/actions/workflows/ci.yml)
[![Security](https://img.shields.io/github/actions/workflow/status/Vr00mm/git-sem-ver/security.yml?branch=main&label=security)](https://github.com/Vr00mm/git-sem-ver/actions/workflows/security.yml)

A GitHub Actions tool that generates semantic versions from GitFlow branch conventions. It reads the latest git tag reachable from HEAD and produces a pre-release version string that encodes the branch type and commit distance.

## Demo

```yaml
- uses: Vr00mm/git-sem-ver@v1
  id: version

- run: echo "Version is ${{ steps.version.outputs.version }}"
```

Example outputs:

| Branch | Latest tag | Commits since | Output |
|---|---|---|---|
| `main` | `v1.2.3` | 4 | `1.3.0-rc.4` |
| `develop` | `v1.2.3` | 7 | `1.2.4-dev.7` |
| `feat/new-api` | `v1.2.3` | 2 | `1.3.0-feat.new-api.2` |
| `fix/login-bug` | `v1.2.3` | 1 | `1.2.4-fix.login-bug.1` |
| `hotfix/crash` | `v1.2.3` | 1 | `1.2.4-hotfix.crash.1` |
| `release/1.3` | `v1.2.3` | 3 | `1.2.4-beta.3` |
| `v1.3.0` (tag) | — | — | `1.3.0` |

## Getting Started

### As a GitHub Action

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
        with:
          fetch-depth: 0  # required: full history for tag discovery

      - uses: Vr00mm/git-sem-ver@v1
        id: version

      - run: echo "Building version ${{ steps.version.outputs.version }}"
```

> **Important:** `fetch-depth: 0` is required. A shallow clone has no tags, which causes git-sem-ver to count from the beginning of the truncated history.

### As a binary

Download the latest release for your platform from the [releases page](https://github.com/Vr00mm/git-sem-ver/releases), or install from source:

```bash
go install github.com/Vr00mm/git-sem-ver/cmd/git-sem-ver@latest
```

Then set the required environment variables and run:

```bash
export GITHUB_REF_TYPE=branch
export GITHUB_REF_NAME=feat/my-feature
git-sem-ver
# → 0.1.0-feat.my-feature.3
```

## Features

### Branch mapping

| Branch pattern | Bump | Pre-release format |
|---|---|---|
| `main`, `master` | patch | `rc.<n>` |
| `develop`, `development` | patch | `dev.<n>` |
| `feat/*`, `feature/*` | **minor** | `feat.<slug>.<n>` |
| `fix/*`, `bugfix/*` | patch | `fix.<slug>.<n>` |
| `hotfix/*` | patch | `hotfix.<slug>.<n>` |
| `release/*` | patch | `beta.<n>` |
| tag | — | *(clean version, no pre-release)* |
| anything else | patch | `branch.<slug>.<n>` |

`<n>` is the number of commits since the latest semver tag. `<slug>` is the branch suffix lowercased, with non-alphanumeric characters replaced by hyphens, truncated to 20 characters.

### Environment variables

| Variable | Required | Description |
|---|---|---|
| `GITHUB_REF_TYPE` | Yes | `branch` or `tag` (set automatically by GitHub Actions) |
| `GITHUB_REF_NAME` | Yes | Branch or tag name (set automatically by GitHub Actions) |
| `GITHUB_SHA` | No | Current commit SHA (informational) |
| `GIT_SEM_VER_BUMP` | No | Override bump type: `major`, `minor`, or `patch` |
| `GITHUB_OUTPUT` | No | Path to GitHub Actions output file (set automatically) |

### Forcing a bump type

Use `GIT_SEM_VER_BUMP` to override the default bump strategy when introducing breaking changes on a non-feature branch:

```yaml
- uses: Vr00mm/git-sem-ver@v1
  id: version
  with:
    bump: major
```

### No tags yet?

When no semver tag exists in the repository, git-sem-ver uses `0.0.0` as the base and counts all commits from the beginning of history.

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md).

## License

[MIT](./LICENSE)
