# Deploying to the server

A push to `main` builds both container images, pushes them to `ghcr.io`, and
restarts the stack on your server. The GitHub-hosted runner reaches the box by
joining your tailnet as an ephemeral node — nothing is exposed to the public
internet and no self-hosted runner is involved.

```
push to main
  └─ test    go vet · go test (against a real Postgres) · tsc · eslint
  └─ build   docker build ×2 → ghcr.io/rakibulbh/unmemorize-{api,web}@sha256:…
  └─ deploy  tailscale up (tag:ci) → ssh → docker compose up -d
             └─ waits for /ready, which checks Postgres and Tika too
```

Both images are pinned by digest, so the server runs exactly what CI built.

Because sign-in is Google OAuth, the app **must** be reachable over public
HTTPS at a real hostname: Google will only redirect back to an address you have
registered, and it will not redirect to a tailnet IP.

## The stack

```
                    ┌──────────────────── proxy-network ─────────┐
internet ──► nginx-proxy-manager ──┬──► web   :3000  (Next.js)   │
             (TLS, your domain)    └──► api   :8080  (Go)        │
                                          │                      │
                    ┌─────── backend ─────┼──────────────────────┘
                    │                     │
                    ├──► db   :5432  (Postgres 18, volume pgdata)
                    └──► tika :9998  (Apache Tika + Tesseract)
```

`db` and `tika` sit on a private network the frontend is not attached to.
Neither publishes a port to the host. The api is published on `127.0.0.1:8080`
only, so the deploy's health check has somewhere to call.

## One-time setup

### 1. Define `tag:ci` in the Tailscale ACL

Do this **first** — the tag has to exist before the OAuth client can be given
it. Admin console → **Access controls**:

```jsonc
{
  "tagOwners": {
    "tag:ci": ["autogroup:admin"],
  },
  "acls": [
    {
      "action": "accept",
      "src":    ["tag:ci"],
      "dst":    ["100.86.149.92:22"],
    },
  ],
}
```

Only port 22 is needed. The health check runs over that ssh connection rather
than against a published port, so 8080 does not have to be opened.

Merge this into your existing policy rather than replacing it — you already
have a `tag:ci` block for `seats_automator`, so you only need to add port 22 of
this host if it is a different machine.

### 2. Tailscale OAuth client

If you already made one for `seats_automator`, **reuse it** — the same client
and the same two secrets work for both repositories. Otherwise: admin console →
**Trust credentials** → **Credential** → **OAuth**:

1. Give the **Keys → Auth Keys** scope **Write**. That is the only scope needed.
2. Select the tag `tag:ci`.
3. **Generate credential**, then copy both the client id and the secret.

The secret is shown once.

### 3. Google OAuth client

[console.cloud.google.com](https://console.cloud.google.com) → **APIs &
Services** → **Credentials** → **Create credentials** → **OAuth client ID** →
**Web application**.

| Field | Value |
| --- | --- |
| Authorised JavaScript origins | `https://unmemorize.example.com` |
| Authorised redirect URIs | `https://unmemorize.example.com/auth/google/callback` |

The redirect URI must match `APP_URL` + `/auth/google/callback` **exactly**,
including the scheme and any trailing path. A mismatch gives
`redirect_uri_mismatch` at sign-in rather than a useful error.

On the **OAuth consent screen**, set the publishing status to **In production**
if anyone other than you will sign in; while it is in **Testing**, only accounts
you list explicitly can get in.

### 4. GitHub secrets

Repo → Settings → Secrets and variables → Actions → **Secrets**:

| secret | value |
| --- | --- |
| `TS_OAUTH_CLIENT_ID` | OAuth client id from step 2 |
| `TS_OAUTH_SECRET` | OAuth client secret from step 2 |
| `DEPLOY_SSH_KEY` | private key whose public half is in the server's `authorized_keys` |

`DEPLOY_SSH_KEY` is the whole file including the `-----BEGIN …-----` lines.

### 5. GitHub variables

| variable | value |
| --- | --- |
| `DEPLOY_HOST` | `100.86.149.92` |
| `DEPLOY_USER` | `deploy` |

Optional overrides:

| variable | default | notes |
| --- | --- | --- |
| `DEPLOY_PATH` | `apps/unmemorize` | relative to the ssh user's home, or absolute |
| `DEPLOY_PORT` | `8080` | host port the api binds on, localhost only |

`DEPLOY_HOST` is the tailnet address deliberately. A GitHub runner has no route
to your LAN, so a `192.168.x.x` address will not work from CI.

### 6. Server

Needs Docker with the compose plugin, the deploy key authorised, and the
`proxy-network` that nginx-proxy-manager uses. Then create the env file:

```sh
mkdir -p ~/apps/unmemorize/data

cat > ~/apps/unmemorize/.env <<'EOF'
# Database. Compose creates the database user with this and hands it to the app,
# which builds and escapes the connection string itself -- so any characters are
# fine here.
POSTGRES_PASSWORD=<a long random string>

ENV=production

# Must be the public HTTPS address. Google redirects back to it.
APP_URL=https://unmemorize.example.com
WEB_URL=https://unmemorize.example.com

GOOGLE_CLIENT_ID=<from step 3>
GOOGLE_CLIENT_SECRET=<from step 3>

# From aistudio.google.com/apikey
GEMINI_API_KEY=<your key>
GEMINI_MODEL=gemini-3.6-flash

# Open signup, so these are what stop one visitor filling the disk
# or spending the whole Gemini quota.
MAX_FILE_SIZE=104857600
USER_STORAGE_QUOTA=524288000
MAX_FILES_PER_MODULE=50
GENERATE_RATE_LIMIT=20

LOG_LEVEL=info
EOF

chmod 600 ~/apps/unmemorize/.env
```

The workflow refuses to deploy if this file is missing or if any of
`POSTGRES_PASSWORD`, `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`,
`GEMINI_API_KEY` or `APP_URL` is empty, rather than starting a container that
would crash on boot or fail at first sign-in.

**The file lives at the compose project root, not in `data/`.** That is the one
place compose reads automatically, which is what lets you run
`docker compose logs` on the server without passing `--env-file` every time.

### 7. nginx-proxy-manager

Add a proxy host for your domain, then set up **two** routes. This is the step
most easily missed, and without it the frontend loads but every request fails.

| | Forward to | Why |
| --- | --- | --- |
| Default | `unmemorize-web:3000` | the pages |
| `/api` | `unmemorize-api:8080` | the JSON API |
| `/auth` | `unmemorize-api:8080` | the Google sign-in round trip |

In nginx-proxy-manager, the extra two go under the proxy host's **Custom
locations** tab. Request a Let's Encrypt certificate on the **SSL** tab and turn
on **Force SSL**.

Both paths must reach the api **on the same hostname as the pages**. The session
cookie is set for that host, so the browser only sends it back to that host —
putting the api on a separate subdomain would mean the cookie never arrives and
every request looks signed-out.

Generation streams progress over a long-lived response. Add this to the proxy
host's **Advanced** tab so nginx does not hold the stream back until it finishes:

```nginx
proxy_buffering off;
proxy_read_timeout 600s;
proxy_request_buffering off;
client_max_body_size 110m;
```

`client_max_body_size` has to exceed `MAX_FILE_SIZE`, or nginx rejects a large
lecture upload before the api ever sees it.

## Running a deploy

Automatic on push to `main` (documentation-only changes are skipped), or
manually from the Actions tab via **Run workflow**.

## Operating

```sh
cd ~/apps/unmemorize
docker compose logs -f api
docker compose ps
docker compose restart api
```

Records also go to `data/app.log`, so they survive a restart and can be read
without docker:

```sh
tail -f ~/apps/unmemorize/data/app.log
grep '"generation failed"' ~/apps/unmemorize/data/app.log
```

The file rolls to `app.log.1` at 8 MB, keeping one previous generation.
`LOG_FILE` changes the path (setting it empty disables the file) and `LOG_LEVEL`
takes `debug`/`info`/`warn`/`error`.

### Backups

Two things hold state, and you need both:

```sh
# The database: subjects, modules, and everything generated.
docker compose exec -T db pg_dump -U unmemorize unmemorize | gzip > unmemorize-$(date +%F).sql.gz

# The uploaded lecture files.
tar czf unmemorize-files-$(date +%F).tar.gz -C ~/apps/unmemorize data/files
```

A database without the files leaves every module pointing at lectures that are
no longer on disk. The files without the database leaves bytes nothing refers
to. The ledger of what has been analysed lives in the database, so losing it
means every file would be sent to Gemini again.

### Upgrading Postgres

The volume is mounted at `/var/lib/postgresql`, not `/var/lib/postgresql/data`,
which is what Postgres 18+ expects — it keeps data in a major-version
subdirectory so `pg_upgrade --link` works across a version bump. Do not change
this mount point back; an 18+ container refuses to start against the old layout.

## Costs to watch

Signup is open to any Google account, so the limits in the env file are the only
thing between a curious visitor and your Gemini bill. `GENERATE_RATE_LIMIT` is
per user per hour and counts requests rather than files, so one person can still
cause a lot of work by putting fifty lectures in a module. If the instance ever
gets found, the quickest lever is to drop `GENERATE_RATE_LIMIT` and
`USER_STORAGE_QUOTA` and redeploy.
