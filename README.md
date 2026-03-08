# dk

A lightweight Docker management CLI and web dashboard.

## Setup

Clone the repo and add an alias to your shell config:

```bash
alias dk='php /path/to/docker-tooling/bin/dk'
```

Requires PHP 8.1+ and Docker.

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
dk help         Show help
```

All commands support partial name matching. Omit the container name for interactive selection.

## Web Dashboard

```bash
dk web          # http://localhost:8080
dk web 9090     # custom port
```

Shows containers grouped by Compose project with start/stop/restart controls and clickable port links. Auto-refreshes every 5 seconds.
