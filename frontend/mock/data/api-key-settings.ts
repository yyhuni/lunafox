import type {
  ApiKeyProviderDefinition,
  ApiKeyProviderState,
  ApiKeySettings,
  ApiKeySettingsUpdateRequest,
} from "@/types/api-key-settings.types"

const requiredProviders = [
  "alienvault",
  "bevigil",
  "bufferover",
  "builtwith",
  "c99",
  "censys",
  "certspotter",
  "chaos",
  "chinaz",
  "digitalyama",
  "dnsdb",
  "dnsdumpster",
  "dnsrepo",
  "domainsproject",
  "driftnet",
  "facebook",
  "fofa",
  "fullhunt",
  "github",
  "intelx",
  "merklemap",
  "netlas",
  "onyphe",
  "profundis",
  "pugrecon",
  "quake",
  "redhuntlabs",
  "robtex",
  "rsecloud",
  "securitytrails",
  "shodan",
  "threatbook",
  "virustotal",
  "whoisxmlapi",
  "windvane",
  "zoomeyeapi",
]

const optionalProviders = ["hackertarget", "leakix", "reconeer"]

const compositeFields: Record<string, string[]> = {
  censys: ["pat", "orgId"],
  dnsrepo: ["token", "apiKey"],
  domainsproject: ["username", "password"],
  facebook: ["appId", "appSecret"],
  fofa: ["email", "apiKey"],
  intelx: ["host", "apiKey"],
  redhuntlabs: ["baseUrl", "blobrKey"],
  zoomeyeapi: ["host", "apiKey"],
}

const secretFieldNames = new Set(["apiKey", "pat", "token", "password", "appSecret", "blobrKey"])

const definitions: ApiKeyProviderDefinition[] = [...requiredProviders, ...optionalProviders].map((key) => ({
  key,
  sourceName: key,
  displayName: providerDisplayName(key),
  keyRequirement: optionalProviders.includes(key) ? "optional" : "required",
  default: true,
  recursive: false,
  fields: (compositeFields[key] ?? ["apiKey"]).map((name) => ({
    name,
    secret: secretFieldNames.has(name),
    required: !(optionalProviders.includes(key) && name === "apiKey") && !(key === "censys" && name === "orgId"),
  })),
}))

export const initialMockApiKeySettings: ApiKeySettings = {
  definitions,
  providers: Object.fromEntries(
    definitions.map((definition) => [definition.key, createProviderState(definition)])
  ),
}

initialMockApiKeySettings.providers.fofa = {
  enabled: true,
  status: "configured",
  values: {
    email: { value: "security@example.test", configured: true },
    apiKey: { configured: true, maskedValue: "********" },
  },
}
initialMockApiKeySettings.providers.shodan = {
  enabled: true,
  status: "configured",
  values: { apiKey: { configured: true, maskedValue: "********" } },
}

const mockApiKeySettings: ApiKeySettings = cloneApiKeySettings(initialMockApiKeySettings)

function createProviderState(definition: ApiKeyProviderDefinition): ApiKeyProviderState {
  return {
    enabled: false,
    status: "unconfigured",
    values: Object.fromEntries(
      definition.fields.map((field) => [field.name, field.secret ? { configured: false } : { value: "", configured: false }])
    ),
  }
}

function cloneApiKeySettings(settings: ApiKeySettings): ApiKeySettings {
  return {
    definitions: settings.definitions.map((definition) => ({
      ...definition,
      fields: definition.fields.map((field) => ({ ...field })),
    })),
    providers: Object.fromEntries(
      Object.entries(settings.providers).map(([key, provider]) => [
        key,
        {
          ...provider,
          values: Object.fromEntries(Object.entries(provider.values).map(([fieldName, value]) => [fieldName, { ...value }])),
        },
      ])
    ),
  }
}

export function getMockApiKeySettings(): ApiKeySettings {
  return cloneApiKeySettings(mockApiKeySettings)
}

export function updateMockApiKeySettings(settings: ApiKeySettingsUpdateRequest): ApiKeySettings {
  for (const [providerKey, update] of Object.entries(settings.providers ?? {})) {
    const previous = mockApiKeySettings.providers[providerKey]
    if (!previous) continue
    mockApiKeySettings.providers[providerKey] = {
      enabled: update.enabled,
      status: update.enabled ? "configured" : "unconfigured",
      values: Object.fromEntries(
        Object.entries(update.values).map(([fieldName, value]) => [
          fieldName,
          secretFieldNames.has(fieldName)
            ? { configured: value.length > 0, maskedValue: value.length > 0 ? "********" : "" }
            : { value, configured: value.length > 0 },
        ])
      ),
    }
  }

  return getMockApiKeySettings()
}

function providerDisplayName(key: string): string {
  const names: Record<string, string> = {
    fofa: "FOFA",
    github: "GitHub",
    intelx: "Intelligence X",
    securitytrails: "SecurityTrails",
    virustotal: "VirusTotal",
    whoisxmlapi: "WhoisXML API",
    zoomeyeapi: "ZoomEye API",
  }
  return names[key] ?? key
}
