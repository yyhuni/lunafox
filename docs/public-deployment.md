# Public Docker Compose deployment

> **GENERATED / READ-ONLY** - edit the private source and regenerate this
> projection through the protected release workflow.

## Deployment package

Each public release contains two complete, versioned ZIP files. Choose exactly
one registry for a deployment:

- `lunafox-<version>-dockerhub.zip`
- `lunafox-<version>-ghcr.zip`

Each package contains `compose.yaml`, `.env`, `.env.example`, the verified final
release manifest, a registry-specific Engine inventory, the Loki and Alloy
configuration, the default fingerprint corpus, and required wordlists. Product
and Engine images are digest-qualified; PostgreSQL, Redis, Loki, and Alloy are
also pinned by multi-platform manifest digest in Compose. Registry candidates
are never mixed and there is no automatic fallback.

Keep the extracted directory: `.env` is the installation's host-owned
configuration and relative resource paths resolve from this directory. Review
`PUBLIC_HOST` and `PUBLIC_PORT`; the internal HTTPS `PUBLIC_URL` is derived from
them.

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

The internal Agent is a Compose service, so all four commands include it.
`down` removes containers and the project network while retaining named
volumes. Do not add `--volumes` unless permanent state deletion is intended and
has been backed up. No command in this deployment automatically migrates,
resets, or deletes an older deployment.

## Initialization

Compose expresses initialization through one-shot services:

1. `config-init` validates the database mode and connection inputs, generates or
   adopts the permitted secrets, and persists the database mode, database
   password, and JWT secret as mode-0600 files in `lunafox_config`.
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

The Compose-managed Agent has in-container self-update disabled. Upgrade its
digest by using a newer deployment package and Compose. Existing remote Agent
installation and self-update behavior are unchanged.

## Logs

All resident services use Docker's `json-file` driver with `max-size=10m` and
`max-file=3`. The ordinary Alloy container reads only labelled LunaFox Server
and internal Agent output through the Docker socket and sends it to the local
Loki service. Alloy stores read positions in `lunafox_alloy`, so collector
restart does not replay the entire retained log set.

The collector preserves the existing selectors
`{component="server",container_name="lunafox-server"}` and
`{agent_id="<id>",container_name="lunafox-agent"}`. It does not collect its own
output. No Loki Docker plugin is installed or required. Docker socket access is
limited to the Compose Agent, its one-shot capability preflight, and the
collector plus development-only tooling; the resident Server has no Docker
control path.

## Certificates and recovery

`cert-init` creates a self-signed RSA certificate with a DNS or IP SAN for
`PUBLIC_HOST`. Existing certificate files must be restricted regular files,
unexpired, match the configured host, and share the private key. Partial or
invalid state fails rather than being overwritten. This release does not add
ACME, user certificate import, or automatic renewal.

Use `docker compose logs config-init agent-preflight migrate bootstrap cert-init agent` to
diagnose a failed one-shot or Agent startup. Correct configuration or explicitly
repair the named volume state, then repeat `docker compose up -d`. There is no receipt
or hidden host lifecycle database; Compose exit status, dependency conditions,
container state, and service health are the deployment status.

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
