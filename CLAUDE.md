# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
go build -o endfield-daily .   # build binary
go vet ./...                   # lint
```

No test suite exists. The project uses Go 1.26 (see `.tool-versions`).

Docker build (multi-arch via CI, or locally):
```bash
docker build -t endfield-daily .
```

## Architecture

Automated daily check-in client for Arknights: Endfield via the SKPort API. The binary has two modes: one-shot (default) and scheduled (when `SCHEDULE` env var is set with a cron expression).

**Packages:**

- `main.go` — Entry point. Parses env vars, runs one-shot or enters cron scheduling loop using `adhocore/gronx`. Handles SIGINT/SIGTERM for graceful shutdown.
- `skport/` — SKPort API client. Three-step OAuth flow (`auth.go`), request signing via HMAC-SHA256+MD5 (`sign.go`), role fetching and check-in (`checkin.go`), API response types (`types.go`).
- `notify/` — Notification system. `Notifier` interface with Discord webhook implementation. `MessageLog` collects results across accounts for batch notification.

**Flow:** Parse tokens → authenticate each account via Gryphline OAuth → fetch player roles → check in each role → optionally notify via Discord.

## Environment Variables

- `ACCOUNT_TOKEN` (required) — newline-separated account tokens
- `SCHEDULE` — cron expression for recurring runs; omit for one-shot
- `DISCORD_WEBHOOK` / `DISCORD_USER` — optional Discord notifications (sent per check-in attempt)
- `NOTIFY_NO_OPS` — if set, also notify when already checked in (by default only notifies on success/error)

## Docker

Final image is Alpine-based (not distroless) so `docker exec <container> /endfield-daily` works for manual triggers alongside a scheduled container. CI publishes multi-arch (amd64 + arm64) images to GHCR on version tags.
