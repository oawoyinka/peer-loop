# PeerLoop (Go)

A Go rebuild of the PeerLoop prototype, starting with the MVP: registration,
login, profile with self-rated skills, and peer matching by skill/level gap.

## Stack

- **Language:** Go, standard library `net/http` only (no web framework)
- **Routing:** Go 1.22+ method-aware `http.ServeMux` (`"GET /profile"` style patterns)
- **Database:** SQLite via `modernc.org/sqlite` (pure Go, no CGo/gcc required)
- **Auth:** bcrypt password hashing (`golang.org/x/crypto/bcrypt`) + server-side
  sessions in a `sessions` table, referenced by an HttpOnly cookie
- **Templates:** `html/template`, server-rendered pages, no JS framework

## Project layout

```
peerloop/
├── cmd/server/main.go        # entrypoint: wires DB, stores, handlers, routes
├── internal/
│   ├── database/              # connection + schema migration
│   ├── models/                # plain structs shared across packages
│   ├── store/                 # SQL queries (users, skills, matching)
│   ├── auth/                  # password hashing + session cookies
│   ├── handlers/               # HTTP handlers (one file per feature area)
│   └── middleware/            # RequireAuth route guard
├── web/
│   ├── templates/             # html/template pages + shared nav partial
│   └── static/style.css
└── data/                      # peerloop.db gets created here at runtime
```

## Why these choices

- **modernc.org/sqlite** instead of `mattn/go-sqlite3`: the latter needs CGo
  and a C compiler, which is one more thing to get right when you're just
  starting out. modernc.org's driver is pure Go — `go build` just works.
- **Standard library routing**: Go 1.22 added `"METHOD /path"` patterns and
  `{param}` wildcards to `http.ServeMux`, which covers most of what a
  framework like Gin gives you for a project this size. Worth learning the
  primitives before reaching for a framework.
- **Server-rendered HTML** rather than a JSON API + frontend framework: keeps
  the whole request/response cycle in Go, which is the point of this
  exercise. You can always add a JSON API alongside this later.

## Setup

1. Install Go 1.22 or later.
2. From the `peerloop/` directory:
   ```
   go mod tidy
   ```
   This downloads `modernc.org/sqlite` and `golang.org/x/crypto` and writes
   `go.sum`. (This step needs network access — it wasn't run when this
   project was generated.)
3. Run it:
   ```
   go run ./cmd/server
   ```
4. Visit http://localhost:8080 — you'll land on the login page. Click
   "Create an account" to register.

The SQLite file is created automatically at `data/peerloop.db` the first
time you run the server, along with a starter list of skills (Prompt
Engineering, RAG, Fine-tuning, Go, Python, etc.) so the "add a skill"
autocomplete isn't empty.

## Trying the matching logic

Matching only works between users who've both rated themselves on the
*same* skill name. To see it in action:

1. Register two accounts (e.g. in two browser tabs, or one normal + one
   incognito, since sessions are cookie-based).
2. On each, go to your profile and add the same skill (e.g. "RAG") at
   different levels (say 4 and 1).
3. Go to "Find Peers" on either account — you should see the other user
   listed, with a note on who could teach whom based on the level gap.

## What's deliberately not built yet

This is the auth + profiles + skill-matching slice only, per the current
scope. Not yet included (all present in the original prototype and worth
building next, in roughly this order):

1. **Chat** between matched peers
2. **Session booking** with a Google Meet link attached
3. **AI-generated call recaps** (would call the Anthropic API or similar
   after a session)
4. **Community rooms / learning circles**
5. **Points, badges, and the leaderboard**
6. **Peer reviews** restricted to people who actually had a session

Each of these fits cleanly into the existing structure: a new table in
`database.go`, a new store in `internal/store/`, a new handler file in
`internal/handlers/`, and new routes in `main.go`.

## Known simplifications (fine for learning, flag before "production")

- Sessions never expire early (just after 7 days) — no "log out
  everywhere" button yet.
- No CSRF protection on forms yet.
- No rate-limiting on login attempts.
- `SetMaxOpenConns(1)` avoids SQLite locking issues but means only one
  request touches the DB at a time — fine for a learning project, not for
  real concurrent traffic (Postgres would remove this limit later).
