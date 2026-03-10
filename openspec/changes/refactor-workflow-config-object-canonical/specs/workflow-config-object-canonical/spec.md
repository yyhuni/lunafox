## ADDED Requirements

### Requirement: Workflow configuration canonical form MUST be a structured object
The system MUST treat workflow configuration as a structured object across profile loading, catalog queries, scan creation, and internal application boundaries. It MUST NOT use YAML text as the canonical in-memory representation.

#### Scenario: Profile loader reads structured configuration
- **GIVEN** a workflow profile file declares `configuration` as a YAML mapping
- **WHEN** the profile loader reads and validates the file
- **THEN** the loaded profile contains a structured configuration object
- **AND** the loader does not require a second YAML parse from an embedded string field

#### Scenario: Scan create receives canonical object input
- **GIVEN** a scan create request includes workflow configuration
- **WHEN** the server validates and plans the scan
- **THEN** the server processes configuration as a structured object root keyed by `workflowId`
- **AND** workflow slice extraction operates on object nodes instead of reparsing YAML text

### Requirement: YAML text MUST be treated as a derived view, not the primary contract
The system MUST treat YAML as a human-editable projection for files, editors, import/export flows, and temporary protocol boundaries. It MUST NOT require YAML text as the primary API or persistence contract after cutover.

#### Scenario: Editor needs YAML text view
- **GIVEN** the frontend holds canonical workflow configuration as an object
- **WHEN** the user opens a YAML editor view
- **THEN** the UI serializes the object into YAML text for editing
- **AND** saving the editor parses the YAML back into the canonical object model before submission

#### Scenario: Runtime boundary still requires YAML temporarily
- **GIVEN** a task assignment boundary still expects workflow YAML text in the current runtime protocol
- **WHEN** the server emits an outbound task payload
- **THEN** the server may serialize the canonical workflow slice object into YAML text at that boundary only
- **AND** that serialized YAML is not persisted as the authoritative configuration source

### Requirement: Workflow configuration persistence MUST use canonical object storage
The system MUST persist scan-level and workflow-slice-level canonical configuration as JSON object data, and MUST NOT rely on text columns as the authoritative configuration store.

#### Scenario: Scan is created successfully
- **GIVEN** a valid workflow configuration object has passed merge, defaulting, and schema validation
- **WHEN** the scan and scan tasks are persisted
- **THEN** the scan stores the canonical configuration root as JSON object data
- **AND** each scan task stores its workflow slice as JSON object data

#### Scenario: Scan detail is queried later
- **GIVEN** a previously created scan exists in persistence
- **WHEN** the application loads scan detail or task projections
- **THEN** the canonical configuration is rehydrated from JSON object storage
- **AND** the application does not reconstruct canonical state by parsing persisted YAML text

### Requirement: Profile artifacts and overlay operations MUST be object-based
Generated profiles, manual profile fixtures, preset overlays, and defaulting flows MUST operate on object structures and MUST NOT rely on string concatenation or embedded YAML blocks.

#### Scenario: Generated profile output is written
- **GIVEN** the contract generator emits a workflow profile artifact
- **WHEN** it writes the profile YAML file
- **THEN** the outer profile document writes `configuration` as a YAML mapping
- **AND** it does not write a block-scalar string that contains a second YAML document

#### Scenario: Preset overlay modifies a subset of configuration
- **GIVEN** a default profile object and a sparse overlay object
- **WHEN** the system applies the overlay and runs schema validation
- **THEN** the merged result is computed structurally by object keys
- **AND** no string concatenation or regular-expression based merge is used as the authoritative behavior
