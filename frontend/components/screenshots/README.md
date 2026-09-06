# Screenshots Gallery Rules

`frontend/components/screenshots/` owns the target and scan screenshot gallery workspaces.

Screenshot URLs are immutable observed strings: UI display and external-link handoff use the stored value, while browser-side request serialization after that handoff is outside the platform guarantee. The gallery does not parse, normalize or rebuild URLs for identity or filtering.

Target and scan galleries use cursor-only pagination. They retain the filtered
result total and page-size selector; the active response `nextPageToken` alone
enables next, while an already visited token in the active-query cache alone
enables previous. The first-page control resets the active token cache and
requests page one. Do not show page numbers, current/total-page text, last
actions, or derive a terminal page from `totalSize`.

截图图库的两个后端排序字段（状态码、创建时间）必须通过 `ScreenshotSortControl` 的一个紧凑排序菜单呈现；不要在工具栏并列重建字段排序按钮。菜单选择已有字段时沿用当前方向切换，切换到另一字段时保持其既定默认方向。

## Loading Geometry

- `screenshots-gallery-layout.ts` owns the gallery root, toolbar, toolbar-control, toolbar-action, and grid geometry. Resolved content and loading state must import these constants instead of restating the same class strings.
- The initial toolbar loading state must mirror the real shared controls: URL search uses `SearchToolbarSkeleton`, and status/sort/action placeholders use `ActionSkeleton`. Do not collapse the toolbar into a single wide `Skeleton` block or hand-roll button-sized `Skeleton h-* w-* rounded-*` placeholders.
- Gallery item loading may remain aspect-ratio media placeholders because the resolved cards are screenshot media frames, but the surrounding grid must stay shared through `SCREENSHOTS_GALLERY_GRID_CLASS`.
- `ScreenshotsGalleryRouteFallback` accepts an explicit stable first-frame item count and an optional target-only selection-action slot. Target detail uses three items plus that action slot because this preserves one desktop grid row and two narrow-screen rows without waiting for unavailable target metadata. Other detail shells may omit both props and retain the shared eight-card/no-target-action defaults; an explicit count must be a positive integer.
- `screenshots-gallery-skeleton.tsx` has been hard-cut. Keep normal query loading inside `ScreenshotsGalleryLoadingState` / `ScreenshotsGalleryContent` so loading and resolved gallery shells share the same owner.
