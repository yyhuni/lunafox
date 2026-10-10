# Global Blacklist

The global exclusion policy lives at `/scan/config/blacklist/` inside the scan
configuration shell. The shell owns the page title. This route mounts
`BlacklistSettingsWorkspace` without `embedded`, so it edits only
`GET/PATCH /v1/blacklistPolicy` and keeps the shared workbench gutter.

`/settings/blacklist/` is not a route and does not redirect here. Target detail
settings continue to mount the same workspace with `embedded` and a `targetId`.
