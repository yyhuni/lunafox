/**
 * Scan engine type definitions
 * 
 * Backend actual return fields: id, name, configuration, created_at, updated_at
 */

// Scan engine interface (user-created, stored in database)
export interface ScanEngine {
  id: number
  name: string
  configuration?: string   // YAML configuration content
  isValid?: boolean        // Whether configuration is compatible with current schema
  createdAt: string
  updatedAt: string
}

// Preset engine interface (system-defined, read from files)
// Note: enabledFeatures is parsed by frontend from configuration using parseEngineCapabilities()
export interface PresetEngine {
  id: string               // e.g., "full_scan", "quick_scan"
  name: string             // Display name
  description?: string     // Brief description
  configuration: string    // YAML configuration content
}

// Create engine request
export interface CreateEngineRequest {
  name: string
  configuration: string
}

// Update engine request
export interface UpdateEngineRequest {
  name?: string
  configuration?: string
}

