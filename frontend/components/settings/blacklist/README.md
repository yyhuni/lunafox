# Shared Blacklist Workbench

`BlacklistSettingsWorkspace` is the shared BlacklistPolicy editor used by the
scan-configuration Blacklist tab and the target detail Settings > Blacklist
workspace.

- The global route is `/scan/config/blacklist/`. The scan-configuration shell owns
  the page title, so this workspace does not render a PageHeader. It keeps the
  page gutter, `blacklist-page-content` handoff, and page-level error action,
  and it reads and PATCHes only the global `BlacklistPolicy`. The global route
  shows a notice under the workbench. Embedded target settings do not.
- Embedded mode omits the page gutter. The target detail shell owns navigation,
  while this component owns the parallel global/local queries, read-only
  inherited rules, Target-local editor, validation, save feedback, and workbench
  loading structure.
- `blacklist-settings-layout.ts` is the shared geometry contract. Keep loading and
  resolved slots paired when changing the rule list, editor, or action row.
- The loading editor reuses `BulkLineValidationInput` with its label visually
  hidden and empty/success feedback suppressed. Its skeleton may mask only the
  textarea contents; do not recreate the editor shell or add loading-only
  feedback rows that change the first resolved frame.
- Rule-group triggers expand and collapse the list, align with the rule-content
  start, and keep a transparent hover surface. Their text/chevron feedback
  distinguishes this lightweight inspection action from editable policy controls.
- The embedded Target view starts its read-only inherited global-rules group
  collapsed, so the editable Target-local rules remain the primary working
  context. Users can expand it whenever they need to compare inherited rules.
- Embedded mode reads the global policy only for the `blacklist-inherited-rules`
  read-only region and PATCHes only `targets/{target}/blacklistPolicy` with the
  Target-local `patterns` and `etag`.
- A 409 conflict reloads the active policy and reports a warning. It never retries,
  overwrites, or reports successful saving.
