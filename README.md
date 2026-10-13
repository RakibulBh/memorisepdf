# Unmemorize 📚 — your lectures, worth revising from

Organise a course into **subjects** and **modules**, upload each module's
lecture slides, and turn them into quizzes, flashcards and teaching cards.

Everything generated is stored. Pressing *Quiz me* on a module you have already
studied sends nothing to the model — it only analyses lectures it has never seen
before, then serves the whole accumulated pool.

```
Computer Science                    ← subject
└── Databases                       ← module
    ├── lecture01.pdf   ✓ quizzed · ✓ flashcards
    ├── lecture02.pdf   ✓ quizzed
    └── lecture03.pdf   — new

    [ Quiz me ]  →  "1 new file to analyse, reusing 2"
```

## What it does

| Surface | What you get |
| --- | --- |
| Subjects and modules | A course per subject, a taught unit per module |
| Uploads | PDF, PPTX, PPT, DOCX, DOC, TXT and MD — scanned decks are OCR'd |
| Quiz me | Multiple choice at easy, medium or hard, with every option explained |
| Flashcards | Term and definition, flipped with a click or the space bar |
| Teach me | The material explained from the ground up, in one of three styles |

Maths, chemistry and Greek symbols come back as LaTeX and are rendered with
KaTeX, so a formula on a slide stays a formula.

## How the no-duplicate rule works

A `generations` row is the record of work already paid for, keyed on
`(file_id, kind, variant)` with a unique index. Asking for study material runs:

1. List the module's files.
2. Skip any with a **completed** row for this kind and setting.
3. Send only the rest to Gemini.
4. Write each result — the ledger row and its items — in one transaction.
5. Serve the module's whole pool, old items and new together.

Consequences worth knowing:

- **Restudying is free.** A module whose files are all analysed costs nothing.
- **A harder quiz is new work.** The difficulty is part of the key, so asking
  for `hard` generates — and your `easy` questions are still there afterwards.
- **Flashcards have no setting**, so there is only ever one flashcard
  generation per file.
- **A failure is recorded as failed**, not complete, so the next attempt retries
  that file rather than leaving a silent gap.
- **A file is its bytes.** Identity is the SHA-256, so re-uploading the same
  deck is recognised instead of analysed again, and the same slides in two
  modules are stored on disk once.

## Stack

| Layer | Choice |
| --- | --- |
| API | Go 1.27, chi v5 |
| Database | Postgres 18, schema embedded with goose |
| Text extraction | Apache Tika 3.3 (`-full`, so Tesseract is included) |
| Generation | Gemini `gemini-3.6-flash` with a response schema per kind |
| Auth | Google OAuth 2.0 + PKCE, opaque sessions stored hashed |
| Frontend | Next.js 16, React 19, Tailwind 4 |
| Hosting | Four-container compose stack on your own server |

See [docs/architecture.md](docs/architecture.md) for why each of these, and
[docs/deploy.md](docs/deploy.md) for the deployment.

## Running it locally

Requires Go 1.27+, Node 20.9+, and Docker.

```bash
make dev-up          # postgres + tika in the background
cp .env.example .env # then fill in the two keys below
make run             # the api on :8080
make web-dev         # the frontend on :3000
```

Two values in `.env` have to be real:

- `GEMINI_API_KEY` from [aistudio.google.com/apikey](https://aistudio.google.com/apikey)
- `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` from a Google OAuth client whose
  redirect URI is `http://localhost:8080/auth/google/callback`

The schema is applied by the api on boot, so there is no migration step to run.

## Development

```bash
make check     # gofmt, go vet, go test, tsc --noEmit, eslint
make test      # go test only
```

The store tests run against a real Postgres, because the no-duplicate rule is
enforced by the schema and cannot be checked without one. They skip unless you
point them at a database:

```bash
TEST_DATABASE_URL='postgres://unmemorize:unmemorize@localhost:5432/unmemorize?sslmode=disable' \
  go test ./internal/store
```

CI runs them with a Postgres service container.

The Tika client has the same arrangement, since the only way to know the request
it builds is one Tika accepts is to send it to a real server:

```bash
TEST_TIKA_URL=http://localhost:9998 TEST_PDF=some-lecture.pdf go test ./internal/tika
```

| Path | Contents |
| --- | --- |
| `cmd/api` | Server entry point |
| `internal/config` | Every environment variable, read once at boot |
| `internal/store` | Postgres queries, one file per table group |
| `internal/migrations` | Embedded goose SQL |
| `internal/auth` | Google OAuth flow, sessions, request middleware |
| `internal/files` | Content-addressed disk storage |
| `internal/tika` | Text extraction over plain HTTP |
| `internal/llm` | Gemini client, prompts, response schemas |
| `internal/generate` | The pipeline that skips already-analysed files |
| `internal/server` | Router, handlers, the progress stream |
| `web` | Next.js frontend |

## API

Everything under `/api` needs a session cookie.

| Route | Does |
| --- | --- |
| `GET /health`, `GET /ready` | Liveness; readiness also checks Postgres and Tika |
| `GET /auth/google` | Starts sign-in |
| `GET /auth/google/callback` | Finishes it and sets the session cookie |
| `POST /auth/logout` | Revokes the session server-side |
| `GET /api/me`, `/api/meta`, `/api/usage` | Account, available options, storage used |
| `GET POST /api/subjects` | List and create subjects |
| `GET PATCH DELETE /api/subjects/{id}` | One subject |
| `GET POST /api/subjects/{id}/modules` | List and create modules |
| `GET PATCH DELETE /api/modules/{id}` | One module |
| `GET POST /api/modules/{id}/files` | List and upload lectures |
| `POST /api/modules/{id}/generate` | Analyse new files, streaming progress |
| `GET /api/modules/{id}/quiz?difficulty=` | The stored question pool |
| `GET /api/modules/{id}/flashcards` | The stored flashcard pool |
| `GET /api/modules/{id}/teaching?style=` | The stored teaching card pool |
| `DELETE /api/files/{id}` | Delete a lecture and everything made from it |
| `GET /api/files/{id}/download` | The original file |

`POST .../generate` answers with an event stream rather than JSON, so the work
and the progress for it share one request — dropping the connection cancels the
work it belonged to. `EventSource` cannot be used because it only issues GETs;
`lib/api.ts` reads the stream off the `fetch` body instead.

## Limits

Signup is open to any Google account, so these are enforced per user:

| Limit | Default | Env |
| --- | --- | --- |
| Single upload | 100 MB | `MAX_FILE_SIZE` |
| Total storage | 500 MB | `USER_STORAGE_QUOTA` |
| Files per module | 50 | `MAX_FILES_PER_MODULE` |
| Generations | 20/hour | `GENERATE_RATE_LIMIT` |

## License

Not currently licensed. Add a `LICENSE` file before sharing this publicly.
