# PeerLoop (Go)

A full Go rebuild of the PeerLoop prototype: auth, profiles, skill-based
matching, chat, session booking with a Google Meet link, AI-generated
session recaps, community rooms, a points/badges economy, and gated peer
reviews.

## Stack

- **Language:** Go, standard library `net/http` only (no web framework)
- **Routing:** Go 1.22+ method-aware `http.ServeMux`
- **Database:** PostgreSQL via `github.com/jackc/pgx/v5` — chosen over
  SQLite specifically because it works with genuinely free, *persistent*
  hosting (see "Hosting" below). Local dev needs a Postgres instance too;
  see Setup.
- **Auth:** bcrypt password hashing + server-side sessions in a `sessions`
  table, referenced by an HttpOnly cookie
- **AI recaps:** direct calls to the Anthropic Messages API over
  `net/http` (no SDK) — see `internal/aiclient/anthropic.go`
- **Templates:** `html/template`, server-rendered pages, no JS framework

## Project layout

```
peerloop/
├── cmd/server/main.go          # entrypoint: wires DB, stores, handlers, routes
├── render.yaml                 # Render blueprint (see Hosting)
├── internal/
│   ├── database/                # connection + full schema migration (Postgres)
│   ├── models/                  # plain structs shared across packages
│   ├── store/                   # SQL queries, one file per feature area
│   ├── aiclient/anthropic.go    # minimal Anthropic Messages API client
│   ├── auth/                    # password hashing + session cookies
│   ├── handlers/                # HTTP handlers, one file per feature area
│   └── middleware/              # RequireAuth route guard
└── web/
    ├── templates/                # html/template pages + shared nav partial
    └── static/style.css
```

## Local setup

1. Install Go 1.22+ and a local Postgres (e.g. `sudo apt install postgresql`,
   or use Docker: `docker run -e POSTGRES_PASSWORD=postgres -p 5432:5432 postgres:16`).
2. Create a database: `createdb peerloop` (or `psql -c "CREATE DATABASE peerloop;"`).
3. From `peerloop/`: `go mod tidy`
4. Set your connection string and run:
   ```
   export DATABASE_URL="postgres://postgres:postgres@localhost:5432/peerloop?sslmode=disable"
   go run ./cmd/server
   ```
5. Visit http://localhost:8080. Tables and starter skills are created
   automatically on first run.
6. (Optional, for AI recaps) `export ANTHROPIC_API_KEY=sk-ant-...` before
   running — without it, "Generate AI recap" shows a setup message
   instead of erroring.

## Hosting it for free (with real persistence)

**Important:** if you're tempted to just deploy this to a platform's free
web-service tier with a local SQLite file, don't — free web services on
every major platform (Render included) wipe their local filesystem on
every restart, redeploy, and idle spin-down. Since Render's free web
services spin down after just 15 minutes of no traffic, a SQLite file
there would reset constantly and everyone's accounts would vanish. That's
exactly why this app uses Postgres: the database lives on a separate
service that survives your web service restarting.

This guide uses two free services together:
- **Neon** — free Postgres with no time limit (unlike Render's free
  Postgres, which auto-deletes 30 days after creation)
- **Render** — free Go web service hosting

Total cost: $0. Trade-off: the free Render web service spins down after
15 minutes idle and takes ~30-60 seconds to wake back up on the next
visit — fine for a project you're sharing with a small group, not for
something expecting instant response at all hours.

### 1. Push your code to GitHub

Render deploys from a Git repo.

```
cd peerloop
git init
git add .
git commit -m "Initial commit"
```
Create a new repo on GitHub, then:
```
git remote add origin https://github.com/<you>/peerloop.git
git branch -M main
git push -u origin main
```

### 2. Create a free Postgres database on Neon

1. Go to https://neon.tech and sign up (no card required).
2. Create a new project — any name, e.g. "peerloop".
3. On the project dashboard, find the **connection string** (Neon shows
   it directly on the overview page) — it looks like:
   ```
   postgres://<user>:<password>@<host>/<dbname>?sslmode=require
   ```
4. Copy that. You'll paste it into Render in step 4.

### 3. Create the web service on Render

1. Go to https://render.com and sign up (no card required for the free
   path).
2. Click **New → Blueprint**, connect your GitHub account, and pick your
   `peerloop` repo. Render reads `render.yaml` (already in this repo) and
   sets up the service automatically — Go runtime, free plan, correct
   build/start commands.
   - If you'd rather not use the blueprint: **New → Web Service**, pick
     your repo, set:
     - Runtime: **Go**
     - Build command: `go build -o bin/server ./cmd/server`
     - Start command: `./bin/server`
     - Instance type: **Free**

### 4. Set your environment variables

In the Render dashboard for your new service, go to **Environment** and
add:

| Key | Value |
|---|---|
| `DATABASE_URL` | the Neon connection string from step 2 |
| `ANTHROPIC_API_KEY` | *(optional)* your Anthropic API key, only if you want AI recaps to work |

Save — Render redeploys automatically.

### 5. Watch it deploy, then visit your link

Render shows build logs live. Once it says "Live", your app is at:
```
https://peerloop-<something>.onrender.com
```
That's the link to send people. On first request after any idle period,
expect a ~30-60 second wait while it wakes up — that's normal for the
free tier, not a bug.

### 6. Every future update

```
git add .
git commit -m "describe your change"
git push
```
Render redeploys automatically on every push to `main`.

### Optional: a custom domain

Render's free tier supports custom domains. In your service's
**Settings → Custom Domains**, add your domain and follow the DNS
instructions (a CNAME record pointing at your `onrender.com` address).
This works on the free plan — no upgrade needed.

### When you outgrow free hosting

- **Cold starts bothering people?** Upgrade the Render service to a paid
  instance ($7/mo) — removes the 15-minute spin-down entirely.
- **Outgrowing Neon's free Postgres limits** (storage/compute caps)?
  Neon's paid tiers scale up without changing your connection string
  setup — same `DATABASE_URL` env var either way.

## Hosting it on Vercel instead

Vercel's Go runtime (currently in Beta) can run a full `net/http` server
straight from `cmd/server/main.go` — no rewrite needed, since this app
already listens on the `PORT` env var and this repo includes a
`vercel.json` telling Vercel to use the Go framework preset. You still
need Neon for the database, for the same reason as the Render setup above:
Vercel Functions have no persistent local disk either.

Two things worth knowing before you pick this over Render:
- **Personal/non-commercial only on the free Hobby plan** — check Vercel's
  current terms if this stops being a hobby project.
- **It's a serverless model, not a persistent process** — each request can
  hit a fresh instance, so there's a per-request duration cap (check your
  Vercel dashboard for the current limit). This shouldn't affect normal
  usage, but a slow AI recap generation is the one place it could
  theoretically matter.

Steps:

1. Push this repo to GitHub (see step 1 above if you haven't).
2. Create your free Neon database (step 2 above) and copy its connection
   string.
3. Go to https://vercel.com and sign up (no card required for Hobby).
4. Click **Add New → Project**, import your `peerloop` GitHub repo.
5. Vercel should detect the Go framework preset automatically from
   `vercel.json`. If it asks you to confirm a framework, choose **Go**.
6. Before deploying, open **Environment Variables** and add:
   - `DATABASE_URL` → your Neon connection string
   - `ANTHROPIC_API_KEY` → *(optional)* if you want AI recaps
7. Click **Deploy**. Vercel builds the Go binary and gives you a live URL
   like `https://peerloop-<something>.vercel.app` — that's your shareable
   link.
8. Future updates: just `git push` to your repo's default branch; Vercel
   redeploys automatically.

If you hit friction with the Go runtime being in Beta, Render (above) is
the more battle-tested path for a full server like this one.



- **Postgres over SQLite**: the entire reason for this migration — see
  "Hosting" above. The trade-off is losing SQLite's "just a file, zero
  setup" simplicity; the payoff is that free hosting actually keeps your
  data.
- **pgx over lib/pq**: pgx is the actively maintained, faster, more
  complete Postgres driver for Go; lib/pq is in maintenance mode.
- **Standard library routing**: Go 1.22's `"METHOD /path"` patterns and
  `{param}` wildcards cover what a framework like Gin gives you at this
  scale.
- **`meet.google.com/new` for session links**: a real Google URL that
  starts a brand-new Meet call when opened by a signed-in Google user —
  no OAuth/Calendar API setup needed. Trade-off: it's the same link for
  every session rather than a unique one tied to that specific booking.
  For real per-session links, swap `generateMeetLink()` in
  `internal/handlers/booking_handlers.go` for a Google Calendar API call
  (`events.insert` with `conferenceData` set), which does require OAuth.
- **Points as a running total, not an event log**: simple and fast; no
  history of *why* someone has however many points. Add a `point_events`
  table later if you want an activity feed.
- **Badges computed on the fly** from the point total (`store.Badges`),
  not stored — avoids a table for something purely derived.

## Known simplifications (fine for a hobby project, flag before "real production")

- Login sessions never expire early — no "log out everywhere" button.
- No CSRF protection on forms.
- No rate-limiting on login attempts.
- Chat and room feeds are plain page reloads, not real-time (no
  WebSockets/SSE).
- The AI recap is generated from whatever you type into the "notes"
  field when booking — there's no actual video/audio transcription.
- Meet links are not unique per session (see above).
- Free Render web services spin down after 15 minutes idle — the first
  visitor after a quiet period waits ~30-60 seconds.
