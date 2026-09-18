# ☾ LaLune Panel

Self-hosted control panel for WebRTC-tunneled VPNs (**olcRTC**, **OpenFlux**).
Single Debian-slim Docker image: Go backend (API on `/api/`) + static
frontend (on `/`).

## Protocol roles

| Protocol | Roles | Notes |
|---|---|---|
| **olcRTC**  | `srv` / `cnc` | Two distinct sides. The UI shows a Server/Client radio and renders different fields per role. The server emits an `olcrtc://` URI for the client. |
| **OpenFlux**| always `exit` | Always an exit node. `--role=exit` is hardcoded by the generator; the UI never asks for a role. |

## Add-client flow

1. **Step 1** — Name, Protocol, (Role radio if the protocol has roles), Days.
2. **Step 2** — protocol-specific fields, built from `common_fields` plus
   `fields_server` / `fields_client` (for olcRTC) or just `common_fields`
   (for OpenFlux). Conditional fields (`show_if`) appear/disappear live.
3. **Result** — the generated config, the shareable `olcrtc://` URI (olcRTC
   server), and the local SOCKS5 address (olcRTC client).

## API

| Method | Path | Description |
|---|---|---|
| `GET`    | `/api/health`    | liveness |
| `GET`    | `/api/stats`     | host metrics + client counts |
| `GET`    | `/api/protocols` | protocols (per-role field lists) |
| `GET`    | `/api/clients`   | list clients |
| `POST`   | `/api/clients`   | create `{name, protocol, role, days, params}` |
| `GET`    | `/api/clients/{id}` | fetch one |
| `DELETE` | `/api/clients/{id}` | delete |

For protocols with `has_roles: false`, the `role` field in the POST body is
ignored and forced to `srv`.

## Adding a protocol

Edit `backend/internal/protocol/protocol.go`:

- `HasRoles: true` + `FieldsServer`/`FieldsClient` if the protocol has two
  distinct sides.
- `HasRoles: false` + only `CommonFields` if it's single-sided.

Then add a matching generator in `config.go`. Use `ShowIf` for conditional
fields (e.g. show `local_ip` only when `exit_mode=l3`).
