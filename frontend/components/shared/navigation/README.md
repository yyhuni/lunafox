# Shared Navigation Motion

`RouteContentTransitionProvider` tracks committed pathname changes inside the
protected shell. `RouteContentTransition` uses that identity to replay a
short opacity-only transition on an explicitly scoped content region.

Use this owner only where a route relationship benefits from continuity, such
as list/detail navigation or detail child tabs. Keep shell chrome, sidebars,
headers, and tab rails outside the wrapper. The provider intentionally leaves
the initial protected route static so it does not duplicate the app's existing
boot/content reveal.

Do not use this owner for loading skeletons, query-only refreshes, full-page
slides, shared-element morphs, or decorative repeated animation.
