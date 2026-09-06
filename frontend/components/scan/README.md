# Scan UI

This module owns scan launch, scheduled scan creation, scan progress, and scan history surfaces.

`ScanStatusBadge` owns the canonical scan-status glyph, tone, and running-state
rotation used by scan history. Its `icon-only` variant is for compact status
columns and retains the localized status as the icon's accessible name and hover hint.

## Trigger Provenance

Every Scan transport projection carries the server-owned, required
`triggerType` vocabulary: `manual`, `scheduled`, or the reserved `ai` value.
The frontend transport adapter rejects a missing or unknown value instead of
inferring `manual`. Scan launch request bodies never send this field: public
quick and batch creation are attributed by the Server, and only a future
approved internal producer may create an AI-triggered Scan.

普通 Scan 不再定义或传输 `scanMode`；`scanWorkflow` 表示执行编排，
`triggerType` 是唯一的创建来源字段。Scheduled Scan 自身的
`scanMode: target|organization` 属于范围选择契约，与普通 Scan 记录无关，
继续由 scheduled-scan 模块维护。

## Execution Input Source

Every Scan transport projection also carries the required persisted
`inputSource`: `scanSnapshot` or `targetInventory`. It is independent of
`scanWorkflow`, Agent assignment, `triggerType`, and Scheduled Scan scope.
`scanSnapshot` exports finalized facts discovered by the current Scan, so
later workflow steps consume this Scan's results rather than a create-time
asset copy. `targetInventory` asks the Server to read the Target's current
`subdomains`, `hostPorts`, and `websiteURLs` when an authorized execution input
is requested. The latter is a request-time view, so a later role request or
retry can observe newer Target assets.

Quick, ordinary/batch, and Scheduled Scan creation state begins at
`scanSnapshot`, but always submits its currently selected source explicitly.
The transport adapter rejects a missing, empty, or unknown request/response
value; UI state must not infer a source after the request crosses the boundary.
The Agent does not choose this value and never receives a UI fallback path.

## Launch Setup Hierarchy

Normal and Quick Scan launch keep Workflow selection as the only expanded
primary decision. The shared execution-options disclosure is collapsed by
default and contains the persisted `inputSource` and optional Agent assignment.
The disclosure changes neither default (`scanSnapshot` and automatic assignment)
nor submit behavior; it only keeps non-default execution controls out of the
initial decision path. A first-step footer must show the selected Workflow's
localized display name, never its transport resource value.

## Global Quick Scan Trigger

`QuickScanHeaderTrigger` owns the unified-header quick-scan trigger: its dialog
composition, compact ghost icon Button, Zap icon, accessible name, and stable
`quick-scan-trigger` selector. `UnifiedHeader` dynamically loads this component
and does not inline a quick-scan dialog trigger. This global icon command follows
the shared `Button` appearance.

Scan log surfaces reuse `RawLogViewer` with `TerminalLogCopyAllButton` in its `topRightAction`; copy the complete currently loaded scan log text, not only a filtered or visible fragment.

## Agent Assignment

Normal, bulk, and quick Scan launch reuse `ScanAgentSelector`. It always shows
registered Agents with online/offline state, real heartbeat CPU/memory/disk
usage, and occupied execution slots. Each online row groups node identity with
a health badge and renders the four resource metrics as icon/label, value, and
meter tiers; task slots are displayed as a `used/total` count without a meter. The selector
uses the same threshold tones as the shared metric bars; offline or
heartbeat-less Agents do not display stale resource values.
Offline or unhealthy Agents remain selectable: selecting one
pins the new Scan and it may remain pending until that exact Agent can claim it.
The omitted option is automatic assignment. Do not let UI callers construct
`agents/{id}` directly; service adapters own the canonical request reference.
`ScanSearchablePicker` is the shared bounded `Popover` + `Command` overlay
used by both Agent and workflow selection. It keeps the Scan drawer body stable
as either list grows and closes only after an enabled result is selected. The
Agent selector does not request the candidate collection while closed. Opening
it requests `GET /v1/admin/agents` with `pageSize=50` and `createdAt desc`, then
uses the opaque `nextPageToken` only when the list-bottom observer reaches the
current loaded range. Its normalized 250ms server search compiles node name,
hostname, and `connectionIp` into the canonical contains OR filter; each
search value owns an isolated query generation. While open, the selector
refreshes only its already loaded pages every 15 seconds. A next-page failure
retains rows, scroll context, selection, and token until explicit retry.

A saved non-automatic selection always resolves through canonical Agent detail,
including while the picker is closed or the Agent is outside every loaded
candidate page. Detail failure preserves and labels the original `agents/{id}`
reference; it must not switch the form to automatic assignment. The workflow
selector keeps its local name/description filtering. The trigger is the only
selected-assignment indicator; Agent rows do not add a selection checkmark.

## Workflow Management

The canonical workflow-management route is `/scan/config/workflows/`. It and
the canonical Engine route `/scan/config/engines/` share the Scan Configuration
title and route-driven primary Tabs. The legacy `/scan/workflow/` and
`/tools/engines/` routes are redirect-only compatibility aliases. The shared
shell applies only to the two management views: entering the composition
builder removes its title and Tabs so an in-progress workflow remains in the
full-height focused canvas.

The scan workflow list owns editing through its rightmost row action column. The edit action opens the existing composition builder directly; the management list must not add a selection-driven detail drawer, YAML preview, or side rail between the table and builder.

Workflow management is cursor-backed: retain `totalSize` as a result summary
and use only cached prior tokens plus the active response `nextPageToken` for
adjacent navigation. The first-page action clears the token cache and requests
page one. Do not calculate total pages or expose last, numbered, or current-page
controls.

`scan-workflow-page-content` is the route's first-screen workspace owner. Keep
its `ContentHandoff` at `layer="workspace"` and registered for the automatic
paired `surface` geometry contract. Its management table must also pair
`scan-workflow-toolbar` on the real search region and
`scan-workflow-primary-region` on the real table container; the route header
is stable chrome, not a second loading owner. The composition builder is not
the first-entry management-list loading target.

The loading and resolved workflow toolbar use the same `SimpleSearchToolbar`
props. Do not add a toolbar-only loading indicator: its extra input padding
changes the intrinsic mobile width and causes a handoff shift.

Workflow management uses the persisted `scanWorkflows` resource. The table
searches and paginates through the backend; create and user-workflow edit use
the management mutations with an ETag update. Built-ins show the built-in state
and open the same canvas in read-only mode, without save, delete, or topology
controls. A workflow with unavailable Engines remains visible and repairable in
management but is disabled in scan selection.

The desktop workflow canvas vertically centers the floating Engine library at
its bounded available height so the internal list can show more Engines before
scrolling; the narrow-layout override remains deliberately shorter and
top-aligned to preserve canvas space.

The composition builder loads its Engine library through the hook-owned
installed Engine Catalog and uses each complete `engineId` as the only Engine
identity in draft and saved topology. Engine names and descriptions are
render-time locale projections; do not restore a handwritten builtin registry,
slug-to-ID map, or first-Engine fallback. While the Catalog is loading or
unavailable, adding Engines and saving stay disabled. A historical Step whose
Engine is no longer installed keeps its exact ID, renders as unavailable, and
may be removed before saving the repaired Workflow.

Scan and scheduled-scan dialogs load the selected workflow's parent-scoped
Profile singleton through the shared strict Profile adapter. The adapter requires
matching parent identity, exact Step coverage, an explicit boolean `enabled`, and a
complete Profile-provided `engineConfig` for every Step. The shared canonical
serializer emits disabled Steps as `{enabled: false}` and enabled Steps with a
complete `engineConfig`; every submit path performs the at-least-one-enabled
preflight and never merges Workflow, catalog, or form defaults. `ENGINE_CONFIG_INVALID`
keeps the draft and requires an explicit Profile refresh/review before retry.

Within an enabled Step, a catalog section declared `requiredEnabled: true` stays
enabled and its inner switch is non-operable. This metadata applies only to that
individual Engine config section: it does not enable the outer Workflow Step or
express Engine orchestration. Optional inner sections retain their ordinary
toggle behavior, while disabling the outer Step still serializes exactly
`{enabled: false}`. Client validation provides early feedback; Server validation
remains authoritative for direct or persisted configuration input.

`ScanConfigViewToggle` owns one complete paginated Wordlist Catalog generation
for every resource selector in the editor. While that generation is loading or
incomplete, selectors preserve their values and stay disabled. On failure they
also preserve values, show the shared retry state, and never fall back to free
text. Only a complete successful generation may prove absence: present values
remain selected, while a confirmed missing value is normalized once to `""`
without selecting the first or another wordlist. An empty complete Catalog is a
valid unselected state.

The selector value is the immutable canonical resource `wordlists/{id}` and its
label is the read-only `fileName`. An Engine Definition default is only a plain
`fileName` preferred candidate; it is converted to a selector value only after a
complete Catalog result contains exactly one match. Loading, failed, incomplete,
absent, or duplicate matches remain unresolved and cannot be persisted or
submitted.

Empty resource values become field errors only when Start Scan invokes the
configuration surface's validate-and-reveal operation. It validates enabled
Steps and enabled configSections, marks every empty resource field, switches
valid YAML to form mode when needed, expands the first affected Engine card,
and scrolls/focuses the first field; no Scan request is sent. Selecting a value
clears only that field's error. Non-field prerequisites continue to own the
action's disabled state, and Server validation remains authoritative for
current Catalog/file availability.

`ScanConfigViewToggle` owns one bounded configuration-content scroll region.
Its advanced YAML-mode controls, engine-configuration heading, and active
form/YAML content scroll together; the surrounding workbench header and footer
remain fixed. YAML keeps its bounded text-editor viewport inside that content
region. Do not reintroduce a form-only scroll owner that leaves the advanced
mode controls fixed above engine configuration.

## Workbench Headers

Initiate scan, quick scan, and scheduled scan creation use the shared
`EdgePanelHeader` `workbench` variant. The owner supplies the semantic
`DrawerTitle`/`SheetTitle` and description; the shared header owns the task-icon
container, responsive title/action layout, and the optional progress slot. Keep
step state in the scan dialog and derive header progress only from the current
step and total step count through `getScanStepProgress`. Configuration
validation and submit readiness belong to the footer feedback and action state;
they must not change the header's step progress. `InitiateScanStepHeader` renders
that slot as a neutral track with an `interaction-accent` line and dot anchored
to the current progress. A scan workbench without
a meaningful concept icon may omit `leading` without reserving an empty column.

Scan workbench footers reserve the shared two-line status rhythm even when the
current step has only a one-line or empty status. `InitiateScanFooter` owns that
slot for scan configuration, and `QuickScanFooter` must match it for the target
step. This reserves navigation action positions across desktop and narrow
layouts; do not remove the slot merely because the current step renders less
feedback.

## Drawer Motion Contract

Scan launch and scheduled scan creation are right-side workbench layouts. They use the shared edge-panel transition; their workbench classes own content width, padding, and overflow rather than a separate animation.

- For scan launch, use the shared Base UI `Drawer` with `swipeDirection="right"`, `DrawerContent`, and `DrawerClose render`.
- For scheduled scan creation, use `SheetContent` with `side="right"`; the shared primitive owns its directional transition.
- Let the shared Sheet/Drawer primitive own the shadcn base-nova backdrop; scan drawers must not pass local backdrop variants.
- Use `scanWorkbenchDrawerContentClassName` for panel width, padding, and overflow.
- Do not inline the scan drawer width or motion class bundle in route components.
- Do not switch these drawers back to `DialogContent` unless the product flow is reduced to a short, simple modal task.

The contract tests for `initiate-scan-dialog` and `create-scheduled-scan-dialog` guard this rule so launch drawers stay visually aligned with every shared edge panel.
