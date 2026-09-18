# Protocol server binaries

Server binaries for each protocol are downloaded at first use into
`<dataDir>/bin/` (i.e. `/data/bin/` inside the container).

| Protocol | amd64 | arm64 |
|---|---|---|
| olcRTC   | `olcrtc-linux-amd64` | `olcrtc-linux-arm64` |
| OpenFlux | `OpenFlux-linux-amd64` | `OpenFlux-linux-arm64` |

Sources (GitHub Releases, `Endlad2/LaLunePanelServers`):

- https://github.com/Endlad2/LaLunePanelServers/releases/latest/download/olcrtc-linux-amd64
- https://github.com/Endlad2/LaLunePanelServers/releases/latest/download/olcrtc-linux-arm64
- https://github.com/Endlad2/LaLunePanelServers/releases/latest/download/OpenFlux-linux-amd64
- https://github.com/Endlad2/LaLunePanelServers/releases/latest/download/OpenFlux-linux-arm64

The backend resolves the correct URL from `runtime.GOARCH` in
`internal/protocol/download.go`.

## Config generation

`internal/protocol/config.go` turns collected UI params into:

- **olcRTC** — a `config.yaml` matching the olcRTC server schema, plus an
  `olcrtc://` URI for the client.
- **OpenFlux** — an exit-node command line (`--role=exit --mode=… --transport=…`)
  plus a plain config summary.
