# LunaFox Deployment

> **GENERATED / READ-ONLY PROJECTION**

[简体中文](README.zh-CN.md)

## Install

Install Docker with Linux container support and Docker Compose 2.24.0 or newer.

### Option 1: Repository

```console
git clone https://github.com/yyhuni/lunafox.git
cd lunafox
./install.sh
```

### Option 2: Release archive

Download `lunafox-<version>.zip`, then run:

```console
unzip lunafox-<version>.zip -d lunafox-<version>
cd lunafox-<version>
./install.sh
```

### Option 3: Direct Docker Compose

From a checkout or extracted archive, run:

```console
docker compose up -d
```

## Env file

Before the first start, review `PUBLIC_HOST`, `PUBLIC_PORT`, `DATABASE_MODE`, and
`RELEASE_REGISTRY` in `.env`. First-party images use Docker Hub by default; use
GHCR explicitly when needed:

```dotenv
RELEASE_REGISTRY=docker.io
# RELEASE_REGISTRY=ghcr.io
```

## Daily operations

| Script | Equivalent command | Behavior |
| --- | --- | --- |
| `./install.sh` | `docker compose up -d` | First start and safe re-run; keeps existing data and `.env`. |
| `./start.sh` | `docker compose up -d` | Starts an existing deployment. |
| `./restart.sh` | `docker compose up -d --force-recreate` | Recreates containers with the current configuration. |
| `./stop.sh` | `docker compose stop` | Stops all resident services; keeps all data. |
| `./status.sh` | `docker compose ps` plus readiness checks | Read-only summary; exits 0 only when fully ready. |
| `./logs.sh` | `docker compose logs --tail 200 --follow` | Read-only logs; forwards service names and Compose options. |
| `./uninstall.sh` | `docker compose down --remove-orphans` | Keeps volumes and `.env` unless `--purge --confirm` is given. |
