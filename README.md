# dk

A lightweight Docker management CLI and web dashboard.

## Setup

Clone the repo, build, and add an alias to your shell config:

```bash
cd docker-tooling
go build -o dk ./cmd/dk/
```

```bash
alias dk='/path/to/docker-tooling/dk'
```

Requires Go 1.21+ (build only) and Docker.

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
dk web          Launch web dashboard (default port 8080)
dk web stop     Stop the web dashboard
dk help         Show help
```

All commands support partial name matching. Omit the container name for interactive selection.

## Web Dashboard

```bash
dk web          # starts background server on http://localhost:8080
dk web 9090     # custom port
dk web stop     # stop the server
```

Runs as a background daemon — no terminal window needed. Shows containers grouped by Compose project with start/stop/restart controls and clickable port links. Auto-refreshes every 5 seconds.
