# Architecture

MemorisePDF turns lecture slides into quizzes, flashcards and teaching cards.
You organise work as **subjects** (a course), each holding **modules**, each
holding the **lecture files** you upload. Pressing *Quiz me* on a module
generates from the files that have never been analysed and reuses everything
already stored.

## The stack

| Layer | Choice | Why |
| --- | --- | --- |
| API | Go 1.27, chi v5 | Already the backend; single static binary deploys cleanly |
| Database | Postgres 18 | The data is a strict hierarchy — foreign keys and cascade deletes do the work application code otherwise repeats |
| Migrations | goose, embedded | Schema travels inside the binary, so a deploy can never run against an older schema |
| Text extraction | Apache Tika, self-hosted | Handles PDF, PPTX and DOCX with one HTTP call |
| Generation | Gemini `gemini-3.6-flash` | Structured JSON output via `responseSchema`, so parsing cannot drift |
| Auth | Google OAuth 2.0 + PKCE | No passwords to store, reset or leak |
| Sessions | Opaque tokens in Postgres | Revocable server-side, unlike a JWT |
| Frontend | Next.js 16, React 19, Tailwind 4 | Existing UI; runs as its own container |
| Files | Content-addressed on disk | A bind-mounted volume survives restarts and is trivial to back up |
| Deploy | ghcr.io + compose over Tailscale | Same pattern as `seats_automator` |

## Request path

```
browser
  │
  ▼
nginx-proxy-manager  (TLS, public domain)
  ├── /           ──►  web    :3000   Next.js standalone
  └── /api, /auth ──►  api    :8080   Go
                         ├──►  db     :5432   Postgres   (volume: pgdata)
                         ├──►  tika   :9998   Apache Tika
                         └──►  /data/files    uploaded slides (bind mount)
```

All four services sit in one compose stack. `web` and `api` share an origin
through the proxy, so there is no CORS in production — the browser only ever
talks to one hostname.

## Data model

```
users
  └── subjects                     "Computer Science"
        └── modules                "Databases"
              └── files            "lecture-01.pdf"
                    └── generations        (file, kind, variant) — UNIQUE
                          ├── quiz_questions ──► quiz_answers
                          ├── flashcards
                          └── teaching_cards
```

Every table below `users` carries its parent's foreign key with
`ON DELETE CASCADE`, so deleting a subject removes its modules, files,
generations and every generated item in one statement.

### The no-duplicate rule

`generations` is the ledger of work already paid for:

```sql
UNIQUE (file_id, kind, variant)
```

- **`kind`** is `quiz`, `flashcard` or `teaching`.
- **`variant`** is the setting that changes the output: the difficulty for a
  quiz, the teaching style for a teaching card, and the empty string for
  flashcards, which have no setting.

Pressing *Quiz me* on a module runs this:

1. List the module's files.
2. For each, look for a `generations` row matching `(file_id, 'quiz', difficulty)`
   with `status = 'complete'`.
3. Send **only** the files with no such row to Gemini.
4. Insert each result — the generation row and its items — in one transaction,
   so a crash mid-write leaves no half-analysed file behind.
5. Serve the module's whole accumulated pool, old items and new together.

Studying a module a second time therefore costs nothing. Asking for a *harder*
quiz is a different `variant`, so it does generate — the questions you already
have are not thrown away, and switching back to the easier quiz still finds
them.

A file is identified by the SHA-256 of its bytes. Uploading the same slides
twice under the same module is rejected as a duplicate rather than analysed
again, and the same file uploaded to two modules is stored on disk once.

## File storage

```
/data/
  files/<user_id>/<sha256[:2]>/<sha256><ext>    uploaded slides
  app.log                                        rotating log
```

Content-addressed, with a two-character fan-out so no directory grows past a
few thousand entries. Postgres holds the metadata and the extracted text; the
disk holds the bytes. Extracted text is cached on the `files` row, so a
regenerate at a new difficulty never re-runs Tika.

## Open signup needs limits

Anyone with a Google account can sign in, so the API enforces:

| Limit | Default | Env |
| --- | --- | --- |
| Single upload size | 100 MB | `MAX_FILE_SIZE` |
| Total storage per user | 500 MB | `USER_STORAGE_QUOTA` |
| Files per module | 50 | `MAX_FILES_PER_MODULE` |
| Generation requests | 20/hour/user | `GENERATE_RATE_LIMIT` |

Without these, one visitor can fill the disk or spend the whole Gemini quota.

## Package layout

| Path | Contents |
| --- | --- |
| `cmd/api` | Server entry point |
| `internal/config` | Every environment variable, read once at boot |
| `internal/logging` | `slog` handler with a rotating file sink |
| `internal/store` | Postgres queries, one file per table group |
| `internal/migrations` | Embedded goose SQL |
| `internal/auth` | Google OAuth flow, sessions, request middleware |
| `internal/files` | Content-addressed disk storage and quota checks |
| `internal/tika` | Text extraction over plain HTTP |
| `internal/llm` | Gemini client, prompts, response schemas |
| `internal/generate` | The pipeline that skips already-analysed files |
| `internal/server` | Router, handlers, SSE progress stream |
| `web` | Next.js frontend |

## What changed from v1

- **Nothing was persisted.** Results lived in a Zustand store and vanished on
  refresh. `internal/store` was an empty `database/sql` stub and MongoDB only
  ever held request logs.
- **One file at a time, no ownership.** No accounts, no subjects, no modules —
  you uploaded a PDF and got a quiz back.
- **Hardcoded third parties.** `internal/tika` replaces a Tika client pinned to
  `apache-tike.fly.dev` in source; `@vercel/blob` is gone along with the unused
  `pdf-parse`, `officeparser` and `fs` packages.
- **Fly.io → your server.** `fly.toml` is replaced by a compose stack.
