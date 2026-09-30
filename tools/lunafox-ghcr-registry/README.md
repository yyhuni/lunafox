# LunaFox GHCR Registry Worker

This is the formal, read-only OCI Distribution API proxy for public `yyhuni/lunafox-*` packages on `ghcr.io`. It is intentionally separate from `tools/ghcr-registry-proxy-poc/`, which remains available as the manual rollback Worker source.

The Worker accepts only `GET` and `HEAD` for `/v2/`, digest-qualified manifest/blob paths, and the exact Sigstore v0.3 `artifactType`-filtered Referrers path. Unfiltered first-party Referrers probes and Docker's legacy `sha256-<digest>` referrer fallback receive a local `404`, matching ordinary Docker compatibility probes without expanding the proxy surface. It has no storage binding, Cache API use, GitHub token, or client authentication. The Cloudflare hostname is a transport endpoint only: release identity and Cosign verification continue to use `ghcr.io`, while the signature bundle may be fetched through this filtered transport.

## Third-party closure

The Worker also serves only the release-frozen manifest digests for
`library/postgres`, `library/redis`, `grafana/loki`, and `grafana/alloy`. Each
entry is an explicit Docker Hub origin, repository, and digest mapping; tags,
other repositories, writes, and unrecognized query parameters are rejected
locally. Docker Hub access uses its anonymous, repository-scoped pull token.
Redirects are followed internally after removing that token.

These entries are LunaFox-reviewed fixed-digest content. Normal OCI/Docker
digest validation protects the delivered bytes, but this Worker does not claim
to authenticate the PostgreSQL, Redis, or Grafana publisher and does not turn a
LunaFox review into publisher signature verification.

## Local checks

```bash
pnpm install --frozen-lockfile
pnpm test
pnpm run typecheck
pnpm exec wrangler types
pnpm exec wrangler check startup
```

After deployment, use a real public fixed-digest LunaFox image and one of its blob digests:

```bash
REGISTRY_HOST=lunafox-ghcr-registry.<account-subdomain>.workers.dev \
IMAGE_REPOSITORY=yyhuni/lunafox-<package> \
IMAGE_DIGEST=sha256:<manifest-or-index-digest> \
BLOB_DIGEST=sha256:<blob-digest> \
pnpm run verify:local
```

The command runs Docker and ORAS fixed-digest pulls, a blob range request, and protocol rejection checks. Its result is evidence for the executing local network only; it does not establish availability for all networks in mainland China.

## GHCR visibility

The Worker intentionally uses no GitHub credential, so every `yyhuni/lunafox-*`
package it serves must be public. After a package is first created by the release
workflow, set its visibility to **Public** in its GitHub Packages settings. The
release workflow then proves this invariant with credential-free Docker and ORAS
pulls; a private package blocks release finalization instead of silently relying
on a GitHub token.

## Rollout and rollback

The protected public release workflow owns the production deployment. It runs
the Worker tests, typecheck, Wrangler startup check, `wrangler deploy`, and an
anonymous digest smoke against `docker.lunafox.cc.cd` before it publishes the
release channel or GitHub Release. The workflow stores
`cloudflare-worker-release-evidence.json` with the release tag, source
identities, canonical policy digest, Worker source digest, deployment tag,
Version ID, response statuses, and elapsed times. Cloudflare credentials are
available only to the `public-release` environment job.

For an independent first-time rollout, or to troubleshoot CI, use the local
checks below. Do not attach the custom domain until the generated `workers.dev`
hostname passes them:

1. Log in with `pnpm exec wrangler login`, confirm it with `pnpm exec wrangler whoami`, then deploy `pnpm exec wrangler deploy`.
2. Run the local checks above against the generated `workers.dev` hostname.
3. In Cloudflare Dashboard, open **Workers & Pages -> lunafox-ghcr-registry -> Settings -> Domains & Routes**, then attach `docker.lunafox.cc.cd`.
4. Run the same verification command with `REGISTRY_HOST=docker.lunafox.cc.cd`.

Each production Worker version is a hard cut: its third-party allowlist is
exactly the current release's `third-party-image-policy.json`. A digest omitted
from that release, including a digest from the immediately preceding release,
is expected to receive local `403`; there is no compatibility window. Existing
installations must update before they can pull the new third-party images.

If a deployment or smoke step fails, the workflow stops before channel, ZIP,
tag, and Release publication. Retrying the same run reuses the immutable
release artifacts. To undo a successful deployment, use the recorded Version
ID with `pnpm exec wrangler rollback <version-id> --yes`, then verify the custom
domain again. `rollback` takes the Version ID as its positional argument; use
the installed CLI's `--help` output if an operator is using a newer Wrangler
syntax. The POC Worker remains a separate, manually recoverable rollback
target; CI never reassigns the custom domain automatically.

To roll back a custom-domain issue, in the same **Domains & Routes** screen remove `docker.lunafox.cc.cd` from the formal Worker and attach it back to `lunafox-ghcr-registry-proxy-poc`. The POC source and its deployment are not changed by this project.
