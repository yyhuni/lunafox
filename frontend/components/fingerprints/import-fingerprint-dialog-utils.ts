export type FingerprintType = "ehole" | "goby" | "wappalyzer" | "fingers" | "fingerprinthub" | "arl"

export type FingerprintConfig = {
  title: string
  description: string
  formatHint: string
  validate: (json: unknown) => { valid: boolean; error?: string }
}

type TranslationFn = ((key: string, params?: Record<string, string | number | Date>) => string) & {
  raw: (key: string) => string
}

export function buildFingerprintConfig(t: TranslationFn): Record<FingerprintType, FingerprintConfig> {
  return {
    ehole: {
      title: t("import.eholeTitle"),
      description: t("import.eholeDesc"),
      formatHint: t.raw("import.eholeFormatHint") as string,
      validate: (json) => {
        if (!json || typeof json !== "object") {
          return { valid: false, error: t("import.eholeInvalidFields") }
        }
        const obj = json as Record<string, unknown>
        if (!obj.fingerprint) {
          return { valid: false, error: t("import.eholeInvalidMissing") }
        }
        if (!Array.isArray(obj.fingerprint)) {
          return { valid: false, error: t("import.eholeInvalidArray") }
        }
        if (obj.fingerprint.length === 0) {
          return { valid: false, error: t("import.emptyData") }
        }
        const first = obj.fingerprint[0] as Record<string, unknown>
        if (!first.cms || !first.keyword) {
          return { valid: false, error: t("import.eholeInvalidFields") }
        }
        return { valid: true }
      },
    },
    goby: {
      title: t("import.gobyTitle"),
      description: t("import.gobyDesc"),
      formatHint: t.raw("import.gobyFormatHint") as string,
      validate: (json) => {
        if (Array.isArray(json)) {
          if (json.length === 0) {
            return { valid: false, error: t("import.emptyData") }
          }
          const first = json[0] as Record<string, unknown>
          if (!first.product || !first.rule) {
            return { valid: false, error: t("import.gobyInvalidFields") }
          }
        } else if (typeof json === "object" && json !== null) {
          if (Object.keys(json).length === 0) {
            return { valid: false, error: t("import.emptyData") }
          }
        } else {
          return { valid: false, error: t("import.gobyInvalidFormat") }
        }
        return { valid: true }
      },
    },
    wappalyzer: {
      title: t("import.wappalyzerTitle"),
      description: t("import.wappalyzerDesc"),
      formatHint: t.raw("import.wappalyzerFormatHint") as string,
      validate: (json) => {
        if (Array.isArray(json)) {
          if (json.length === 0) {
            return { valid: false, error: t("import.emptyData") }
          }
          return { valid: true }
        }
        if (!json || typeof json !== "object") {
          return { valid: false, error: t("import.wappalyzerInvalidFormat") }
        }
        const obj = json as Record<string, unknown>
        const apps = obj.apps || obj.technologies
        if (apps) {
          if (typeof apps !== "object" || Array.isArray(apps)) {
            return { valid: false, error: t("import.wappalyzerInvalidApps") }
          }
          if (Object.keys(apps).length === 0) {
            return { valid: false, error: t("import.emptyData") }
          }
          return { valid: true }
        }
        if (typeof json === "object" && json !== null) {
          if (Object.keys(json).length === 0) {
            return { valid: false, error: t("import.emptyData") }
          }
          return { valid: true }
        }
        return { valid: false, error: t("import.wappalyzerInvalidFormat") }
      },
    },
    fingers: {
      title: t("import.fingersTitle"),
      description: t("import.fingersDesc"),
      formatHint: t.raw("import.fingersFormatHint") as string,
      validate: (json) => {
        if (!Array.isArray(json)) {
          return { valid: false, error: t("import.fingersInvalidArray") }
        }
        if (json.length === 0) {
          return { valid: false, error: t("import.emptyData") }
        }
        const first = json[0] as Record<string, unknown>
        if (!first.name || !first.rule) {
          return { valid: false, error: t("import.fingersInvalidFields") }
        }
        return { valid: true }
      },
    },
    fingerprinthub: {
      title: t("import.fingerprinthubTitle"),
      description: t("import.fingerprinthubDesc"),
      formatHint: t.raw("import.fingerprinthubFormatHint") as string,
      validate: (json) => {
        if (!Array.isArray(json)) {
          return { valid: false, error: t("import.fingerprinthubInvalidArray") }
        }
        if (json.length === 0) {
          return { valid: false, error: t("import.emptyData") }
        }
        const first = json[0] as Record<string, unknown>
        if (!first.id || !first.info) {
          return { valid: false, error: t("import.fingerprinthubInvalidFields") }
        }
        return { valid: true }
      },
    },
    arl: {
      title: t("import.arlTitle"),
      description: t("import.arlDesc"),
      formatHint: t.raw("import.arlFormatHint") as string,
      validate: (json) => {
        if (!Array.isArray(json)) {
          return { valid: false, error: t("import.arlInvalidArray") }
        }
        if (json.length === 0) {
          return { valid: false, error: t("import.emptyData") }
        }
        const first = json[0] as Record<string, unknown>
        if (!first.name || !first.rule) {
          return { valid: false, error: t("import.arlInvalidFields") }
        }
        return { valid: true }
      },
    },
  }
}

const parseAcceptedFileTypes = (value?: string): string[] => {
  if (!value) return []
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean)
}

const buildAcceptConfig = (extensions: string[]): Record<string, string[]> => {
  const accept: Record<string, string[]> = {}
  const pushExt = (key: string, ext: string) => {
    if (!accept[key]) accept[key] = []
    accept[key].push(ext)
  }

  extensions.forEach((rawExt) => {
    const normalized = rawExt.startsWith(".") ? rawExt : `.${rawExt}`
    if (normalized === ".json") {
      pushExt("application/json", normalized)
      return
    }
    if (normalized === ".yaml" || normalized === ".yml") {
      pushExt("application/x-yaml", normalized)
      pushExt("text/yaml", normalized)
      return
    }
    pushExt("application/octet-stream", normalized)
  })

  return accept
}

export function getAcceptConfig(
  fingerprintType: FingerprintType,
  acceptedFileTypes?: string
): Record<string, string[]> {
  const customExtensions = parseAcceptedFileTypes(acceptedFileTypes)
  if (customExtensions.length > 0) {
    return buildAcceptConfig(customExtensions)
  }
  if (fingerprintType === "arl") {
    return {
      "application/json": [".json"],
      "application/x-yaml": [".yaml", ".yml"],
      "text/yaml": [".yaml", ".yml"],
    }
  }
  return { "application/json": [".json"] }
}
