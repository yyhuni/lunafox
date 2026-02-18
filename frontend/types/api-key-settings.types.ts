/**
 * API Key configuration type definition
 * For subfinder third-party data source configuration
 */

// Single field Provider configuration (hunter, shodan, zoomeye, securitytrails, threatbook, quake)
export interface SingleFieldProviderConfig {
  enabled: boolean
  apiKey: string
}

// FOFA Provider configuration (email + apiKey)
export interface FofaProviderConfig {
  enabled: boolean
  email: string
  apiKey: string
}

// Censys Provider configuration (apiId + apiSecret)
export interface CensysProviderConfig {
  enabled: boolean
  apiId: string
  apiSecret: string
}

// Complete API Key configuration
export interface ApiKeySettings {
  fofa: FofaProviderConfig
  hunter: SingleFieldProviderConfig
  shodan: SingleFieldProviderConfig
  censys: CensysProviderConfig
  zoomeye: SingleFieldProviderConfig
  securitytrails: SingleFieldProviderConfig
  threatbook: SingleFieldProviderConfig
  quake: SingleFieldProviderConfig
}

// Provider type
export type ProviderKey = keyof ApiKeySettings

// Provider configuration union type
export type ProviderConfig = FofaProviderConfig | CensysProviderConfig | SingleFieldProviderConfig
