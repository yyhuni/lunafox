## ADDED Requirements

### Requirement: Workflow presets MUST be generated from contract definitions
The system MUST generate workflow preset configuration files from code-first workflow contract definitions, and MUST NOT treat hand-edited preset YAML as the long-term source of truth.

#### Scenario: Generate default preset from contract
- **GIVEN** a workflow contract definition is registered
- **WHEN** contract generation runs
- **THEN** a `default` preset configuration is generated for that workflow
- **AND** generated configuration conforms to the workflow schema version tuple

#### Scenario: Missing required default blocks generation
- **GIVEN** contract-derived default configuration lacks fields required by schema constraints
- **WHEN** preset generation runs
- **THEN** generation fails with explicit validation errors
- **AND** no invalid preset artifact is emitted

### Requirement: Scenario profiles MUST be expressed as sparse overlays
The system MUST represent optional scenario presets (for example `fast`/`deep`) as sparse overlay definitions over the `default` baseline.

#### Scenario: Overlay modifies only strategy knobs
- **GIVEN** a `default` preset exists for a workflow
- **AND** a `fast` profile overlay defines only a subset of fields
- **WHEN** generation merges `default + fast overlay`
- **THEN** final preset includes inherited baseline fields and overridden strategy fields
- **AND** unchanged fields are not duplicated in overlay sources

#### Scenario: Workflow without optional profiles remains valid
- **GIVEN** a workflow defines only `default` and no additional overlays
- **WHEN** generation runs
- **THEN** generation succeeds
- **AND** system does not require `fast` or `deep` profile artifacts

### Requirement: Generated presets MUST pass schema validation before publication
The system MUST validate each generated preset against workflow schema gates before writing or publishing artifacts.

#### Scenario: Overlay introduces unknown key
- **GIVEN** a profile overlay includes a key not defined by schema
- **WHEN** generation validates merged preset
- **THEN** generation fails with schema validation error
- **AND** invalid preset is not written to output directory

### Requirement: Preset generation MUST be deterministic and CI-enforced
The system MUST provide deterministic preset generation outputs and CI checks that fail when generated artifacts are out of date.

#### Scenario: CI detects stale preset artifacts
- **GIVEN** contract or overlay source changes without regenerated artifacts
- **WHEN** CI runs generation and diff checks
- **THEN** CI fails with non-empty diff
- **AND** required output directories include `server/internal/workflow/profile/profiles`

### Requirement: Pre-launch migration MUST cut over in one release without legacy preset compatibility paths
For pre-launch phase, the system MUST complete generated-preset migration in one release and MUST NOT keep long-lived legacy preset compatibility readers.

#### Scenario: One-release cutover removes legacy preset path
- **GIVEN** generated preset artifacts are ready and validated
- **WHEN** migration release is applied in pre-launch environment
- **THEN** runtime loads only generated preset artifacts as canonical source
- **AND** legacy hand-maintained preset compatibility paths are removed or disabled

#### Scenario: Profile endpoint remains available on new baseline
- **GIVEN** migration cutover is completed
- **WHEN** client requests `GET /api/workflows/profiles`
- **THEN** endpoint remains available and returns profiles from generated artifacts
- **AND** response semantics follow the new post-cutover baseline
