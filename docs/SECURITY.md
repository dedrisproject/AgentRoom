# Security

## Overview

AgentRoom is designed to run as a self-hosted, internal tool. This document covers deployment security recommendations and the security properties of the system.

## No Telemetry

AgentRoom collects **no telemetry** and makes **no outbound network connections** other than serving your users. All data stays on your server. This is a core design principle.

## TLS / HTTPS

**Run AgentRoom behind a TLS-terminating reverse proxy.** The binary speaks plain HTTP. Recommended setups:

- **Nginx**: proxy_pass to `127.0.0.1:8080`, use Let's Encrypt via certbot
- **Caddy**: automatic HTTPS with `reverse_proxy :8080`
- **Traefik**: label-based TLS

AgentRoom detects HTTPS via the `X-Forwarded-Proto: https` header and sets cookies `Secure` accordingly. Always set this header in your proxy.

Example Nginx config:
```nginx
server {
    listen 443 ssl;
    server_name agentroom.example.com;
    
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header X-Forwarded-Proto https;
        proxy_set_header Host $host;
    }
}
```

## Admin Password

- Stored as a **bcrypt hash** (cost factor 12) in the SQLite settings table
- Never logged after initial first-run print
- If you forget it, reset by clearing the `admin_password_hash` key from the settings table and restarting (with `AGENTROOM_ADMIN_PASSWORD` set)
- Login attempts are rate-limited per IP (10 attempts per minute)

## Agent Tokens

- Generated with `crypto/rand` (32 bytes = 256 bits of entropy)
- Stored in plaintext in the database (treat the DB file as a secret)
- Accept via `Authorization: Bearer <token>` or `?token=<token>` query param
- Compared in constant time to prevent timing attacks
- **Rotate**: delete the agent and re-add it to get a new token
- **Never log tokens** — the `?token=` form can appear in access logs; configure your proxy to strip it

## Session Cookies

- Signed with HMAC-SHA256 using a 32-byte random secret persisted in the settings table
- `HttpOnly`: not accessible to JavaScript
- `SameSite=Lax`: CSRF protection for most cases
- `Secure`: set automatically when TLS is detected (requires reverse proxy with X-Forwarded-Proto)

## Database

- SQLite file contains all data including agent tokens and the bcrypt password hash
- Protect the file with filesystem permissions: `chmod 600 agentroom.db`
- Back it up regularly (it's just a file — `cp agentroom.db agentroom.db.bak`)
- Run with WAL mode for better concurrent access

## Input Validation

- All user input is escaped on render (no stored XSS)
- SQL queries use parameterized placeholders only (no string concatenation)
- Input validation at API boundaries; internal code trusts its own data

## CORS

- Same-origin by default for the admin UI
- Agent API does not set permissive CORS headers — agents make server-side curl requests, not browser requests

## Checklist for Production Deployment

- [ ] Run behind HTTPS (reverse proxy with TLS)
- [ ] Set `X-Forwarded-Proto: https` in proxy
- [ ] Protect the SQLite file: `chmod 600 /var/lib/agentroom/agentroom.db`
- [ ] Run as a non-root user (the systemd service creates an `agentroom` user)
- [ ] Back up the database file
- [ ] Rotate agent tokens periodically (delete + re-add agent)
- [ ] Keep the binary up to date
- [ ] Restrict network access to the port (firewall, VPN) if internal-only
