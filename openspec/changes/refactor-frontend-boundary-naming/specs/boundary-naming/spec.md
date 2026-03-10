## ADDED Requirements

### Requirement: Frontend HTTP boundary MUST use camelCase contract names
The frontend HTTP boundary MUST use `camelCase` names for request parameters, request bodies, response DTOs, and pagination metadata when talking to the current Go backend API.

#### Scenario: audited service sends pagination and identity fields in camelCase
- **GIVEN** a frontend service file is classified as an audited HTTP boundary file
- **WHEN** it sends pagination or identity-related request fields to the current Go backend
- **THEN** it uses names such as `pageSize`, `targetId`, and `organizationId`
- **AND** it does not rely on `page_size`, `target_id`, or `organization_id`

### Requirement: Frontend compatibility MUST be isolated by audited scope
The repository MUST classify frontend files into audited boundary scope, legacy hold scope, and non-boundary scope before enforcing naming rules.

#### Scenario: legacy frontend module remains documented but not force-migrated
- **GIVEN** a frontend module cannot be confirmed as part of the current Go backend contract
- **WHEN** the repository applies frontend boundary naming governance
- **THEN** the module is documented as legacy hold scope instead of being force-migrated immediately
- **AND** the audited HTTP boundary scope remains enforceable without broad false positives

### Requirement: Automated naming checks MUST target audited frontend boundary files only
The repository MUST extend automated naming checks to audited frontend boundary files without applying HTTP naming rules to `proto`, generated files, or documented non-boundary areas.

#### Scenario: CI rejects a new snake_case DTO in audited frontend scope
- **GIVEN** a change introduces a new `snake_case` HTTP DTO or request field in an audited frontend boundary file
- **WHEN** the interface naming check runs in CI
- **THEN** the check fails with the matching rule name and file location
- **AND** `.proto`, generated contracts, DB / SQL related code, documented legacy hold files, and route-example comments remain excluded
