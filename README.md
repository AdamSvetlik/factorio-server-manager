# Factorio Server Manager

A CLI tool for managing multiple Factorio dedicated server instances via Docker.

## Features

- **Multi-server management** — create, start, stop, delete, and monitor multiple Factorio server instances
- **Docker-based** — uses the official [`factoriotools/factorio`](https://hub.docker.com/r/factoriotools/factorio) image
- **Interactive TUI dashboard** — live status table with keybindings (powered by bubbletea)
- **Save management** — list, copy, export, and delete save files per server
- **Mod management** — search, install, update, and remove mods via the Factorio mod portal API
- **Configuration management** — view and edit `server-settings.json` per server
- **RCON support** — send in-game commands from the CLI

## Installation

### From GitHub Releases

Download the latest binary for your platform from the [Releases](https://github.com/AdamSvetlik/factorio-server-manager/releases) page.

```bash
# Linux / macOS
chmod +x factorio-server-manager
sudo mv factorio-server-manager /usr/local/bin/
```

### Build from source

```bash
git clone https://github.com/AdamSvetlik/factorio-server-manager
cd factorio-server-manager
go build -o factorio-server-manager .
```

## Requirements

- Docker installed and running
- Go 1.21+ (for building from source)

## Quick Start

```bash
# Create a new server
factorio-server-manager server create myserver --port 34197 --desc "My Factorio Server"

# Start it
factorio-server-manager server start myserver

# Open the TUI dashboard
factorio-server-manager dashboard

# View logs
factorio-server-manager server logs myserver --follow

# Stop the server
factorio-server-manager server stop myserver
```

## Commands

### Server Management

```
server create <name>    Create a new server instance
server list             List all servers with status
server start <name>     Start a server
server stop <name>      Stop a server
server status <name>    Show detailed server status
server logs <name>      Stream server logs (--follow / -f)
server update <name>    Pull latest image & recreate container
server delete <name>    Remove server (--remove-data to delete files too)
server rcon <name> <cmd>  Execute an RCON command
```

### Save Management

```
save list <server>               List save files
save copy <file> <server>        Import a save file
save export <server> <save>      Export a save file (--out <dir>)
save delete <server> <save>      Delete a save file
```

### Mod Management

```
mod list <server>                List installed mods
mod search <query>               Search the mod portal
mod info <mod-name>              Show mod details and releases
mod install <server> <mod-name>  Download and install a mod (--version <ver>)
mod remove <server> <mod-file>   Remove an installed mod
mod update <server>              Update all mods to latest versions
```

Mod downloads require Factorio credentials:

```bash
factorio-server-manager auth login
```

### Configuration

```
config show <server>             Print server-settings.json
config set <server> <key> <val>  Set a config value
config edit <server>             Open config in $EDITOR
```

### Authentication

```
auth login     Save Factorio username and token
auth logout    Remove saved credentials
auth status    Check if credentials are set
```

Your token can be found at: https://factorio.com/profile

### Dashboard

```
factorio-server-manager dashboard
```

**Keybindings:**

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `s` | Start selected server |
| `S` | Stop selected server |
| `r` | Refresh |
| `q` | Quit |

## Data Directory

All server data is stored in `~/.factorio-manager/` by default.
Override with the global `--data-dir` flag.

```
~/.factorio-manager/
├── config.json           # Global config (credentials, defaults)
├── servers.json          # Server registry
└── servers/<name>/
    ├── config/           # server-settings.json, rconpw, etc.
    ├── mods/             # .zip mod files
    └── saves/            # .zip save files
```

## Releases

Releases are built automatically via GoReleaser + GitHub Actions when a `v*` tag is pushed.

```bash
git tag v1.0.0
git push origin v1.0.0
```

Binaries are published for:
- Linux: amd64, arm64
- macOS: amd64, arm64
- Windows: amd64

## License

MIT
