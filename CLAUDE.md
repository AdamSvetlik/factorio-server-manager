# CLAUDE.md — Factorio Server Manager

## Project Overview
`factorio-server-manager` is a Go CLI tool that manages multiple Factorio dedicated server instances via Docker. Each server runs in a `factoriotools/factorio` Docker container.

## Module Name
`github.com/AdamSvetlik/factorio-server-manager`

## Build & Run
```bash
go build ./...                         # compile all packages
go vet ./...                           # static analysis
go run main.go <command>               # run without building
go build -o factorio-server-manager . # build binary
```

## Project Layout
```
.
├── main.go                            # entrypoint
├── cmd/
│   ├── root.go        # cobra root, --data-dir global flag, init of cfgManager
│   ├── server.go      # server create/list/start/stop/status/delete/logs/update/rcon
│   ├── save.go        # save list/copy/export/delete
│   ├── mod.go         # mod list/search/info/install/remove/update
│   ├── config_cmd.go  # config show/set/edit
│   ├── auth.go        # auth login/logout/status
│   └── dashboard.go   # bubbletea TUI entry
├── internal/
│   ├── config/
│   │   ├── config.go     # AppConfig, ServerConfig, registry, Manager struct
│   │   └── settings.go   # ServerSettings (server-settings.json), GetSet helpers
│   ├── docker/
│   │   └── client.go     # Docker SDK wrapper: create/start/stop/remove/inspect/logs/pull
│   ├── server/
│   │   └── manager.go    # server lifecycle: Create/Start/Stop/Delete/Status/List/Logs/Update
│   ├── mods/
│   │   └── portal.go     # mod portal API: Search/GetMod/Download, ListInstalled, RemoveMod
│   ├── rcon/
│   │   └── client.go     # RCON client wrapper (gorcon/rcon)
│   └── tui/
│       ├── styles.go     # lipgloss color palette and style definitions
│       └── dashboard.go  # bubbletea Model: server table + detail panel, auto-refresh
├── .goreleaser.yaml       # goreleaser: linux/darwin/windows amd64+arm64
└── .github/workflows/
    └── release.yml        # GitHub Actions: trigger on v* tag
```

## Data Directory Structure
Default: `~/.factorio-manager/` (override with `--data-dir`)
```
~/.factorio-manager/
├── config.json              # AppConfig: username, token, default_image_tag
├── servers.json             # ServersRegistry: array of ServerConfig
└── servers/<name>/
    ├── config/
    │   ├── server-settings.json     # mounted at /factorio/config/
    │   ├── rconpw                   # auto-generated RCON password
    │   └── (other factorio configs)
    ├── mods/                        # mounted at /factorio/mods/
    └── saves/                       # mounted at /factorio/saves/
```

## Docker Container Conventions
- Container names: `factorio-<servername>`
- Volume mount: `<serverDir>:/factorio` (single volume for all factorio data)
- Labels: `factorio-server-manager=true`, `fsm.server=<name>`
- Default image: `factoriotools/factorio:stable`
- Game port: 34197/udp (configurable)
- RCON port: 27015/tcp (configurable)
- RCON password: auto-generated 32-char hex, stored in `config.json` and `config/rconpw`

## Key Design Decisions
- **Single volume mount**: The entire server directory is mounted at `/factorio`, matching the factoriotools/factorio image convention (config/, mods/, saves/ are all subdirs).
- **RCON password**: Auto-generated on server creation, stored in registry and written to `rconpw` file (picked up by the Docker image).
- **cfgManager global**: `cmd/root.go` initializes `cfgManager *config.Manager` in `PersistentPreRunE`, shared by all subcommands.
- **Error wrapping**: Use `fmt.Errorf("context: %w", err)` throughout.
- **No CGO**: `CGO_ENABLED=0` for fully static binaries in goreleaser.

## Libraries
| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/spf13/cobra` | v1.10+ | CLI framework |
| `github.com/docker/docker` | v28+ | Docker SDK |
| `github.com/docker/go-connections` | v0.7+ | Docker port bindings |
| `github.com/charmbracelet/bubbletea` | v1.3+ | TUI framework |
| `github.com/charmbracelet/lipgloss` | v1.1+ | TUI styling |
| `github.com/charmbracelet/bubbles` | v1.0+ | TUI components |
| `github.com/gorcon/rcon` | v1.4+ | RCON protocol |
| `golang.org/x/term` | v0.42+ | Secure password input |

## Mod Portal API
Base URL: `https://mods.factorio.com/api/mods`
- GET `/api/mods?q=<query>&page=1&page_size=20` — search
- GET `/api/mods/<name>/full` — full mod info including all releases
- Download: `https://mods.factorio.com<download_url>?username=<u>&token=<t>`
- Authentication required for download (username + token from factorio.com/profile)

## Releasing
```bash
git tag v1.0.0
git push origin v1.0.0
# GitHub Actions picks up the tag and runs goreleaser
```

Or locally:
```bash
goreleaser release --clean
goreleaser check     # validate .goreleaser.yaml
```

## TUI Dashboard Keybindings
| Key | Action |
|-----|--------|
| `j` / `↓` | Move selection down |
| `k` / `↑` | Move selection up |
| `s` | Start selected server |
| `S` | Stop selected server |
| `r` | Force refresh |
| `q` / `Ctrl+C` | Quit |

Auto-refreshes every 3 seconds.
