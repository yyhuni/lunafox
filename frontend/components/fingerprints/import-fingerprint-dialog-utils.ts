export const FINGERPRINT_IMPORT_MAX_FILE_SIZE = 30 * 1024 * 1024
type TranslationFn = ((key: string, params?: Record<string, string | number | Date>) => string) & { raw: (key: string) => string }
export type FingerprintConfig = { title: string; description: string; formatHint: string }
export function buildFingerprintConfig(t: TranslationFn): FingerprintConfig { return { title: t("import.fingerprinthubTitle"), description: t("import.fingerprinthubDesc"), formatHint: t.raw("import.fingerprinthubFormatHint") as string } }
export function getAcceptConfig(): Record<string, string[]> { return { "application/json": [".json"], "application/octet-stream": [".json"] } }
