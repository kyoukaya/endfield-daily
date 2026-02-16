# endfield-daily

Automated daily check-in for Arknights: Endfield via the SKPort API. Runs once and exits, or loops on a cron schedule. Supports multiple accounts and optional Discord webhook notifications.

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `ACCOUNT_TOKEN` | yes | One or more account tokens, separated by newlines. |
| `SCHEDULE` | no | Cron expression (e.g. `0 4 * * *`). If set, the binary runs immediately then repeats on schedule. If unset, the binary runs once and exits. |
| `DISCORD_WEBHOOK` | no | Discord webhook URL for notifications. |
| `DISCORD_USER` | no | Discord user ID to mention in notifications. |

## Setup

### Docker

Run once:

```
docker run --rm -e ACCOUNT_TOKEN=<token> ghcr.io/kyoukaya/endfield-daily:latest
```

Run on a schedule:

```
docker run -d --restart unless-stopped \
  -e ACCOUNT_TOKEN=<token> \
  -e SCHEDULE="0 4 * * *" \
  ghcr.io/kyoukaya/endfield-daily:latest
```

Trigger a manual run while the scheduled container is running:

```
docker exec <container> /endfield-daily
```

### Docker Compose

Create a `.env` file:

```
ACCOUNT_TOKEN=<token>
```

Then start the service:

```
docker compose up -d
```

The default compose configuration schedules runs daily at 04:00 UTC. Edit `SCHEDULE` in `docker-compose.yml` to change this.
