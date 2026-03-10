## ADDED Requirements

### Requirement: Workflow artifact responsibilities MUST remain distinct
The system MUST keep workflow artifact responsibilities distinct so that the same workflow fact is not hand-maintained across manifest, schema, and profile.

#### Scenario: Validation constraints belong to schema
- **GIVEN** a workflow parameter has range, enum, pattern, or length constraints
- **WHEN** workflow artifacts are generated or consumed
- **THEN** schema is the authoritative validation artifact
- **AND** manifest does not become a second validation source of truth

#### Scenario: Default values belong to generated default profile
- **GIVEN** a workflow contract defines default values
- **WHEN** artifacts are generated
- **THEN** a default profile artifact is produced from those defaults
- **AND** manifest does not require hand-maintained defaultValue copies

### Requirement: Manifest MUST focus on catalog and orchestration metadata
The workflow manifest MUST focus on identity, display metadata, supported target types, and stage/tool orchestration description.

#### Scenario: Catalog reads thin manifest
- **GIVEN** server loads workflow metadata for catalog display
- **WHEN** manifest is parsed
- **THEN** server obtains workflow identity, display text, target type support, default profile reference, and stage/tool structure
- **AND** server does not depend on manifest as a validation source

### Requirement: Scenario profiles MUST be sparse overlays over default
The system MUST represent non-default profiles as sparse overlays over the generated default profile.

#### Scenario: Scenario profile only defines differences
- **GIVEN** a workflow defines a generated default profile
- **AND** a `fast` profile exists
- **WHEN** profile artifacts are generated
- **THEN** the `fast` source only contains values different from default
- **AND** the generated merged profile remains schema-valid
