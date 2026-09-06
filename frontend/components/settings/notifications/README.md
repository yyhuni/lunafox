# Notification Settings

Notification settings keep the resolved page and loading state on one shared
geometry path. Shared workspace, page, vertically ordered destination editor, webhook
field, and footer action geometry lives in `notification-settings-layout.ts`.

- `notification-settings-loading-state.tsx` owns
  `NotificationSettingsPageLoadingState`; resolved content and workspace data
  loading MUST reuse that lightweight visual state instead
  of reintroducing a standalone page skeleton file.
- Sidebar and the protected shell do not inject a notification-settings
  skeleton; the destination workspace decides whether its own loading state is
  needed or ready content can commit directly.
- `NotificationSettingsWorkspace` must statically import the route-critical
  `NotificationSettingsPageContent`. A null dynamic fallback can let its data
  resolve before the route boundary has mounted both branches, leaving no
  observable cold-entry handoff pair.
- The route-level cold-entry geometry pairs the visible
  `notification-settings-header` and
  `notification-settings-channel-workbench` structures. Once that boundary
  becomes ready, do not insert a nested page handoff around the same content or
  those slots will no longer belong to the route owner.
- Loading form controls SHOULD reuse the real `Input` shell or shared loading
  action placeholders such as `ActionSkeleton` instead of local `h-*`, `w-*`, or
  `rounded-*` button/input approximations.
- The route skeleton takes the latest cached destination enablement,
  remediation, and supported-kind count from `NotificationSettingsWorkspace`.
  Its cards must preserve the resolved `Collapsible` state and the same control
  rhythm before handoff; do not replace this with a fixed all-expanded form
  skeleton.
- Subscription loading labels keep the resolved `textRole.bodySubtle` line box
  around their skeleton value. Do not replace that shell with a bare fixed-height
  rectangle, because the shared label typography determines the card height.
- The channel workbench shows the fixed Discord, WeCom, and Feishu editors in
  that order in one vertical stack. `NOTIFICATION_DESTINATION_PROVIDERS` is the
  single Frontend owner of that order for resolved content, loading state, and
  mock fixtures. The page owns independent saved baselines and drafts,
  then uses one shared save action to PATCH only dirty providers. A partial
  success advances only the successful baseline; failed drafts stay editable
  with provider-local feedback.
- Channel card headers use the shared 24px spacing rhythm. Their enablement
  switches keep the title, saved status, and switch visible, while collapsing
  credential, subscription, test, remediation, and error details for disabled
  drafts. Re-enabling restores those details without discarding the draft.
- The page content shell is the only vertical scroll owner. Channel cards and
  the shared save action keep their natural height in one document-flow stack;
  the save action scrolls after the final card and must never shrink editors or
  create an inner card scrollbar.
- When changing the destination editor stack or webhook field layout,
  update the layout contract and the local contract test before editing the
  resolved and loading views.
- The workbench renders only the three independent installation destinations
  Discord, WeCom, and Feishu. Each external destination owns its credential draft,
  exact-kind subscriptions, enablement validation, visibility toggle, and a
  one-shot test action. The Server remains the only provider URL policy owner;
  the UI shows its `requiresWebhookUpdate` remediation state without parsing
  provider URLs. The fixed in-product inbox has no settings row, transport, or
  editor. Email remains intentionally absent.
- Dirty drafts exist only in component memory. The page adds the browser-native
  `beforeunload` guard while dirty, without browser storage or an in-app
  navigation blocker. Mock test delivery returns an explicit unavailable result
  and never represents a message as sent.
