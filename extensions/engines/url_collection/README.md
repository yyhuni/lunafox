# URL Collection Engine

`engine.lunafox.url_collection` is the first-party URL Collection Engine. It
uses the Server-produced `websiteURLs` role at runtime through
`Execution.Input.WebsiteURLs.Path(ctx)` and supports domain, IP, and CIDR
Targets. The generated contract still exposes the complete closed input Registry;
this Engine does not declare an input allow-set.

The Server prepares `websiteURLs` as finalized Website URL facts from the
current scan. It does not inject Target baseline URLs. The Engine constructs a
Target-baseline-first workspace seed, appends snapshot URLs in their stored
order, and keeps the existing overlap rules. A successful zero-byte facts file is
valid as long as Target baseline generation succeeds.

The seed materializer streams finalized Website URL lines directly to the
task-local Katana artifact and reports a checked `uint64` count. Target/CIDR
baselines are emitted one address at a time; no URL, byte, record, or CIDR
business cap is applied and no prefix is silently discarded. Framing, write,
close, cancellation, and counter-overflow failures abort before a collector
starts and remove the incomplete seed. Candidate identity is the exact raw URL;
only the existing baseline-root overlap is removed, while source order is
preserved for the downstream external sort. URL Collection has no Engine
retry/replay or input-sized identity map, and a partial seed never starts a
collector.

The runtime runs applicable Waymore and Katana collectors concurrently, uses
Uro for optional cleanup, and uses HTTPX for optional verification. All tool
arguments are assembled from the fixed manifest configuration without shell
execution or arbitrary tool flags. A failure in any enabled applicable stage
fails the task; disabling HTTPX explicitly is the sole path that creates a
minimal unverified Endpoint.

Collected URLs are admitted without URL rewriting, scope-filtered against the
input Target, and staged on task-local disk. After every enabled tool succeeds,
raw `asset.endpoint.v1` records are submitted through
`Execution.Results.Endpoints`. Endpoint identity is the Collector/HTTPX input
URL, not a redirect target. URL Collection adds no retry, resume, checkpoint,
intermediate-file reuse, historical Website fallback, authenticated collection,
directory scanning, or frontend surface.

Runtime Image tools and their exact versions are declared in `Dockerfile`,
`requirements.lock`, and the container conformance fixtures. The Python lock
contains direct and transitive Waymore/Uro dependencies, so the runtime image
contains executable CLIs rather than only their entrypoint scripts. Agent task cleanup removes the seed copy,
workspace, tool outputs, and unsubmitted staging on every task terminal.
