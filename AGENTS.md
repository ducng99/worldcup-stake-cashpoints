# AGENTS.md

This file provides guidance to coding agents when working with code in this repository.

## Commands

### Backend (Go)
```bash
# Run with live reload (builds frontend first, watches .go/.tsx/.css/.html)
air

# Build and run manually
go build -o ./tmp/main . && ./tmp/main

# Set required env var for live match sync
export FOOTBALL_DATA_API_KEY=<your-key>
```

### Frontend (SolidJS/Vite)
```bash
cd frontend

# Dev server with hot reload (proxies /api to localhost:8080)
npm run dev

# Build (TypeScript check + Vite, outputs to frontend/dist/)
npm run build
```

The Go binary embeds `frontend/dist/` at build time via `//go:embed all:dist` (`frontend/embed.go`), so the frontend must be built before running Go in production. Static files (e.g. `robots.txt`) live in `frontend/public/` and are copied to `frontend/dist/` by Vite on build.

## Architecture

**Monorepo:** Go/Gin backend + SolidJS frontend. The frontend is compiled to `frontend/dist/` and embedded into the Go binary. In dev, Vite proxies `/api` requests to `:8080`.

**Backend flow:**
1. SQLite DB init + schema migration (`db/db.go`)
2. Seed 7 players and 48 FIFA WC 2026 teams if tables are empty (`db/seed.go`)
3. Background sync goroutine polls `https://api.football-data.org/v4/competitions/WC/matches` every 5 minutes (requires `FOOTBALL_DATA_API_KEY`)
4. Gin serves `/api/*` routes and falls back to `index.html` for SPA routing

**API routes:**
| Method | Path | Handler |
|--------|------|---------|
| GET | `/api/users` | Returns seeded players for subscription settings |
| GET | `/api/matches` | Returns matches + team→owner map |
| GET | `/api/leaderboard` | Returns players ranked by wins |
| GET | `/api/push/vapid-public-key` | Returns configured VAPID public key |
| POST | `/api/push/subscriptions` | Upserts a browser push subscription |
| PATCH | `/api/push/subscriptions/preferences` | Updates selected player/preferences for an endpoint |
| DELETE | `/api/push/subscriptions` | Deletes a browser push subscription |
| POST | `/api/sync` | Manually triggers a match sync |

**Database:** SQLite file `stake.db`. Tables: `users`, `teams`, `user_teams`, `matches`, `push_subscriptions`, `notification_deliveries`, `leaderboard_state`. No migration tool — schema is created on startup. Scoring: 1 point per finished match win for an owned team.

**Frontend routes:**
- `/` — Matches page (upcoming/finished, grouped)
- `/leaderboard` — Players ranked by points
- `/notifications` — Browser push subscription settings

**Match sync:** `handlers/sync.go` maps football-data.org TLA codes to local team IDs. Matches are upserted with `ON CONFLICT`. Logs a warning for unrecognised teams.

## Environment Variables
- `FOOTBALL_DATA_API_KEY` — Required for live match sync from football-data.org
- `VAPID_PUBLIC_KEY` — Public VAPID key for browser push subscriptions
- `VAPID_PRIVATE_KEY` — Private VAPID key used to sign Web Push requests
- `VAPID_SUBJECT` — Contact subject for VAPID, e.g. `mailto:admin@example.com`
- `PORT` — Server port (default: `8080`)

Generate VAPID keys with `github.com/SherClockHolmes/webpush-go`'s `webpush.GenerateVAPIDKeys()` helper in a tiny local Go snippet or command.
