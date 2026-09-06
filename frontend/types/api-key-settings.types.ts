/**
 * Registry-backed API key configuration for subfinder third-party data sources.
 */

export type ProviderStatus =
  | "configured"
  | "enabled"
  | "disabled"
  | "unconfigured"
  | "pending"
  | "requiresReconfiguration"
  | "unsupported"

export interface ApiKeyFieldValue {
  value?: string
  configured?: boolean
  maskedValue?: string
}

export interface ApiKeyProviderState {
  enabled: boolean
  status: ProviderStatus
  values: Record<string, ApiKeyFieldValue>
}

export interface ApiKeyProviderFieldDefinition {
  name: string
  secret: boolean
  required: boolean
}

export interface ApiKeyProviderDefinition {
  key: string
  sourceName: string
  displayName: string
  docsUrl?: string
  keyRequirement: "required" | "optional"
  default: boolean
  recursive: boolean
  fields: ApiKeyProviderFieldDefinition[]
}

export interface ApiKeySettings {
  providers: Record<string, ApiKeyProviderState>
  definitions: ApiKeyProviderDefinition[]
}

export interface ApiKeyProviderUpdate {
  enabled: boolean
  status?: ProviderStatus
  values: Record<string, string>
}

export interface ApiKeySettingsUpdateRequest {
  providers: Record<string, ApiKeyProviderUpdate>
}

export type ProviderKey = string
