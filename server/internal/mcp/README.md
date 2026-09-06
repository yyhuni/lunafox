# MCP Server Boundary

`internal/mcp` is the protocol adapter for the `/mcp` investigation and
constrained-creation surface. It owns Streamable HTTP admission, static-key
authentication, bounded tool execution, MCP-specific DTOs, error mapping, and
best-effort audit metadata.

The package deliberately depends on narrow application ports supplied by
bootstrap wiring. It must not issue loopback HTTP requests, implement
organization membership, or persist bearer plaintext. Business writes are
limited to `create_organization`, `create_target`, the four explicit
vulnerability review actions, and one immediate `start_scan`; each delegates
to the shared application command path synchronously. `create_target.organization`
is an optional canonical `organizations/{id}` business-group reference applied
to the whole batch, not a tenant, user scope, or data-isolation selector. The
official `github.com/modelcontextprotocol/go-sdk` dependency is consumed only
by `transport` and `tools` registration code; REST handlers remain independent.

The first release supports only MCP `2026-07-28`, stateless POST requests, and
the existing Server public HTTPS edge. A valid active static Bearer MCP key is
the only call-authentication condition. Origin is intentionally ignored: this
does not provide browser cross-origin or DNS-rebinding protection. A security
review is mandatory before adding browser MCP clients, Cookie/session auth,
multi-tenant isolation, broader write tools, or any new authentication model.

## Tool catalog

The registry is a single static allowlist. Every listed tool is registered by
default when the Server starts; there is no feature flag or capability
negotiation layer.

| Area | Tools |
| --- | --- |
| Target and scan reads | `list_targets`, `get_target`, `list_scans`, `get_scan`, `list_target_websites`, `list_target_subdomains`, `list_target_endpoints`, `list_target_directories`, `list_target_host_ports`, `list_vulnerabilities`, `get_vulnerability` |
| Organization investigation | `list_organizations`, `get_organization`, `list_organization_targets` |
| Target investigation | `list_target_vulnerabilities`, `list_target_screenshots`, `get_screenshot_image` |
| Configuration discovery | `list_scan_workflows`, `get_scan_workflow`, `get_scan_workflow_profile`, `list_engines`, `get_engine`, `list_wordlists`, `get_wordlist` |
| Diagnostics | `list_server_log_entries`, `list_agent_log_entries` |
| Vulnerability disposition | `review_vulnerability`, `unreview_vulnerability`, `batch_review_vulnerabilities`, `batch_unreview_vulnerabilities` |
| Scan operation | `start_scan`, `get_operation` |
| Existing constrained creates | `create_organization`, `create_target` |

This is a deployment-level trusted surface. A valid static Bearer key grants
the same deployment-wide data scope regardless of caller, organization, or
`Origin`; an organization is only a business grouping. It is not a tenant,
RBAC subject, or permission switch. Target and organization soft-delete
visibility, retained Scan history, and operation retention continue to follow
the existing application owners.

## Configuration and operation rules

`get_scan_workflow_profile` returns the complete current frontend-equivalent
configuration shape. A caller may edit only public Step/section/parameter and
resource fields, then `start_scan` must receive one canonical target, one
workflow, and the complete `configuration`. The Server re-resolves the latest
Workflow/Engine/resources at submit time, validates with the canonical Scan
decoder/validator/planner, and freezes the resulting plan and package facts.
Profile data is an editing starting point, never an implicit fallback; omitted
Steps, unknown fields, implicit booleans, secrets, plans, commands, and package
identity fail fast.

`start_scan` creates one immediate Scan and a durable `operations/{uuid}` in the
same transaction. `get_operation` projects the Scan lifecycle and is intended
for ordinary cross-request polling. The optional business `request_id` replays
an identical committed request and rejects a conflicting fingerprint; it does
not retry Scan work. Existing Scan stop/cancellation behavior remains the sole
stop surface, and a disconnect after commit does not cancel the Scan.

All collection tools use bounded `page_size`/`page_token` envelopes with
query- and parent-bound opaque cursors. Screenshot lists contain metadata only;
`get_screenshot_image` returns one `image/webp` image block. Server and Agent
logs use fixed sources, the existing 15-day lookback, cursor directions, line
normalization/truncation, and fetch ceilings. Every complete MCP result remains
subject to the 1 MiB budget and returns a bounded public error when exceeded.

## Deliberate exclusions

The MCP boundary does not expose scheduled scans, batch `start_scan`,
`delete_operation`, arbitrary LogQL/source/time/level filters, log export,
Task-progress log queries, generic vulnerability patch/delete, arbitrary
organization/target updates or deletes, organization members/RBAC, single
child-asset Get tools, raw Screenshot downloads/signed URLs, plaintext
provider secrets, Wordlist paths or contents, MCP-specific audit storage/read
APIs, or a second authentication/tenant-isolation model. Adding any of these
requires a separate OpenSpec/security review.
