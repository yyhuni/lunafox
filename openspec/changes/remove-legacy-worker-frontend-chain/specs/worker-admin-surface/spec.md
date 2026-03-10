## ADDED Requirements

### Requirement: Workers settings surface MUST use the supported agent contract
The runtime `/settings/workers/` surface MUST use the supported agent management contract and MUST not depend on an unsupported `/workers` frontend HTTP chain.

#### Scenario: workers settings page loads through agent APIs
- **GIVEN** a user opens the workers settings page
- **WHEN** the page fetches runtime data
- **THEN** it uses the agent query chain backed by `/admin/agents`
- **AND** it does not require a frontend `/workers` service layer

### Requirement: Unsupported legacy worker frontend chain MUST be removed
The repository MUST remove unsupported legacy frontend worker service and hook files that target non-existent backend worker routes.

#### Scenario: repository no longer carries orphan `/workers` frontend service
- **GIVEN** the frontend repository is searched for supported runtime worker HTTP access
- **WHEN** the cleanup is complete
- **THEN** no supported runtime worker service or hook targets `/workers`
- **AND** the active workers runtime surface remains available through the agent contract

### Requirement: Unsupported worker-only placeholder UI MUST not remain in runtime flow
The repository MUST remove or isolate worker-only placeholder UI artifacts that depend on backend deploy routes which do not exist.

#### Scenario: deploy terminal placeholder no longer blocks contract clarity
- **GIVEN** a frontend worker-only UI artifact depends on `/ws/workers/:id/deploy/`
- **WHEN** the cleanup is applied
- **THEN** the artifact is removed from runtime flow or migrated to a supported backend contract
- **AND** the repository no longer implies that the unsupported backend route exists
