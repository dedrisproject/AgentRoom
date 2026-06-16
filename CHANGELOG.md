# Changelog

All notable changes to AgentRoom will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of AgentRoom
- Room management (create, delete)
- Agent management (add, edit, soft-delete, reactivate)
- Message threading with priority (normal/blocker) and status (open/closed)
- Agent API with bearer token auth (read inbox, send request, reply, close)
- Admin dashboard with global blockers panel
- `agent-room.md` onboarding file generation with copy/download
- Admin session authentication with bcrypt + signed cookies
- Login rate limiting per IP
- SQLite persistence with WAL mode
- Zero-config first-run with auto-generated admin password
- Base URL auto-detection (supports reverse proxy / TLS terminator)
- POSIX shell installer with systemd support
- Multi-platform builds (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64)
- Docker support (FROM scratch image)
- No telemetry, no phone-home
