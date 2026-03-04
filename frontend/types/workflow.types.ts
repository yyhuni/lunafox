/**
 * Scan workflow template type definitions
 * 
 * Backend actual return fields: id, name, configuration, created_at, updated_at
 */

// User workflow template interface (stored in database)
export interface ScanWorkflow {
  id: number
  name: string
  configuration?: string   // YAML configuration content
  isValid?: boolean        // Whether configuration is compatible with current schema
  createdAt: string
  updatedAt: string
}

// Preset workflow interface (system-defined, read from files)
// Note: enabledFeatures is parsed by frontend from configuration using parseWorkflowCapabilities()
export interface PresetWorkflow {
  id: string               // e.g., "full_scan", "quick_scan"
  name: string             // Display name
  description?: string     // Brief description
  configuration: string    // YAML configuration content
}

// Create workflow template request
export interface CreateWorkflowRequest {
  name: string
  configuration: string
}

// Update workflow template request
export interface UpdateWorkflowRequest {
  name?: string
  configuration?: string
}
