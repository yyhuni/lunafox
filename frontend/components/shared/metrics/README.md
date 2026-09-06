# Shared Metrics Rules

`frontend/components/shared/metrics/` owns fixed-height metric-strip primitives
such as `StatMetricRow`.

`SegmentedMetricProgress` owns compact and panel segmented usage bars. It keeps
status-colored segments by default, and value text also follows status tone by
default so threshold-bearing Agent metrics can highlight warning/error values.
Use `valueTone="neutral"` only when the numeric percentage should read as plain
metadata while the bar itself still communicates status, such as the global
server resource popover. Use `barTone="neutral"` together with a neutral value
when an overview surface needs a system-monitor style meter without semantic
color accents. Use `barTone="threshold"` with an explicit `threshold` when only
values at or beyond that limit should receive a status color.

The `runtime-card` variant owns the compact two-tier resource row used by the
overview's server-resource card: icon and label plus capacity detail on the
first line; a full-width segmented meter with its percentage on the second.
The overview uses `barTone="threshold"` with neutral value text, so ordinary
resource use stays visually neutral and only an exceeded visual threshold gains
a status color. The wider `panel` variant remains status-colored for the global
resource popover.

The compact variant uses a fixed three-track row around a flexible bar. The
label starts at the row's left edge, the bar keeps one shared start/end track
across all metric rows, and the value ends at the row's right edge.

The `inline` variant is for dense selectable rows that compare several resource
values in one line. It presents an icon and label, then the exact value, then a
full-width continuous progress bar directly below. Callers may omit that bar
for non-utilization values such as task slots, and own responsive wrapping
rather than allowing the row to overflow horizontally.

## Inline Metric Loading

Fixed-height metric strips MAY keep their resolved shell mounted during initial
loading and replace only the volatile numeric value with an inline skeleton.

`SegmentedMetricProgress` exposes `loading` for the compact row. Use that
native variant when the row shell is pending so its label, bar, and value tracks
keep the same line-box geometry as the resolved metric.

Use this pattern when all of the following are true:

- the strip layout, labels, footer copy, divider rhythm, and row height are
  already stable
- the loading uncertainty is localized to the value slot instead of the whole
  section geometry
- keeping the real shell mounted avoids a second scene cut more effectively
  than wrapping the strip in a larger `ContentHandoff`

This is a standard progressive-loading pattern, not a fallback exception. The
user keeps one stable metric strip while the value position swaps from inline
skeleton to inline number.

## Do And Do Not

- DO keep the real label, footer, dot/icon, divider rhythm, and row height
  mounted while loading.
- DO keep the inline loading placeholder line-box compatible with the resolved
  numeric typography.
- DO expose a section-local `dataSlot` such as
  `scan-history-stats-skeleton` when browser probes need to verify that the
  metric strip is still in its inline skeleton state.
- DO use a larger inline placeholder only for the value slot that is visually
  larger, such as a featured metric.
- DO NOT wrap a metric strip in a page-wide `ContentHandoff` only because the
  numbers are pending if the strip shell itself is already stable.
- DO NOT reveal the strip as `title -> footer -> number`; the whole strip shell
  stays mounted and only the value slot changes.
- DO NOT use the inline metric pattern when the whole section height, control
  layout, tabs, or surrounding geometry are still unstable. In those cases use
  a section or route owner from `@/components/shared/loading`.

## Audited Example

- `/scan/history/`
  - desktop `1280x720`: `1300ms` shows `scan-history-stats-skeleton` together
    with the list skeleton; `2200ms` the stat strip is resolved while the list
    handoff has also reached content
  - mobile `390x844`: `1300ms` shows `scan-history-stats-skeleton` together
    with the list skeleton; `2200ms` the stat strip is resolved while the list
    handoff has reached content

This example is intentionally distinct from same-layer staged reveal. The stats
strip is one stable shell with inline value placeholders, while the list below
owns a separate first-screen table workspace.

## Verification

- update `components/shared/metrics/__tests__/stat-metric-row.contract.test.ts`
- update feature-local contracts such as
  `components/scan/history/__tests__/scan-history-stat-cards.contract.test.ts`
- when auditing route timing, use `LOADING_SMOKE_TARGET_IDS=<route-id>` and
  `LOADING_SMOKE_TIMELINE_MS=700,1300,2200,4000` so `visibleDataSlots`
  captures the metric-strip skeleton slot
