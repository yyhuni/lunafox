# security/router

## Structure
- Public entry: `routes.go` (`RegisterSecurityRoutes`)
- Resource routes: `vulnerabilities.go` (private `registerVulnerabilityRoutes`)

## Constraints
- External callers should register security routes only through `RegisterSecurityRoutes`
- Keep resource-level registration helpers private
