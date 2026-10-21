# Directory Scan Engine

`engine.lunafox.directory_scan` is a first-party Engine for Domain, IP, and
IPv4 CIDR Targets. It consumes the finalized current-Scan `websiteURLs` fact
product and reports strict `asset.directory.v1` observations through the
generated Engine API v2 Facade.

## Ownership Boundary

Server decides Target applicability, freezes manifest-derived configuration,
resolves the selected wordlist, and produces the finalized `websiteURLs`
input. Agent eagerly materializes the wordlist, lazily materializes a requested
Registry input into the fixed read-only inputs directory, mounts the writable workspace,
and starts the Runtime Image. Directory owns Target baseline expansion,
candidate ordering, FFUF semantics and process execution, result parsing,
bounded exact-URL deduplication, and lifecycle progress.

The Engine consumes only generated values: `Execution.Target`,
`Execution.Input.WebsiteURLs.Path(ctx)`, `Execution.Config.Ffuf`,
`Execution.Workspace`, and `Execution.Results.Directories`. It does not query
Server persistence, request resources at runtime, or add a Directory-specific
Agent or Server execution channel. The generated Results port is an authoring
surface, not manifest-based result authorization.

## Manifest Contract

The manifest declares one independently default-enabled `ffuf` section. The
builtin Workflow keeps the outer Directory Step disabled until a user enables
it. Its resource-backed `wordlist` parameter defaults to the preferred Catalog
name `dir_default.txt`; unavailability fails through the shared resource
lifecycle rather than selecting another wordlist.

Wordlist entries are relative paths in the same convention as SecLists and
other directory-bruteforce scanners: the Engine stages an Engine-private
normalized copy, removes at most one leading `/` from each entry, and passes the
copy as `-w`. A `//admin` entry keeps its second slash, a lone `/` stays `/`,
and empty lines, `CRLF`, a final line without a newline, and zero-byte inputs
remain legal. The selected source file is never rewritten.

`match-codes` and `delay` remain opaque strings through Frontend, Server,
Workflow persistence, and saved plans. Directory validates their FFUF-specific
syntax before reading candidates or launching a process and passes accepted
original values unchanged. Recursion, automatic calibration, Website
concurrency, FFUF threads/rate, request and Website timeouts, redirects, and
HTTP/2 are the complete configurable command surface.

## Inputs And Results

Directory emits generated Target baselines first, then appends raw `websiteURLs`
in producer order while suppressing only exact baseline overlaps. A valid
zero-record Website input and a verified zero-byte wordlist remain legal. Each
FFUF target is `normalized(candidate) + "FUZZ"`: when the candidate path is
empty or does not end with `/`, exactly one separator is inserted before the
query, the fragment, or the end of the string. Scheme, authority, query, and
fragment bytes otherwise stay untouched; a candidate whose path already ends
with `/` is passed through byte-for-byte.

When automatic calibration is enabled the Engine adds only `-ac` or `-ach` and
uses the pinned FFUF built-in calibration strategies. It does not append
`-acc`, `-acs`, or `-ack`: a product-wide fixed calibration path would be a
single point of failure that a target-specific WAF rule can neutralize, while
the built-in strategy groups are randomized per invocation and are calibrated
independently. The trade-off is up to six calibration requests per Website
instead of one; the normalized target keeps every calibration path on the
target host.

The optional `ffuf.target-variants` switch (default `false`) appends
target-derived backup filename variants to each Website candidate's wordlist.
Names are the complete host, the eTLD+1, the `www.`-stripped host, and the first
two labels; suffixes are `.zip`, `.tar.gz`, `.tar`, `.rar`, `.7z`, `.sql`,
`.bak`, and `_bak.zip`. The fixed rule set bounds the count without an
artificial limit. Variants are deduplicated against the normalized source
wordlist, appended in deterministic order, and written only to a per-candidate
workspace file (`directory-wordlist/<ordinal>.txt`) that is removed after the
invocation. IP and CIDR candidates are skipped; host parsing failures are
best-effort. The selected source wordlist and typed results stay unchanged.

The optional `ffuf.waf-smart-skip` switch (default `false`) runs a two-request
pre-scan per Website origin before FFUF: one random nonexistent path and one
control path `/`. An origin is treated as WAF-blocked only when the random path
returns 403/429 or a strong challenge signature and the control path is healthy;
such a Website is skipped, counted as normally completed, and reported through
the Engine log and the run summary's WAF-skipped count. Probe failures and
ambiguous responses continue the normal scan. This is a pre-scan only: it does
not identify WAF vendors, adapt during a running scan, retry, or fabricate
results.

Two default-off result denoise switches suppress stable false positives before
submission: `denoise-auth-redirect` drops 301/302/303/307 records whose
`redirectlocation`, resolved against the record URL, points exactly at or below
a fixed login/SSO path (`/login`, `/signin`, `/auth/login`, `/sso`,
`/cas/login`, `/oauth2/authorize`, `/oauth/authorize`, `/user/login`,
`/admin/login`, `/wp-login.php`); `denoise-backup-html` drops URLs ending in a
fixed backup suffix (`.zip`, `.tar.gz`, `.tar`, `.rar`, `.7z`, `.sql`, `.bak`,
`_bak.zip`) whose content type is `text/html`. Records with 401/403,
`text/plain`, `application/*`, empty content type, or missing
`redirectlocation` are kept. Filtered counts are logged; raw FFUF JSONL
artifacts remain in the workspace and typed results are unchanged.

Accepted results contain exactly URL, status, content length, content type, and
nanosecond duration. Directory submits globally immutable, exact-URL winners
through `Execution.Results.Directories`; raw FFUF JSONL remains an Engine
workspace diagnostic artifact and response bodies are never submitted.

Target HTTPS findings are unauthenticated observations. Stock FFUF v2.2.1 does
not verify the target certificate chain, hostname, or server identity; this
exception applies only to FFUF target traffic and does not weaken LunaFox
control-plane TLS.

Valid observations are staged in a private workspace directory using an 8 MiB
chunk budget and merge fan-in 8. Exact URL groups retain the complete record
with the greatest `(candidateOrdinal, physicalRecordOrdinal)`, and the complete
winner file is closed before the first typed submission. Every return path
removes only this private staging directory; ordinal-named raw FFUF artifacts
remain for the upper-level retention policy. If every Website invocation times
out, complete pre-deadline winners are still acknowledged before the Engine
returns failure; later failure never compensates an acknowledged batch.

## Failure And Acknowledgement Windows

Configuration and candidate-pass-one failures start no FFUF process and submit
no result. A single Website deadline stops only that invocation; the run may
still succeed when another Website exits zero. An all-Website timeout submits
and acknowledges complete valid pre-deadline winners, then fails the Task.
Active-context FFUF non-zero exit and shared replay, parsing, progress,
submission, or private-cleanup faults fail the Task and stop remaining work.
Directory performs no Website retry, checkpoint, cross-execution resume, or
scanner fallback.

`ReportProgress` records fixed user-visible Task log milestones only. Even a
persisted `aggregation-completed` log is not terminal authority because its
acknowledgement or a later Engine, Container, parent-budget, or Agent step may
fail. Agent terminal reporting remains authoritative. Conversely, Server-backed
acknowledgement commits a result batch; later Task or Scan failure/cancellation
does not roll it back or hide it from Directory read surfaces.

## Engine Package And Runtime Image

The release is one independently installable Engine Package v2 bound to
digest-qualified Runtime Image refs. Package identity, catalog registration,
builtin Workflow selection, generated artifacts, and both `linux/amd64` and
`linux/arm64` image manifests must agree before publication.

The Runtime Image installs only FFUF `v2.2.1` as its directory scanner. It
selects the official `ffuf_2.2.1_linux_amd64.tar.gz` or
`ffuf_2.2.1_linux_arm64.tar.gz` release archive from `TARGETARCH`, verifies the
repository-pinned architecture-specific SHA-256 before extraction, and rejects
every other architecture. It does not build FFUF from source, use `go install`,
resolve a floating release, or fall back to another scanner.

FFUF runs with `XDG_CONFIG_HOME=/run/lunafox/ffuf-config`. The image creates
that location without a `ffufrc`, so host or image defaults cannot add headers,
scrapers, encoders, SNI, matchers, or arbitrary flags outside the Engine-owned
argv. Image conformance executes FFUF, verifies the exact version and ELF
architecture, and checks the generated Engine API v2 entrypoint on both
`linux/amd64` and `linux/arm64` release manifests.

`ffuf.headers` is a `stringArray` with `format: http-headers` and default `[]`.
Each `Name: Value` line is passed unchanged as a separate `-H` argument, so
Cookie and Authorization values may contain commas, colons, semicolons, and
spaces. Invalid names, empty values, and control characters fail before FFUF
starts; validation errors never include credentials. Duplicate names preserve
argv order, with wire semantics owned by pinned FFUF. The optional pinned
FFUF integration suite includes real local-endpoint header delivery.

Runtime unit tests skip fixed-version process behavior when a pinned binary is
not available. Set `LUNAFOX_FFUF_INTEGRATION_BINARY` to an executable official
FFUF v2.2.1 binary to exercise raw payload, redirect, target TLS, HTTP/2, HTTP/1
fallback, no-h2c, and best-effort auto-calibration behavior:

```sh
cd extensions/engines/directory_scan
LUNAFOX_FFUF_INTEGRATION_BINARY=/path/to/ffuf go test ./runtime -count=1
```

From the repository root, the source-owned build context is:

```sh
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg ENGINE_IMAGE_VERSION=0.0.0-dev \
  --build-context contracts=./contracts \
  --build-context engine-go=./engine-go \
  -f extensions/engines/directory_scan/Dockerfile \
  extensions/engines
```
