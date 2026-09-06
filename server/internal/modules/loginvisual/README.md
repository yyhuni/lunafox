# Login Visual Module

This module owns the singleton login-page visual configuration, draft/public publication boundary, and managed filesystem payload metadata. It never exposes a storage key through HTTP.

- Protected routes require the current database-backed active-superuser check.
- Authenticated discoverability check/unlock routes persist the cosmetic GitHub easter-egg state per account; they never grant media-management authority.
- GORM persistence records live under `repository/persistence`; the repository package keeps transaction behavior and domain mapping only.
- Draft media is available only through the protected preview route.
- Public routes resolve only the singleton published pointer and fail closed to the built-in login visual.
- Payloads live under `LOGIN_VISUALS_BASE_PATH`; PostgreSQL stores metadata and pointer state only.
