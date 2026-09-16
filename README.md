# LunaFox public deployment

> **GENERATED / READ-ONLY PROJECTION**

This repository is generated from
[`yyhuni/lunafox-private`](https://github.com/yyhuni/lunafox-private). It
contains the reviewed Runtime source closure and the files used to build the
versioned Docker Compose deployment packages. Changes belong in the private
source repository and are projected through the protected release workflow.

## Install

Install Docker with Linux container support. On macOS or Linux, a public source
checkout can prepare the matching immutable deployment package:

```console
git clone --branch <release-tag> --depth 1 https://github.com/yyhuni/lunafox.git
cd lunafox
./prepare-deployment.sh
cd .lunafox-deployment
```

For a checkout that is not at a release tag, select the version explicitly:

```console
./prepare-deployment.sh --version <release-tag>
```

The command defaults to the Docker Hub package. Use `--registry ghcr` to select
the complete GHCR closure explicitly. It verifies the package SHA-256 from the
matching GitHub Release and never falls back between registries. The root
`compose.yaml` is a release template and is not directly deployable by copying
`.env.example`; use the prepared directory.

Alternatively, including on Windows, download one deployment ZIP from the
matching LunaFox GitHub Release and extract it into a permanent directory:

- `lunafox-<version>-dockerhub.zip` uses Docker Hub for the complete LunaFox
  image and Engine closure.
- `lunafox-<version>-ghcr.zip` uses GHCR for the complete LunaFox image and
  Engine closure.

Review `PUBLIC_HOST`, `PUBLIC_PORT`, and `DATABASE_MODE` in the deployment
directory's `.env`, then run with Docker Compose 2.24.0 or newer:

```console
docker compose up -d
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

Use Docker Compose for the resident lifecycle:

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
Engine references to immutable digests. It builds both deployment packages only
after the final manifest passes verification, publishes their SHA-256 records,
and refuses to replace a conflicting asset with the same release name.

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
