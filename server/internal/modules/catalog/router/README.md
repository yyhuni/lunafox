# catalog/router

## Structure
- Public entry: `routes.go` (`RegisterCatalogRoutes`)
- Other files contain package-private sub-route registrars (`register*Routes`)

## Files
- `targets.go`: target routes
- `engines.go`: engine routes
- `presets.go`: preset routes
- `wordlists.go`: wordlist routes
- `worker.go`: worker routes

## Constraints
- External callers should only use `RegisterCatalogRoutes`
- Keep resource route registration split by resource, with private helper functions
