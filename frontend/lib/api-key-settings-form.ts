import type {
  ApiKeyFieldValue,
  ApiKeyProviderDefinition,
  ApiKeyProviderState,
  ApiKeyProviderUpdate,
  ApiKeySettings,
  ProviderStatus,
} from "@/types/api-key-settings.types"

export interface ApiKeyProviderFormConfig {
  enabled: boolean
  status: ProviderStatus
  values: Record<string, string>
  initialValues: Record<string, ApiKeyFieldValue>
  dirtyFields: Record<string, boolean>
}

export type ApiKeyProviderFormState = Record<string, ApiKeyProviderFormConfig>

function fieldConfigured(fieldValue: ApiKeyFieldValue | undefined): boolean {
  return fieldValue?.configured === true || Boolean(fieldValue?.maskedValue)
}

function initialFieldValue(
  field: ApiKeyProviderDefinition["fields"][number],
  fieldValue: ApiKeyFieldValue | undefined,
): ApiKeyFieldValue {
  if (field.secret) {
    // Secret responses intentionally never become editable plaintext state.
    return {
      configured: fieldConfigured(fieldValue),
      ...(fieldValue?.maskedValue ? { maskedValue: fieldValue.maskedValue } : {}),
    }
  }

  return {
    value: fieldValue?.value ?? "",
    configured: fieldConfigured(fieldValue),
  }
}

export function toApiKeyProviderFormState(settings: ApiKeySettings): ApiKeyProviderFormState {
  return Object.fromEntries(
    settings.definitions.map((definition) => {
      const providerState: ApiKeyProviderState | undefined = settings.providers[definition.key]
      const initialValues = Object.fromEntries(
        definition.fields.map((field) => [
          field.name,
          initialFieldValue(field, providerState?.values?.[field.name]),
        ]),
      )

      return [
        definition.key,
        {
          enabled: providerState?.enabled ?? false,
          status: providerState?.status ?? "unconfigured",
          values: Object.fromEntries(
            definition.fields.map((field) => [
              field.name,
              field.secret ? "" : providerState?.values?.[field.name]?.value ?? "",
            ]),
          ),
          initialValues,
          dirtyFields: {},
        },
      ]
    }),
  )
}

export function updateApiKeyProviderFormState(
  formData: ApiKeyProviderFormState,
  providerKey: string,
  field: string,
  value: string | boolean,
): ApiKeyProviderFormState {
  const current = formData[providerKey] ?? {
    enabled: false,
    status: "unconfigured" as ProviderStatus,
    values: {},
    initialValues: {},
    dirtyFields: {},
  }

  const updated = field === "enabled"
    ? { ...current, enabled: Boolean(value) }
    : {
        ...current,
        values: { ...current.values, [field]: String(value) },
        dirtyFields: { ...current.dirtyFields, [field]: true },
      }

  return { ...formData, [providerKey]: updated }
}

export function hasApiKeyProviderRequiredCredentials(
  provider: ApiKeyProviderDefinition,
  formData: ApiKeyProviderFormState,
): boolean {
  const config = formData[provider.key]
  if (!config) return false

  return provider.fields.every((field) => {
    if (!field.required) return true
    if (config.values[field.name]?.trim().length) return true
    return field.secret
      && !config.dirtyFields[field.name]
      && config.initialValues[field.name]?.configured === true
  })
}

export function getApiKeyProviderStatus(
  provider: ApiKeyProviderDefinition,
  formData: ApiKeyProviderFormState,
  dirtyProviderKeys: ReadonlySet<string>,
): ProviderStatus {
  if (dirtyProviderKeys.has(provider.key)) return "pending"

  const config = formData[provider.key]
  if (!config) return "unconfigured"
  if (config.status === "requiresReconfiguration" || config.status === "unsupported") {
    return config.status
  }
  if (!hasApiKeyProviderRequiredCredentials(provider, formData)) return "unconfigured"
  return config.enabled ? "enabled" : "disabled"
}

export function toApiKeyProviderUpdate(
  provider: ApiKeyProviderDefinition,
  config: ApiKeyProviderFormConfig,
): ApiKeyProviderUpdate {
  const values = Object.fromEntries(
    provider.fields
      .filter((field) => config.dirtyFields[field.name])
      .map((field) => [field.name, config.values[field.name] ?? ""]),
  )

  return { enabled: config.enabled, values }
}

export function getApiKeyProviderFieldPlaceholder(
  field: ApiKeyProviderDefinition["fields"][number],
  config: ApiKeyProviderFormConfig,
  fallback: string,
): string {
  if (!field.secret || config.dirtyFields[field.name]) return fallback
  return config.initialValues[field.name]?.maskedValue || fallback
}
