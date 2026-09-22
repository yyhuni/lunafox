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

## Login and initial credentials

After a fresh deployment is ready, open the configured LunaFox URL and sign in
with the default administrator account:

| Username | Password |
| --- | --- |
| `admin` | `admin` |

Change the password from the account settings after the first login.

If the administrator password is forgotten, run this command from the deployment
directory:

```console
docker compose exec server resetadmin
```

The command prints a new password once and invalidates existing administrator
sessions.

## Env file

Before the first start, review `PUBLIC_HOST`, `PUBLIC_PORT`, `DATABASE_MODE`, and
`RELEASE_REGISTRY` in `.env`. First-party images use Docker Hub by default; use
GHCR explicitly when needed:

```dotenv
RELEASE_REGISTRY=docker.io
# RELEASE_REGISTRY=ghcr.io
```

After the first successful start, `DATABASE_MODE`, `DB_PASSWORD`, and
`JWT_SECRET` are persisted in `lunafox_config`. Changing them to a different
non-empty value in `.env` does not migrate or rotate a running deployment and
fails fast; empty credential inputs reuse the persisted values. See the
[configuration-change guidance](docs/public-deployment.md#configuration-changes-after-first-start)
for recovery and new-deployment steps.

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
