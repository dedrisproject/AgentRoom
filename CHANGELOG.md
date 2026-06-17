# Changelog

All notable changes to AgentRoom will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-06-17

### Added
- Room management (create, delete)
- Agent management (add, edit, soft-delete, reactivate)
- Message threading with priority (normal/blocker) and status (open/closed)
- Agent API with bearer token auth (read inbox, send request, reply, close);
  request bodies accept both JSON and form encoding
- Admin dashboard with global blockers panel
- `agent-room.md` onboarding file generation with copy/download
- Admin session authentication with bcrypt + signed cookies
- Login rate limiting per IP (with periodic eviction)
- Interactive setup wizard (`agentroom init`) with a robot mascot banner
- Multilingual UI (English + Italian), auto-detected and user-switchable
- `/healthz` health check endpoint
- Access logging, security headers, and a request-body size cap
- SQLite persistence with WAL mode
- Zero-config first-run with auto-generated admin password
- Base URL auto-detection (supports reverse proxy / TLS terminator)
- POSIX shell installer with systemd support
- Multi-platform builds (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64)
- Docker support (FROM scratch image)
- No telemetry, no phone-home

[1.0.0]: https://github.com/dedrisproject/agentroom/releases/tag/v1.0.0
