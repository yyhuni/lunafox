# catalog/router

## Structure
- Public entry: `routes.go` (`RegisterCatalogRoutes`)
- Other files contain package-private sub-route registrars (`register*Routes`)

## Files
- `targets.go`: target routes
- `workflows.go`: workflow catalog routes
- `workflow_profiles.go`: workflow profile routes
- `wordlists.go`: wordlist routes

## Constraints
- External callers should only use `RegisterCatalogRoutes`
- Keep resource route registration split by resource, with private helper functions
