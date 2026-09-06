# Closed Agent artifact notice

`lunafox-private` is the sole development and binary-signing authority for the
closed Agent artifact. The public repository packages that verified binary and
owns the final product release. This notice applies only to Agent source, Agent
credentials, private signing/release material, and the resulting closed Agent
artifact.

Engine Runtime Images and Engine Packages are public artifacts.

First-party Engine source is GPL-3.0-only. Included third-party components
remain governed by their applicable licenses and attribution terms. This notice
does not limit GPL-3.0-only or applicable third-party license rights for Engine
Runtime Images or Engine Packages.

For the closed Agent artifact only, the recipient receives a limited,
non-exclusive runtime grant to download the official artifact, install it, run
it on infrastructure it owns or controls, and retain internal backups for
operation, recovery, and disaster recovery. That grant does not provide private
Agent source access, private signing material, publication rights for the
closed Agent artifact, public/third-party redistribution, external registry
forwarding, SaaS or hosted operation, managed-service resale, or white-label
rights.

## Canonical identity

The canonical Compose projection, release manifest, channel record, and release
evidence are published under [`yyhuni/lunafox`](https://github.com/yyhuni/lunafox).
For the closed Agent artifact, Docker Hub and GHCR references must use the
recorded immutable digest. A mirror or derived Agent artifact is not canonical
and must not claim official support. This official-support identity does not
restrict the public Engine artifact licenses.

## Internal mirrors and cache

An organization may use a pull-through cache, Registry mirror, or isolated copy
of the closed Agent artifact only on infrastructure it owns or controls. The
copy must preserve the complete manifest/blob set for the exact digest, the
original signature, provenance, SBOM, license, and attribution. Access must
remain organization controlled. The official public path consumes immutable
digests and does not install or run Cosign; retaining mirror evidence is not a
claim of automatic user-side signer verification. This permission does not
authorize public or third-party redistribution, modified repackaging, or
commercial hosting of the closed Agent artifact.

## Derived internal packages

An internal derived closed Agent artifact may add organization
configuration/layers, but must not alter official executable bytes or remove
proprietary notices. It must use the organization's name, version, and new
digest; be labelled non-canonical and unsupported; record the base official
ref/digest; and retain the original signature, provenance, SBOM, license, and
attribution. The official verifier rejects a derived ref as an official Agent
artifact. External publication requires written authorization.

## Redistribution and hosted services

Standalone redistribution of the closed Agent artifact outside the authorized
organization boundary needs prior written authorization identifying the
recipient, channel, term, exact artifact/digest, and support/trademark limits.
Provider-controlled SaaS, hosting, managed operation, white-label service, or
resale access involving the closed Agent artifact needs a separate written
commercial agreement covering tenants, deployment location, data/security
responsibilities, support/SLA, metering/fees, marks, and termination/revocation.
Helping a customer operate the software on the customer's own controlled
infrastructure does not itself grant closed Agent artifact redistribution
rights.

The GPL-3.0-only license for public source and first-party Engine source does
not grant closed Agent artifact or commercial rights. Conversely, the closed
Agent artifact terms do not narrow rights granted for public Engine artifacts.
