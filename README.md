# World Cup Stake Cashpoints

A World Cup 2026 sweepstake tracker with live match syncing, leaderboard scoring, and browser push notifications.

The app is a Go/Gin backend with a SolidJS/Vite frontend. The frontend builds into `frontend/dist/` and is embedded into the Go binary for production serving.

## Features

- Track World Cup matches, scores, stages, and owners for each team
- Rank players on a leaderboard by finished match wins
- Subscribe to browser push notifications for match starts and leaderboard changes
- Sync live match data from football-data.org
- Persist data in SQLite

## Tech Stack

- Backend: Go, Gin, SQLite via `modernc.org/sqlite`
- Frontend: SolidJS, Vite, TypeScript
- Push notifications: Web Push / VAPID
- Deployment: Docker and Docker Compose

## Requirements

- Go 1.26+
- Node.js 24+ and npm
- Optional: Docker and Docker Compose
- Optional: `air` for backend live reload

## Environment Variables

| Variable | Required | Description |
| --- | --- | --- |
| `FOOTBALL_DATA_API_KEY` | No | API key for syncing matches from football-data.org. Without it, live sync and manual sync are disabled. |
| `VAPID_PUBLIC_KEY` | No | Public VAPID key for browser push subscriptions. |
| `VAPID_PRIVATE_KEY` | No | Private VAPID key for signing Web Push messages. |
| `VAPID_SUBJECT` | No | Contact subject for VAPID, such as `mailto:admin@example.com`. |
| `DATA_DIR` | No | Directory for SQLite data. Defaults to `data`. |
| `PORT` | No | HTTP port. Defaults to `8080`. |

## Development

Install frontend dependencies:

```bash
cd frontend
npm install
```

Run the frontend dev server:

```bash
cd frontend
npm run dev
```

The Vite dev server proxies `/api` requests to the Go backend on `localhost:8080`.

Run the backend:

```bash
go run .
```

Or run with live reload if `air` is installed:

```bash
air
```

## Production Build

Build the frontend first because the Go binary embeds `frontend/dist/`:

```bash
cd frontend
npm run build
```

Build and run the Go server:

```bash
go build -o ./tmp/main .
./tmp/main
```

The app will be available at `http://localhost:8080` unless `PORT` is set.

## Docker

Build and run with Docker Compose:

```bash
docker compose up --build
```

Compose mounts `./data` into the container at `/data` and sets `DATA_DIR=/data`.

## API Routes

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/api/users` | Returns seeded players for subscription settings. |
| `GET` | `/api/matches` | Returns matches and team ownership data. |
| `GET` | `/api/leaderboard` | Returns players ranked by wins. |
| `GET` | `/api/push/vapid-public-key` | Returns the configured VAPID public key. |
| `POST` | `/api/push/subscribe` | Creates or updates a browser push subscription. |
| `PUT` | `/api/push/preferences` | Updates push notification preferences. |
| `POST` | `/api/push/unsubscribe` | Deletes a browser push subscription. |
| `POST` | `/api/sync` | Triggers a match sync when `FOOTBALL_DATA_API_KEY` is set. |

## Data

On startup, the backend creates the SQLite schema and seeds players and World Cup teams when the relevant tables are empty. Scoring is 1 point per finished match win for an owned team.

The SQLite database is stored at `${DATA_DIR}/stake.db`, or `data/stake.db` by default.
