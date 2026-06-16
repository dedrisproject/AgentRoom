# Contributing to AgentRoom

Thank you for your interest in contributing to AgentRoom!

## Getting Started

### Prerequisites

- Go 1.22 or later
- Git

### Local Setup

```bash
git clone https://github.com/dedrisproject/agentroom.git
cd agentroom
go mod download
go build ./cmd/agentroom
```

Run the setup wizard:

```bash
./agentroom init
```

## How to Contribute

### Reporting Bugs

Open an issue using the **Bug Report** template. Include:
- Steps to reproduce
- Expected vs actual behavior
- OS, architecture, and AgentRoom version (`agentroom --version`)

### Suggesting Features

Open an issue using the **Feature Request** template. Describe the use case and why it would benefit other users.

### Submitting Pull Requests

1. Fork the repository and create a branch from `master`
2. Make your changes
3. Ensure the code compiles: `go build ./...`
4. Run tests if applicable: `go test ./...`
5. Open a pull request with a clear description of what was changed and why

### Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Keep changes focused — one concern per PR
- No new external runtime dependencies without prior discussion

## Project Structure

```
cmd/agentroom/      # Entry point and CLI (init wizard, main)
internal/
  api/              # HTTP handlers (admin + agent APIs)
  auth/             # Session and token middleware
  config/           # Config file load/save
  db/               # SQLite open and migrations
  i18n/             # Translations (EN, IT)
  instructions/     # agent-room.md generator
  store/            # Data access layer
  web/              # Embedded templates and static assets
install.sh          # POSIX installer
.goreleaser.yaml    # Multi-platform release builds
```

## Translations

Translation strings live in `internal/i18n/translations.go`. To add a new language:

1. Add the language code to `SupportedLangs` in `internal/i18n/i18n.go`
2. Add a translation map entry in `translations.go`
3. Open a PR with the new language

## Release Process

Releases are handled automatically via GoReleaser on tag push:

```bash
git tag v1.x.x
git push origin v1.x.x
```
