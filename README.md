# ☾ LaLune Panel

A self-hosted control panel for WebRTC-tunneled VPNs (**olcRTC**, **OpenFlux**).
Ships as a single Debian-slim Docker image: Go backend (API on `/api/`) +
static frontend (on `/`). Protocol server binaries are pulled on demand from
GitHub Releases.

## Quick install

```bash
curl -fsSL https://raw.githubusercontent.com/Endlad2/LaLunePanel/master/install.sh | bash
```

The installer asks:

1. **install** / **uninstall**
2. (on install) the **port** to host on — default `6333`

By default it **pulls the prebuilt multi-arch image from GHCR**
(`ghcr.io/endlad2/lalunepanel:latest`) — fast, no Go toolchain needed.
If the pull fails (private package, missing tag), it offers to build locally.

### Installer flags

| Flag ↕▾ | Effect ↕▾ |
|---|---|
| −`--build` | force a local `docker build` from source instead of pulling |
| −`--tag=v1.2.3` | use a specific image tag (default `latest`) |
| −`--port=8080` | skip the port prompt |
| −`--branch=master` | override the git branch used for local builds |
⚙

Examples:

```
# local build (dev)
curl -fsSL .../install.sh | bash -s -- --build

# pinned version
curl -fsSL .../install.sh | bash -s -- --tag=v1.0.0 --port=8080
```

## What's inside

| Path ↕▾ | Purpose ↕▾ |
|---|---|
| −`install.sh` | curl-able installer/uninstaller (pull from GHCR or build locally) |
| −`Dockerfile` | multi-stage build (Go builder → Debian slim runtime) |
| −`backend/` | Go API server, protocol orchestration, client store |
| −`frontend/` | vanilla-JS SPA (dashboard + clients) |
| −`protocols/` | protocol docs (binaries fetched at runtime) |
| −`.github/workflows/release.yml` | CI: build binaries + tarballs, push multi-arch image to GHCR |
⚙

## Docker image (GHCR)

Published on every push to `master`/`main` and on `v*` tags:

```
ghcr.io/endlad2/lalunepanel:latest   # default branch
ghcr.io/endlad2/lalunepanel:edge     # default branch
ghcr.io/endlad2/lalunepanel:v1.2.3   # tag
ghcr.io/endlad2/lalunepanel:1.2      # tag (major.minor)
ghcr.io/endlad2/lalunepanel:1        # tag (major)
```

Platforms: `linux/amd64`, `linux/arm64`.

> The package must be **public** for anonymous `docker pull` to work.
> Set it in: GitHub repo → Packages → `lalunepanel` → Package settings → Change visibility.

Manual run without the installer:

```
docker run -d --name lalune-panel --restart unless-stopped \
    -p 6333:6333 \
    -v /opt/lalune/data:/data \
    ghcr.io/endlad2/lalunepanel:latest
```

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

