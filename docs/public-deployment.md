# Public Docker Compose deployment

> **GENERATED / READ-ONLY** - edit the private source and regenerate this
> projection through the protected release workflow.

## Installation sources

The public `main` branch and each release's single `lunafox-<version>.zip`
contain equivalent deployment snapshots. Each snapshot includes `compose.yaml`,
`.env`, `.env.example`, the verified final release manifest, a dual-registry
Engine inventory, the Loki and Alloy configuration, the default fingerprint
corpus, required wordlists, and the Bash lifecycle scripts with their shared
helper. Product and Engine images are digest-qualified; PostgreSQL, Redis, Loki,
and Alloy are pinned by multi-platform manifest digest.

Keep the extracted directory: `.env` is the installation's host-owned
configuration and relative resource paths resolve from this directory. Review
`PUBLIC_HOST` and `PUBLIC_PORT`; the internal HTTPS `PUBLIC_URL` is derived from
them. Once you write `DB_PASSWORD` or `JWT_SECRET`, restrict the file with
`chmod 600 .env`; the lifecycle scripts enforce the same mode before they change
anything.

### Install from the public repository

In a Bash environment, review `.env` and run the lifecycle script. It starts the
deployment with one `docker compose up -d` and waits until the deployment is
fully ready:

```console
git clone https://github.com/yyhuni/lunafox.git
cd lunafox
./install.sh
```

`docker compose up -d` is equally supported and is the native Windows
PowerShell entry point:

```console
git clone https://github.com/yyhuni/lunafox.git
cd lunafox
docker compose up -d
```

### Install from a release

Download `lunafox-<version>.zip` and its SHA-256 metadata from the matching
GitHub Release, verify it when required by local policy, and extract it into a
permanent directory:

```console
unzip lunafox-<version>.zip -d lunafox-<version>
cd lunafox-<version>
./install.sh
```

The extracted ZIP also supports the direct Compose path:

```console
unzip lunafox-<version>.zip -d lunafox-<version>
cd lunafox-<version>
docker compose up -d
```

Windows operators extract the same ZIP and run the Compose command in native
PowerShell.

### Select a Registry

Both installation sources default to Docker Hub in `.env`:

```dotenv
RELEASE_REGISTRY=docker.io
```

To use GHCR, change the value before first startup:

```dotenv
RELEASE_REGISTRY=ghcr.io
```

Only `docker.io` and `ghcr.io` are accepted. The selected value applies to the
entire first-party Runtime, Agent, Engine, bootstrap, and upgrader closure.
LunaFox never falls back to the other Registry and never mixes the two within
one deployment. After a successful in-place upgrade, the persisted override is
bound to its original Registry; changing Registry requires a fresh deployment
or an explicit migration that preserves application data and configuration.

`DATABASE_MODE=embedded` is the default and enables the bundled PostgreSQL
service. `DB_PORT=5432`, `DB_USER=postgres`, and `DB_NAME=lunafox` are editable
defaults. Leave `DB_HOST` and `DB_SSLMODE` empty in this mode. An empty
`DB_PASSWORD` generates a random value on first initialization.

To use a database that already exists outside this Compose project, set all of
the following before the first start:

```dotenv
DATABASE_MODE=external
DB_HOST=database.example
DB_PORT=5432
DB_USER=postgres
DB_NAME=lunafox
DB_SSLMODE=require
DB_PASSWORD='the existing remote database password'
```

Keep `COMPOSE_PROFILES=${DATABASE_MODE:-embedded}` unchanged so the selected
mode controls the service graph. External mode requires a reachable PostgreSQL
server, an existing account and database, and permission to apply LunaFox schema
migrations. LunaFox does not create those resources or copy data from the
bundled database. `DB_SSLMODE` accepts `disable`, `allow`, `prefer`, `require`,
`verify-ca`, or `verify-full`; `verify-ca` and `verify-full` use the container's
system trust roots. Custom CA file mounting is not supported by this release.

`JWT_SECRET` remains optional and is generated when empty. Configuration inputs
reach only the initializer as Compose secret files and must not be committed or
printed in support logs.

## Start and lifecycle

Docker with Linux container support and Docker Compose 2.24.0 or newer are the
host prerequisites. From the extracted directory run:

```console
docker compose up -d
docker compose ps
```

Compose validates the configuration and pulls missing images during
`docker compose up -d`. `docker compose config --quiet` is an optional
configuration diagnostic, and `docker compose pull` is an optional prefetch
step; neither is required for installation.

On Windows, run these commands directly in PowerShell. On macOS, Linux, Docker
Desktop, or OrbStack, use the Docker endpoint already selected by the Docker
CLI. LunaFox does not inspect or allowlist the host OS, distribution, Docker
brand, or context name. Docker reports connection, platform, bind path, and
published-port errors itself.

The ordinary lifecycle is:

```console
docker compose stop
docker compose start
docker compose restart
docker compose down
```

### Lifecycle scripts

Every snapshot also ships seven small Bash entry points at the deployment root
plus the shared helper `lunafox-lifecycle.sh`. They wrap the same Compose
commands, add the checks Docker cannot express, and never keep a second
deployment state:

| Script | Equivalent Compose command | Behaviour |
| --- | --- | --- |
| `./install.sh` | `docker compose up -d` | First start and safe re-run. Keeps an existing `.env` and every named volume. |
| `./start.sh` | `docker compose up -d` | Starts an existing deployment. |
| `./restart.sh` | `docker compose up -d --force-recreate` | Applies the current `.env` and `compose.override.yaml` to every container. |
| `./stop.sh` | `docker compose stop` | Stops every resident service and keeps all data. |
| `./status.sh` | `docker compose ps` plus the readiness checks | Read-only summary; exits 0 only for a fully ready deployment. |
| `./logs.sh` | `docker compose logs --tail 200 --follow` | Read-only log access that forwards service names and Compose options. |
| `./uninstall.sh` | `docker compose down --remove-orphans` | Removes containers and the project network; keeps volumes and `.env`. |

`install.sh`, `start.sh`, and `restart.sh` report success only after the one-shot
tasks, core service health, auxiliary service stability, the resident Agent's
claim-ready state, and the public HTTPS endpoint all pass. `install.sh` allows
fifteen minutes and the other two allow five; `LUNAFOX_READY_TIMEOUT_SECONDS`
overrides the window for one invocation.

A failure prints `FAILED`, names the failing service or check, and leaves the
deployment in place: containers, volumes, and configuration stay as compose up
created them, and nothing is rolled back. The printed diagnostics keep that
order — conclusion, stage, preserved scene, next commands — and point at
`./logs.sh <service>` when a service was identified and `./logs.sh` otherwise.

`install.sh` describes the run it classified instead of always narrating a first
start: a directory without any LunaFox data is a first start; a directory whose
volumes exist without containers recreates the containers while keeping its
named volumes and installed engines, and is not an upgrade; a running
deployment is started with its configuration and volumes preserved. When such a
recreated deployment fails a first-start task such as `bootstrap`, the footer
adds one optional last line after the `logs.sh` and `status.sh` hints: if the
preserved data can be discarded, `./uninstall.sh --purge --confirm` is the only
way to delete the named volumes and start over.

`install.sh` accepts only `--help`, reads every setting from `.env`, and runs
`docker compose up -d` exactly once. It never pulls, builds, removes containers,
or deletes data. `uninstall.sh` keeps all data unless you pass
`--purge --confirm`, which deletes only the named volumes declared by the
current `compose.yaml` after verifying their LunaFox ownership.

The scripts need Bash 3.2 or newer and work from any working directory. Windows
keeps the direct Compose commands in PowerShell.

### Deployment lock and recovery

A lifecycle mutation and an Upgrade Operation never overlap. Both take a
deployment-level lock in `.lunafox-lifecycle.lock`, and `./status.sh` reports it
without waiting:

```console
./status.sh
LunaFox:   address: https://localhost:443
LunaFox:   lock: upgrade recovery fence (operation <id>)
...
LunaFox deployment status: starting
```

The lock is never reclaimed automatically, because a slow operation must not
look like a dead one. Confirm that no lifecycle command and no upgrade is
running, then remove the directory by hand:

```console
cat .lunafox-lifecycle.lock/owner
rm -rf .lunafox-lifecycle.lock
```

Direct `docker compose` commands cannot take this lock, so they are outside the
mutual exclusion. The upgrade journal in `lunafox_upgrade_state` remains the
source of truth for an interrupted upgrade.

The internal Agent and restricted upgrader are Compose services, so all four
commands include them.
`down` removes containers and the project network while retaining named
volumes, including `lunafox_upgrade_state`. Do not add `--volumes` unless
permanent state deletion is intended and
has been backed up. No command in this deployment automatically migrates,
resets, or deletes an older deployment.

## Initialization

Compose expresses initialization through one-shot services:

1. `config-init` validates the database mode, the derived `COMPOSE_PROFILES`
   value, and the connection inputs, generates or adopts the permitted secrets,
   and persists the database mode, database password, and JWT secret as
   mode-0600 files in `lunafox_config`.
2. `agent-preflight` uses the released Agent image to verify Docker API,
   exact named-volume identities, volume-subpath isolation, and read-only
   execution mounts through a real sibling container round trip.
3. `cert-init` creates or validates the certificate and private key in
   `lunafox_ssl`.
4. In embedded mode, PostgreSQL starts after configuration initialization and
   `migrate` waits for its health. In external mode, the bundled PostgreSQL
   service is absent and `migrate` connects directly to the configured host.
5. `bootstrap` waits for both preflight and migration, installs the verified
   Engine inventory and default resources, then creates or validates the
   internal Agent credential.
6. Server starts only after bootstrap succeeds. Agent starts after both
   bootstrap and Server health succeed; Nginx starts after certificate, Server,
   and Frontend readiness.

Repeated startup reuses the files in `lunafox_config`. After a successful first
initialization, changing `DATABASE_MODE` fails before any database consumer
starts. A later non-empty `DB_PASSWORD` or `JWT_SECRET` that differs from the
persisted value also fails; Compose does not rotate live credentials. An older
configuration volume without a mode record adopts embedded mode by default.
Adopting external mode from such a volume requires an explicit `DB_PASSWORD`
that matches the persisted value.

For embedded mode, back up `lunafox_config` with `lunafox_postgres`. For external
mode, back up `lunafox_config` together with the external database using its
operator-approved procedure. Deleting only the configuration volume loses the
password needed to connect to the selected database.

The credential is a mode-0600 regular JSON file in `lunafox_agent_state` and is
never passed through the host environment or Docker container metadata.
Registration, database binding, and credential publication are serialized.
Repeated successful `docker compose up -d` reuses the same registered Agent.
Missing, malformed, permissive, partially published, deleted, or database-
inconsistent identity state fails with an explicit repair error and never
silently registers another Agent.

Default resources follow the same fail-closed rule. A fresh state imports the
packaged fingerprint corpus and wordlists once. Later bootstrap runs validate
the original records, files, hashes, sizes, descriptions, and tags while
leaving additional user resources untouched. Missing, partially deleted, or
changed defaults fail with an explicit repair error; bootstrap never fills in
or overwrites that state. An initialized corpus that was fully cleared is not
silently reseeded.

The preflight checks the actual Docker socket, Linux execution node,
named-volume subpaths, and read-only execution mounts before business bootstrap.
The resident Agent checks the socket, platform, API version, and exact volume
identities again when it starts. Unsupported capability fails explicitly; there
is no weaker whole-volume fallback. Server never receives the Docker socket.

The Compose-managed Agent has in-container self-update disabled. The restricted
upgrader updates its fixed `agent` service from the release Manifest. Existing
remote Agent installation and `update_required` self-update behavior are
unchanged.

## System updates

The release package fixes its `stable` or `canary` channel and public metadata
source inside `compose.yaml`. There is no host update script: version changes go
through the Upgrade Operation and the restricted Compose upgrader, and the
lifecycle scripts neither copy nor bypass that state machine. The selected Registry comes from `.env` before
first startup and is then preserved by upgrades. The Server fetches
the current schema-v3 channel record over HTTPS, validates its bounded Manifest
path and raw SHA-256, then caches the immutable Manifest in
`lunafox_upgrade_state`. It only offers a candidate whose semantic version is
newer than the running release.

An administrator can check, confirm, and start an eligible update through the
existing frontend update control. Server sends only the Operation ID, fixed
action, and Manifest digest through the shared Unix Socket. The Compose-managed
upgrader continues if the browser closes or Server is recreated. It has no
network port and uses a fixed project, file set, command argv, service allowlist,
and `--no-deps` recreation. Server never receives the Docker Socket.

The upgrader's read-write Docker Socket grants control of the local Docker
daemon. The service also mounts the extracted package directory so it can read
`compose.yaml` and `.env` and atomically install `compose.override.yaml` after
service, migration, Agent, health, and digest verification. Keep that override
with the extracted directory: ordinary `docker compose up -d`, `start`, and
`restart` automatically retain the confirmed image and version target without
rewriting database, public address, or secret settings.

The completion receipt proves only that the bounded Compose deployment work
finished. Server reconciles it with the database Operation, journal, migration,
service, API, and Agent evidence before reporting success. A migration failure
or uncertain outcome requires recovery; an Agent timeout requires attention.
The current `disposable-development` policy still provides no preserve-data
rollback or automatic backup.

Automatic update supports compatible image-only releases. A release that
changes the Compose structure, named volumes, initialization resources, or the
upgrader protocol must declare itself incompatible; download its new deployment
ZIP, preserve the documented state, review its README, and run Compose from the
new extracted directory. The running release must satisfy the Manifest's
`upgrade.compatibilityRange`; otherwise the update remains visible but cannot
create an automatic Upgrade Operation. This is also the manual fallback if the
host cannot expose its local Docker Socket to the upgrader container.

## Logs

All resident services use Docker's `json-file` driver with `max-size=10m` and
`max-file=3`. `./logs.sh` follows every service with the same bounded output;
pass a service name or a Compose logs option to narrow it, for example
`./logs.sh server`. The ordinary Alloy container reads only labelled LunaFox Server
and internal Agent output through the Docker socket and sends it to the local
Loki service. Alloy stores read positions in `lunafox_alloy`, so collector
restart does not replay the entire retained log set.

The collector preserves the existing selectors
`{component="server",container_name="lunafox-server"}` and
`{agent_id="<id>",container_name="lunafox-agent"}`. It does not collect its own
output. No Loki Docker plugin is installed or required. Docker socket access is
limited to the restricted upgrader, Compose Agent, its one-shot capability
preflight, and the collector plus development-only tooling; the resident Server
has no Docker control path.

## Certificates and recovery

`cert-init` creates a self-signed RSA certificate with a DNS or IP SAN for
`PUBLIC_HOST`. Existing certificate files must be restricted regular files,
unexpired, match the configured host, and share the private key. Partial or
invalid state fails rather than being overwritten. This release does not add
ACME, user certificate import, or automatic renewal.

Use `docker compose logs config-init agent-preflight migrate bootstrap cert-init agent upgrader` to
diagnose a failed one-shot or Agent startup. Correct configuration or explicitly
repair the named volume state, then repeat `docker compose up -d`. Nginx readiness probes use `127.0.0.1` to match its IPv4 listener even on
hosts where `localhost` resolves to IPv6 first. For ordinary
initialization, Compose exit status, dependency conditions, container state,
and service health are the deployment status. Upgrade journal and completion
receipt files live only in `lunafox_upgrade_state` and are reconciled by the
Upgrade Operation API.

When upgrading from a package that stored credentials only in `.env`, keep the
existing non-empty `DB_PASSWORD` and `JWT_SECRET` for the first start of the new
package. `config-init` copies them into `lunafox_config`. After that migration,
empty inputs reuse the persisted files. A conflicting value fails explicitly;
online password and JWT rotation are outside this deployment workflow. Moving
an existing installation between embedded and external PostgreSQL requires a
separate, operator-managed data migration and new configuration state.

Administrator password reset remains available through the service command:

```console
docker compose exec server resetadmin
```

## Release and security boundary

This is a single-node Compose deployment. It does not provide rolling upgrade,
automatic backup, preserve-data rollback, or cross-node recovery. The current
`disposable-development` migration baseline does not promise compatibility with
existing persisted deployments. Release generation and contract checks use
isolated fixtures; they do not touch operator volumes.

The current Agent authentication credential remains a long-lived
eight-character hexadecimal bearer. Agent TLS certificate-chain identity,
replay protection, rotation, and revocation are deferred. TLS and credential
hardening require a separate security change. The package is for self-hosted
use on infrastructure the operator owns or controls; the closed Agent artifact
remains subject to `NOTICE-CLOSED-ARTIFACTS.md`.

Exported first-party Runtime source and deployment files are SPDX
`GPL-3.0-only`.

Engine Runtime Images and Engine Packages are public artifacts. First-party
Engine source is GPL-3.0-only. Included third-party components remain governed
by their applicable licenses and attribution terms.
