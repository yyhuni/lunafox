# Screenshot Engine

`engine.lunafox.screenshot` is the independent browser-backed observation
engine. Its generated Engine API v2 entrypoint is
`screenshot-runtime-engine`; the Engine owns its Runtime Image, task workspace,
progress protocol, image conversion, and `asset.screenshot.v1` result
submission.

## Inputs and candidates

The Engine receives the canonical `Execution.Target` and requests the
Server-produced `websiteURLs` file through
`Execution.Input.WebsiteURLs.Path(ctx)`. Workflow orchestration, not the Engine, waits for
`website_discovery -> url_collection` to finish and authorizes the Screenshot
Task. The file is a finalized current-scan product and may be empty.

Candidates are baseline-first and streamed without input-sized deduplication:

- Domain and IPv4 Target: `http://<target>`, then `https://<target>`.
- IPv4 CIDR: every address from network through broadcast, HTTP before HTTPS.
- Finalized `websiteURLs` are appended in file order, including exact baseline
  overlaps and repeated facts. Explicit ports, paths, queries, and fragments
  remain candidates.

No HostPort, endpoint, redirect, or other undeclared fact is converted into a
candidate, and this Engine has no candidate-count cap.

Candidate generation is one-pass and writes directly to task-local disk. HTTPX
rows are parsed and submitted only when HTTPX emits them. The observed `url`
field is the result URL; `input` is optional diagnostic data and is never used
for membership, association, duplicate rejection, or URL fallback. A valid
empty product keeps its Target baseline, and no input is silently truncated.

## Configuration

The `capture` Engine section exposes only:

| Parameter | Default | Range | Meaning |
| --- | ---: | ---: | --- |
| `page-timeout` | 15 | 1--120 | One page screenshot timeout in seconds |
| `concurrency` | 5 | 1--20 | Direct HTTPX `-threads` value |
| `retries` | 1 | 0--3 | HTTPX probe retry count |

`configSections[].defaultEnabled` remains an Engine-internal section default;
it does not enable the Workflow Step. A disabled Workflow Step is skipped by
Server planning and never starts this container. If the enabled capture section
is disabled inside the Engine configuration, the defensive runtime path returns
without starting HTTPX or emitting progress; normal Scan planning rejects an
all-disabled Engine configuration before a container can start.

## Runtime image and HTTPX

The image pins HTTPX `v1.10.0`, Chromium `131.0.6778.85` from a dated Debian
security snapshot with architecture-specific SHA256 verification, and official
cwebp/libwebp `v1.6.0`. The same browser build is supported on amd64 and arm64.
Browser acquisition is offline at runtime: HTTPX always uses `-system-chrome`.
The image fixes Chromium to headless software rendering with no zygote and no
shared-memory dependency so the same root runtime remains stable on amd64 and
arm64. The image and its subprocesses run as root without browser sandbox
claims, which is the accepted isolation boundary for this Engine.

Every pass uses a fixed viewport (`1280x720 @ 1x`) and task-scoped storage:

```text
-json -ss -no-screenshot-full-page -system-chrome
-screenshot-timeout <page-timeout>s -sid 1s
-threads <concurrency> -retries <retries>
-ho window-size=1280,720 -ho force-device-scale-factor=1
-ob -esb -ehb -srd <task-workspace>
```

HTTPX writes JSONL to stdout; the Engine does not use `-o`. `-ob`, `-esb`, and
`-ehb` keep response bodies, screenshot bytes, and rendered HTML out of JSON.
The Engine consumes bounded JSONL records synchronously in one Engine pass.
Missing rows, failed rows, malformed rows, and conversion failures do not cause
an Engine-owned replay. HTTPX's own `-retries` setting remains the only retry
mechanism configured by this Engine.

## Image and result contract

HTTPX produces task-scoped temporary PNG files. Before conversion the Engine
requires a workspace-local regular PNG, valid PNG structure, at most 8 MiB
encoded bytes, and at most 8,000,000 decoded pixels. cwebp first encodes a
proportional maximum width of 800px at quality 72. Only when that output is over
256 KiB does it encode once at maximum width 640px and quality 65. A second
over-budget WebP becomes `skippedImageBudget`; quality is not reduced further.

Only `asset.screenshot.v1` is submitted. Each closed item contains the raw
observed HTTPX `url`, optional integer `statusCode` in 100--599, and canonical
RFC 4648 standard-base64 encoded WebP bytes. The Server admission boundary
accepts only structurally valid WebP no wider than 800px, no larger than
8,000,000 pixels, and no larger than 256 KiB decoded bytes. It stores accepted
WebP bytes unchanged and never resizes or transcodes them. PNG, JPEG, AVIF, and
other formats are rejected. This change adds no separate manual Screenshot HTTP
API compatibility layer.

## Progress and cleanup

An enabled runtime reports aggregate-only events:

```text
input_ready candidates=<n> pageTimeout=<seconds> concurrency=<n> retries=<n>
screenshot_attempt started attempt=1 candidates=<n>
screenshot_attempt completed attempt=1 processed=<n> submitted=<n> retryPending=0
completed candidates=<n> processed=<n> submitted=<n> attempts=<n> skippedNavigation=<n> skippedScreenshot=<n> skippedInvalidPNG=<n> skippedConversion=<n> skippedImageBudget=<n>
```

No progress event contains a URL, status code, final URL, response body, HTML,
command, local path, or image bytes. URL-local failures can leave the Task
successful with zero results. Process, protocol, workspace, progress,
submission, cancellation, or cleanup infrastructure failures fail the Task.

The Engine reclaims each consumed PNG and removes its candidate file and
incomplete intermediates on every terminal path.
Only accepted WebP results are persistent product data.
