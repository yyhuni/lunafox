# Response Evidence Panel

`ResponseEvidencePanel` owns the shared tabbed inspection of captured HTTP
response values. It is domain-neutral: callers provide response strings and
localized labels, while search and website modules keep their own surrounding
identity, metadata, screenshots, and navigation.

## Contract

- `body` is the default pane and `headers` is always the second tab.
- `location` is rendered as the third tab only when its value is non-empty.
- `emptyLabel` is required and is shown for an empty selected response value;
  the panel does not silently replace missing content with a hyphen.
- `showMetadata` gates the optional content-type/response-size footer. Search
  cards keep those summaries in their own header badges; website relation
  evidence enables the footer.
- The panel intentionally has no copy action. Detail drawers retain their own
  existing vertical response fields and copy behavior.
- Long values stay inside the panel's bounded `ScrollArea` and use the shared
  `textRole.code` typography with wrapping for long lines.
- `ResponseEvidencePanelLoadingState` reuses the same panel frame, tab rhythm,
  bounded body region, and optional metadata footer without exposing response
  data. Consumers with a paired loading handoff should use it instead of
  approximating the panel with fixed-height rectangles.

Consumers must import the shared panel through `@/components/shared/response-evidence`
and must not pass website/search domain objects or translation hooks into it.
