# Icon Standards

`frontend/components/icons` owns production Tabler imports and semantic icon mapping.

## Semantic Map First

Production pages MUST use `semanticIcons` from `@/components/icons` for
repeated stable product concepts and actions.

- stable product concepts: `semanticIcons.concept.organization`, `target`,
  `vulnerability`, `scan`, `scheduledScan`, `workflow`, `agent`, `tool`,
  `database`, `automaticAssignment`, `mcp`, and their documented
  resource subtypes
- navigation-only utilities: `semanticIcons.navigation.overview`, `search`,
  `support`, `about`
- repeated actions: `semanticIcons.action.add`, `edit`, `delete`, `view`, `run`, `stop`, `cancel`, `schedule`
- lifecycle/status: `semanticIcons.status.enabled`, `disabled`, `running`, `cancelled`, `failed`, `error`. Scan `cancelled` and `failed` states use dedicated entries; they are not interchangeable with the `stop` or local `cancel` actions.
- non-entity metrics: `semanticIcons.metric.latestScan`, `serverResources`, `enabledScans`, `taskSlots`
- persisted scan provenance: `semanticIcons.triggerSource.manual`,
  `scheduled`, and `ai` (the initiating source is represented by `IconUser`,
  `IconCalendarClock`, and `IconRobot`; these are source markers, not action or
  lifecycle icons)

Runtime-detail cards use `semanticIcons.concept.agentTopology` only as an Agent-source decoration, `semanticIcons.status.unknown` for a named unknown state, and `semanticIcons.metric.cpu` / `memory` / `disk` for source-owned server measurements. These entries keep the dashboard from selecting lookalike raw glyphs locally.

`semanticIcons.concept` is the only glyph source for a stable product noun. A
generic target, organization, vulnerability, or scan MUST use the same concept
entry in navigation, lists, dialogs, details, and metrics. Explicit subtypes
such as domain, IP, and CIDR select their own concept entries. Active tone,
size, background shell, and accessibility treatment are presentation concerns
owned by theme tokens and UI primitives; they must not select a different
product glyph.

Keep alert or failure states on status icons such as
`semanticIcons.status.error`; do not use a status glyph for an ordinary product
concept or replace a product concept with an action glyph.

Do not choose a raw icon locally for a stable semantic concept just because
another icon also looks plausible. If the registry is missing a stable concept,
extend `semantic-icons.tsx`, this guide, and the related contract before using
it. The component-foundation guard rejects audited raw stable-concept imports
outside this owner.

## Raw Icon Exports

`index.tsx` exports raw Tabler aliases for one-off visual affordances. New
production UI MUST NOT use them when the icon represents a stable product noun;
use the semantic map for actions, statuses, and navigation utilities as well.
New production UI MUST NOT use raw aliases when the icon represents a stable
product noun; use the semantic map for actions, statuses, and navigation
utilities as well.

Direct third-party icon imports remain limited to this folder, and
`@tabler/icons-react` is the only approved production glyph dependency unless a
reviewed exception exists. Pages still import all icons through the shared
`@/components/icons` entrypoint.
