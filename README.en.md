<h1 align="center">LunaFox</h1>

<p align="center">
  <b>Self-hosted Attack Surface Management (ASM) | Asset discovery and security automation for authorized testing and defensive research</b>
</p>

<p align="center">
  <a href="https://github.com/yyhuni/lunafox/stargazers"><img src="https://img.shields.io/github/stars/yyhuni/lunafox?style=flat-square&logo=github" alt="GitHub stars"></a>
  <a href="https://github.com/yyhuni/lunafox/network/members"><img src="https://img.shields.io/github/forks/yyhuni/lunafox?style=flat-square&logo=github" alt="GitHub forks"></a>
  <a href="https://github.com/yyhuni/lunafox/issues"><img src="https://img.shields.io/github/issues/yyhuni/lunafox?style=flat-square&logo=github" alt="GitHub issues"></a>
  <a href="https://github.com/yyhuni/lunafox/blob/main/LICENSE"><img src="https://img.shields.io/github/license/yyhuni/lunafox?style=flat-square" alt="License"></a>
</p>

<p align="center">
  <a href="#features">Features</a> •
  <a href="#quick-start">Quick start</a> •
  <a href="#documentation">Documentation</a> •
  <a href="#feedback-and-contributing">Feedback</a>
</p>

<p align="center">
  <sub>Keywords: ASM | attack surface management | asset discovery | authorized security testing | distributed scanning | global search | vulnerability management</sub>
</p>

**English** | [简体中文](README.md)

---
## Live demo

The [public demo](https://lunafox-demo.vercel.app/) shows the interface with mock data, requires no login, and is not connected to a real backend.

<p align="center">
  <img src="docs/images/en/overview.png" alt="Overview" width="24%">
  <img src="docs/images/en/workflows.png" alt="Scan workflows" width="24%">
  <img src="docs/images/en/scan-history.png" alt="Scan history" width="24%">
  <img src="docs/images/en/nodes.png" alt="Agent nodes" width="24%">
</p>

---
## Documentation

- [Public deployment guide](docs/public-deployment.md) - installation, day-2 operations, database, upgrades, and recovery
- [Release notes](release-notes/README.md) - changes in each release
- [Contributing](CONTRIBUTING.md) - issues, pull requests, and contribution conventions

---
## Features

### Built-in engines

The following are the default built-in engines; engines are distributed as Engine Packages and can be extended.

| Engine | Tools | Description |
|------|------|------|
| Subdomain discovery | Subfinder, PureDNS, MassDNS | Passive collection plus active enumeration |
| Port scan | Naabu | Custom port ranges |
| Website discovery | HTTPX | Status code, title, and technology detection |
| Fingerprinting | Finger | Multi-source fingerprint library |
| URL collection | Katana, Waymore, HTTPX | Historical data plus active crawling |
| Directory scan | FFUF | Calibration, recursion, and concurrency control |
| Vulnerability scan | Nuclei | Template-based vulnerability detection |
| Screenshots | Chromium, HTTPX | Screenshot and page metadata capture |

### Platform capabilities

| Capability | Description |
|------|------|
| Target management | Domain / IP / CIDR targets |
| Organizations | Multiple organizations and asset ownership |
| Global search | Expression syntax with multi-field queries |
| Vulnerability management | Review, retest, and asset context |
| Scheduled scans | Cron-based recurring tasks |
| Scan orchestration | Workflow composition, freezing, and history |
| Distributed execution | Multiple Agents with load-aware scheduling |
| Wordlists and fingerprints | Wordlists, fingerprint library, and Nuclei POC catalog |
| Notifications | Discord and Webhook |
| MCP integration | AI clients for investigation and constrained actions |
| System upgrade | In-app upgrade, progress, and recovery |

### Global asset search

- Multiple asset types: Website and Endpoint
- Expression syntax: `=` contains, `==` exact, `!=` not equal
- Logical composition: `&&` (AND) and `||` (OR)
- Fields: url, host, title, statusCode, tech, responseHeaders, hasScreenshot
- CSV export: stream the full result set

```bash
# basic search
host="api"
statusCode==200
tech="nginx"

# combinations
host="api" && statusCode==200
tech="vue" || tech="react"
```

### Visual interface

- Statistics: assets, scans, vulnerabilities, and Agent fleet status
- Live progress: scan progress, Engine execution state, and history
- Notifications: Discord and Webhook
- UI languages: Simplified Chinese and English

---
## Quick start

### Requirements

- Docker Engine with Linux container support and Docker Compose 2.24.0 or newer
- Bash 3.2 or newer for lifecycle scripts; on Windows, run Compose directly in PowerShell
- A free port for `PUBLIC_PORT`; the default is 443
- Outbound access to Docker Hub, GHCR, or the configured registry
- Disk space for images, database data, scan results, and logs

### Install

The public repository and every `lunafox-<version>.zip` release contain equivalent deployment snapshots.

**Option 1 - from the public repository**

```console
git clone https://github.com/yyhuni/lunafox.git
cd lunafox
./install.sh
```

**Option 2 - from a release archive**

```console
unzip lunafox-<version>.zip -d lunafox-<version>
cd lunafox-<version>
./install.sh
```

**Option 3 - direct Docker Compose**

```console
docker compose up -d
```

Options 1 and 2 run the installer, which validates arguments, preheats images, and waits for readiness. Option 3 skips those checks and is only recommended when you handle them yourself.

For a deployment reachable from other machines, set the public address and port:

```console
./install.sh --public-host luna.example.com --public-port 8443
```

The first start creates a self-signed certificate for `PUBLIC_HOST`; this release does not manage certificates. See [database environment variables](docs/public-deployment.md#database-environment-variables) and [configuration changes after the first start](docs/public-deployment.md#configuration-changes-after-first-start).

---
## Feedback and contributing

- Bugs or suggestions: open an [Issue](https://github.com/yyhuni/lunafox/issues)
- Want to contribute: read the [Contributing guide](CONTRIBUTING.md) first
- Security: do not publicly disclose unfixed vulnerabilities; contact the maintainers privately as described in the guide

---
## Security and compliance

1. LunaFox targets authorized security testing, defensive research, and management of infrastructure you own
2. You must have explicit authorization for the target systems, data, and environments
3. The default TLS certificate is self-signed; this release does not manage certificates
4. Agents use a long-lived bearer token; TLS identity, replay protection, rotation, and revocation are not implemented yet
5. This single-node deployment does not provide automatic backup, preserve-data rollback, or cross-node recovery

Do not use this project for unauthorized scanning, intrusion, or disruption.

---
## Star History

[![Star History Chart](https://star-history.dera.page/svg?repos=yyhuni/lunafox&type=Date)](https://star-history.dera.page/#yyhuni/lunafox&Date)

---
## License

Deployment files and first-party public Runtime/Engine source are [GPL-3.0-only](LICENSE); Engine Runtime Images and Engine Packages are public artifacts; third-party components keep their own licenses and attribution terms.

> **GENERATED / READ-ONLY PROJECTION.** Generated from the private LunaFox source; report defects against the matching release tag via [CONTRIBUTING.md](CONTRIBUTING.md).
