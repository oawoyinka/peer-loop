# PeerLoop (Go)

A full Go rebuild of the PeerLoop prototype: auth, profiles, skill-based
matching, chat, session booking with a Google Meet link, AI-generated
session recaps, community rooms, a points/badges economy, and gated peer
reviews.

## Stack

- **Language:** Go, standard library `net/http` only (no web framework)
- **Routing:** Go 1.22+ method-aware `http.ServeMux` (`"GET /profile"` style patterns, `{id}` wildcards)
- **Database:** SQLite via `modernc.org/sqlite` (pure Go, no CGo/gcc required)
- **Auth:** bcrypt password hashing + server-side sessions in a `sessions`
  table, referenced by an HttpOnly cookie
- **AI recaps:** direct calls to the Anthropic Messages API over
  `net/http` (no SDK) — see `internal/aiclient/anthropic.go`
- **Templates:** `html/template`, server-rendered pages, no JS framework

## Project layout

```
peerloop/
├── cmd/server/main.go          # entrypoint: wires DB, stores, handlers, routes
├── internal/
│   ├── database/                # connection + full schema migration
│   ├── models/                  # plain structs shared across packages
│   ├── store/                   # SQL queries, one file per feature area:
│   │   ├── user_store.go
│   │   ├── skill_store.go        # skills, ratings, matching, IsMatch
│   │   ├── message_store.go      # chat
│   │   ├── booking_store.go      # session booking
│   │   ├── room_store.go         # learning circles
│   │   ├── points_store.go       # points + badges + leaderboard
│   │   └── review_store.go       # peer reviews
│   ├── aiclient/anthropic.go    # minimal Anthropic Messages API client
│   ├── auth/                    # password hashing + session cookies
│   ├── handlers/                # HTTP handlers, one file per feature area
│   └── middleware/              # RequireAuth route guard
├── web/
│   ├── templates/                # html/template pages + shared nav partial
│   └── static/style.css
└── data/                         # peerloop.db gets created here at runtime
```

## Setup

1. Install Go 1.22 or later.
2. From the `peerloop/` directory:
   ```
   go mod tidy
   ```
   Downloads `modernc.org/sqlite` and `golang.org/x/crypto`, writes `go.sum`.
3. (Optional, for AI recaps) set your Anthropic API key:
   ```
   export ANTHROPIC_API_KEY=sk-ant-...
   ```
   Without this, the app still runs fine — clicking "Generate AI recap"
   just shows a message telling you the key isn't set, instead of erroring.
4. Run it:
   ```
   go run ./cmd/server
   ```
5. Visit http://localhost:8080.

## Feature tour

Register **two accounts** (e.g. one normal + one incognito window) to see
matching, chat, and booking actually do something — most of this only
lights up once two people share a skill.

1. **Register / log in** → lands on **Dashboard**: your skill count,
   potential-peer count, points, and top matches.
2. **Profile**: add a bio, self-rate skills 1-5. Adding a skill earns
   points.
3. **Find Peers**: anyone who shares a skill with you, ranked by how big
   the level gap is (biggest gaps = clearest teacher/learner pairing).
   Each row has **Message** and **Book** links.
4. **Messages**: 1:1 chat, restricted to people you're actually matched
   with. Your first message to a given peer earns points.
5. **My Sessions → Book**: propose a session (skill, date/time, notes).
   A Google Meet link (`meet.google.com/new`) is attached automatically —
   see the note on this below. The invited peer confirms or cancels;
   either side can mark it **completed**, which unlocks recaps and reviews.
6. **AI recap**: on a completed session, click "Generate AI recap" to have
   Claude turn your session notes into a structured summary. Needs
   `ANTHROPIC_API_KEY`.
7. **Review**: after completion, either participant can leave a 1-5 rating
   + comment about the other — one review per person per session.
8. **Rooms**: create or join a topic-based learning circle with a shared
   post feed.
9. **Leaderboard**: points from adding skills, first messages, booking
   and completing sessions, room posts, and leaving reviews. Badges are
   derived thresholds (10 / 50 / 100 / 250 points), not separately stored.

## Why these choices

- **modernc.org/sqlite** instead of `mattn/go-sqlite3`: pure Go, no CGo/gcc
  needed to build.
- **Standard library routing**: Go 1.22's `"METHOD /path"` patterns and
  `{param}` wildcards cover what a framework like Gin gives you at this
  scale.
- **Server-rendered HTML**: keeps the whole request/response cycle in Go.
- **`meet.google.com/new` for session links**: this is a real Google URL
  that starts a brand-new Meet call when opened by a signed-in Google
  user — no OAuth/Calendar API setup needed to get booking working end to
  end. The trade-off: it's the *same* link for every session rather than
  a unique one tied to that specific booking, and it only auto-creates a
  meeting for whoever's signed into a Google account in that browser. For
  real per-session links, swap `generateMeetLink()` in
  `internal/handlers/booking_handlers.go` for a call to the Google
  Calendar API (`events.insert` with `conferenceData` set), which does
  require OAuth.
- **Points as a running total, not an event log**: `points` is a single
  row per user updated with `total = total + N`. Simple and fast; the
  trade-off is no history of *why* someone has however many points. If
  you want an activity feed later, add a `point_events` table and switch
  `PointsStore.Award` to insert there instead.
- **Badges computed on the fly** from the point total (see
  `store.Badges`), not stored — avoids a whole table for something
  that's purely derived.

## Known simplifications (fine for learning, flag before "production")

- Sessions (login) never expire early — no "log out everywhere" button.
- No CSRF protection on forms.
- No rate-limiting on login attempts.
- `SetMaxOpenConns(1)` avoids SQLite locking issues but serializes all DB
  access — fine for learning, not for real concurrent traffic.
- Chat and room feeds are plain page reloads, not real-time (no
  WebSockets/SSE) — sending a message redirects back to the same page.
- The AI recap is generated from whatever you type into the "notes"
  field when booking — there's no actual video/audio transcription of
  the call itself.
- Meet links are not unique per session (see above).
