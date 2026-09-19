# LunaFox public deployment

> **GENERATED / READ-ONLY PROJECTION**

[简体中文](README.zh-CN.md)

This repository is generated from
[`yyhuni/lunafox-private`](https://github.com/yyhuni/lunafox-private). It
contains the reviewed Runtime source closure and the files used to build the
versioned Docker Compose deployment packages. Changes belong in the private
source repository and are projected through the protected release workflow.

## Install

Install Docker with Linux container support and Docker Compose 2.24.0 or newer.
The public `main` branch is a complete deployment snapshot, so a source checkout
starts directly:

```console
git clone https://github.com/yyhuni/lunafox.git
cd lunafox
./install.sh
```

`./install.sh` reads `.env`, runs one `docker compose up -d`, and waits until the
deployment is actually ready. The same checkout works without the scripts, which
is also the native Windows PowerShell path:

```console
git clone https://github.com/yyhuni/lunafox.git
cd lunafox
docker compose up -d
```

Every new GitHub Release also contains one equivalent deployment archive:

```console
unzip lunafox-<version>.zip -d lunafox-<version>
cd lunafox-<version>
./install.sh
```

Both paths include `.env`, `.env.example`, `compose.yaml`, the release manifest,
and the Engine inventory. Review `PUBLIC_HOST`, `PUBLIC_PORT`, and
`DATABASE_MODE` in `.env` before startup. First-party images default to Docker
Hub:

```dotenv
RELEASE_REGISTRY=docker.io
```

To use GHCR for the complete first-party Runtime and Engine closure, change it
before first startup:

```dotenv
RELEASE_REGISTRY=ghcr.io
```

Only `docker.io` and `ghcr.io` are accepted. LunaFox does not automatically
fall back or mix registries. Compose validates configuration and pulls missing
images as part of `docker compose up -d`. These commands are optional
diagnostics or prefetch steps:

```console
docker compose config --quiet
docker compose pull
docker compose ps
```

`PUBLIC_URL` is derived internally. `DB_USER=postgres` and `DB_NAME=lunafox`
are editable defaults. Embedded mode generates empty `DB_PASSWORD` and
`JWT_SECRET` values once and retains them in the `lunafox_config` volume. For an
existing remote PostgreSQL database, set `DATABASE_MODE=external`, `DB_HOST`,
`DB_SSLMODE`, and its real `DB_PASSWORD` before first startup; LunaFox applies
its schema but does not create the remote database or migrate existing data.

Windows users extract the Release ZIP and run the same Compose commands in
native PowerShell. A PowerShell downloader, WSL terminal, Docker context name
check, and Loki Docker logging plugin are not part of this deployment path.

Use the lifecycle scripts for everyday operation, or the equivalent Docker
Compose commands:

| Script | Equivalent command | Behaviour |
| --- | --- | --- |
| `./install.sh` | `docker compose up -d` | First start and safe re-run; keeps existing data and `.env`. |
| `./start.sh` | `docker compose up -d` | Starts an existing deployment. |
| `./restart.sh` | `docker compose up -d --force-recreate` | Recreates containers with the current configuration. |
| `./stop.sh` | `docker compose stop` | Stops every resident service; keeps all data. |
| `./status.sh` | `docker compose ps` plus readiness checks | Read-only summary; exits 0 only when fully ready. |
| `./logs.sh` | `docker compose logs --tail 200 --follow` | Read-only logs; forwards service names and Compose options. |
| `./uninstall.sh` | `docker compose down --remove-orphans` | Keeps volumes and `.env` unless `--purge --confirm` is given. |

`install.sh`, `start.sh`, and `restart.sh` print `SUCCESS`, the public HTTPS
address, and the `admin / admin` first-login hint only after every service, the
resident Agent, and the public endpoint are ready. Otherwise they print `FAILED`
with the failing service and leave the deployment in place for inspection. The
scripts need Bash 3.2 or newer and work from any working directory.

Docker Compose remains a complete lifecycle interface:

```console
docker compose stop
docker compose start
docker compose restart
docker compose down
```

`down` retains named volumes, including the generated configuration secrets.
Data deletion is a separate, explicit operator
action. LunaFox never automatically migrates or clears an older script-managed
deployment; inspect and back up old state before adopting a new package.

See [`docs/public-deployment.md`](docs/public-deployment.md) for initialization,
recovery, registry, certificate, and security boundaries.

## Release identity

The release workflow resolves Server, Frontend, Nginx, Agent, Bootstrap, and
Engine references to immutable cross-registry digests. It builds the unified
deployment package only after the final manifest passes verification, publishes
its SHA-256 record, and refuses to replace a conflicting asset with the same
release name.

The private release run ends after requesting protected public export merge. A
green private run means public handoff was requested; final publication status
comes from the public repository workflow and GitHub Release.

## Source and security scope

The exported Runtime source closure includes Frontend, Server, Contracts,
`engine-go`, Proto, first-party Extensions, Nginx, Bootstrap, and their reviewed
tests and build metadata. Agent source and private signing material remain
outside the public source boundary. The closed Agent artifact terms are in
[`NOTICE-CLOSED-ARTIFACTS.md`](NOTICE-CLOSED-ARTIFACTS.md).

The current Agent credential is still a long-lived eight-character hexadecimal
bearer. This release does not add Agent TLS certificate-chain identity, replay
protection, rotation, or revocation. TLS and credential hardening require a
separate security change. The deployment is for self-hosted use on
infrastructure the operator owns or controls.

Exported first-party Runtime source and deployment files are licensed under
SPDX `GPL-3.0-only`.

Engine Runtime Images and Engine Packages are public artifacts. First-party
Engine source is GPL-3.0-only. Included third-party components remain governed
by their applicable licenses and attribution terms.
