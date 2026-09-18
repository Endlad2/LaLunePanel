# ☾ LaLune Panel

A self-hosted control panel for WebRTC-tunneled VPNs (**olcRTC**, **OpenFlux**).
Ships as a single Debian-slim Docker container: Go backend (API on `/api/`) +
static frontend (on `/`). Protocol server binaries are pulled on demand from
GitHub Releases.

## Quick install

```bash
curl -fsSL https://raw.githubusercontent.com/Endlad2/LaLunePanel/master/install.sh | bash
```

The installer asks:

1. **install** / **uninstall**
2. (on install) the **port** to host on — default `6333`

Then it clones the repo to `/opt/lalune`, builds the image, and starts the
container with a persistent data volume at `/opt/lalune/data`.

## What's inside

| Path ↕▾ | Purpose ↕▾ |
|---|---|
| −`install.sh` | curl-able installer/uninstaller |
| −`Dockerfile` | multi-stage build (Go builder → Debian slim runtime) |
| −`backend/` | Go API server, protocol orchestration, client store |
| −`frontend/` | vanilla-JS SPA (dashboard + clients) |
| −`protocols/` | protocol docs (binaries fetched at runtime) |
| −`.github/workflows/release.yml` | CI: build + release tarballs (with `permissions: write`) |
⚙

## Panels

- **Dashboard** — CPU load, memory, cores, uptime, client counts (polled every 5 s).
- **Clients** — table of clients (name, protocol, days left, status) with a
**+ Add client** flow: pick a protocol, the UI renders that protocol's
required fields, then the backend downloads the binary and generates a config
(+ `olcrtc://` URI when applicable).

## API

| Method ↕▾ | Path ↕▾ | Description ↕▾ |
|---|---|---|
| −`GET` | `/api/health` | liveness |
| −`GET` | `/api/stats` | host metrics + client counts |
| −`GET` | `/api/protocols` | available protocols + their fields |
| −`GET` | `/api/clients` | list clients |
| −`POST` | `/api/clients` | create client `{name, protocol, days, params}` |
| −`GET` | `/api/clients/{id}` | fetch one client |
| −`DELETE` | `/api/clients/{id}` | delete client |
⚙

## Local development

```
cd backend
go run ./cmd/server
# → http://localhost:6333
```

Or with Docker:

```
docker compose up --build
```

## Adding a protocol

Edit `backend/internal/protocol/protocol.go`, add an entry to `registry`
(ID, name, fields, binary URLs) and a matching generator in `config.go`.
The frontend builds the input form automatically from the `fields` array.

## License

Provided as-is. See the original protocol projects for their terms.

