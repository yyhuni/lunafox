# identity/router

## Structure
- Public entry: `routes.go` (`RegisterIdentityRoutes`)
- Other files contain package-private sub-route registrars (`register*Routes`)

## Files
- `auth_routes.go`: auth routes
- `user_routes.go`: user routes
- `organization_routes.go`: organization routes

## Constraints
- Expose only the module entry function
- Use sub-registration helpers only within this package
