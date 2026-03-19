# dk

A lightweight Docker management CLI and web dashboard.

## Installation

Download the latest binary from the [releases page](https://github.com/jimbarrett/dk/releases), then:

```bash
chmod +x dk_*
mv dk_* ~/.local/bin/dk
```

If `~/.local/bin` is not on your PATH, add this to your shell profile:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Requires Docker.

### Building from source

```bash
git clone git@github.com:jimbarrett/dk.git
cd dk
make build
cp dk ~/.local/bin/dk
```

### Updating

```bash
dk update
```

## CLI Usage

```
dk              Status overview (running containers, image count, disk)
dk ls           List containers (running by default)
dk ls -a        List all containers (including stopped)
dk ls -g        List grouped by Compose project
dk start        Start a container
dk stop         Stop a container
dk restart      Restart a container
dk sh           Shell into a container (auto-detects bash/sh)
dk logs         Tail container logs (-f to follow)
dk ports        Show port mappings
dk rm           Remove a container (-f to force)
dk rmi          Remove an image (-f to force)
dk clean        Prune stopped containers, dangling images, unused volumes
dk web          Launch web dashboard (auto-selects port starting at 10100)
dk web stop     Stop the web dashboard
dk version      Show version and check for updates
dk update       Update to the latest version
dk help         Show help
```

All commands support partial name matching. Omit the container name for interactive selection.

## Web Dashboard

```bash
dk web          # starts background server (auto-selects port starting at 10100)
dk web 9090     # start on a specific port
dk web stop     # stop the server
```

Runs as a background daemon — no terminal window needed. Auto-selects the first available port starting at 10100 (scans up to 100 ports). Shows containers grouped by Compose project with start/stop/restart controls and clickable port links. Auto-refreshes every 5 seconds.
