import { cookies, headers } from "next/headers"

import { defaultLocale, isLocale, LOCALE_COOKIE_NAME, type Locale } from "@/i18n/config"

type LanguagePreference = {
  range: string
  quality: number
  order: number
}

function parseLanguagePreferences(acceptLanguage: string | null): LanguagePreference[] {
  if (!acceptLanguage) return []

  return acceptLanguage
    .split(",")
    .map((entry, order) => {
      const [rawRange, ...parameters] = entry.trim().split(";")
      const range = rawRange?.trim()
      if (!range || range === "*") return null

      const qualityParameter = parameters.find((parameter) => {
        return parameter.trim().toLowerCase().startsWith("q=")
      })
      const quality = qualityParameter ? Number(qualityParameter.trim().slice(2)) : 1

      if (!Number.isFinite(quality) || quality <= 0 || quality > 1) return null
      return { range, quality, order }
    })
    .filter((preference): preference is LanguagePreference => preference !== null)
    .sort((left, right) => right.quality - left.quality || left.order - right.order)
}

export function resolveAcceptLanguageLocale(acceptLanguage: string | null): Locale | undefined {
  for (const { range } of parseLanguagePreferences(acceptLanguage)) {
    try {
      const language = new Intl.Locale(range).language
      if (isLocale(language)) return language
    } catch {
      // Ignore malformed language ranges and continue with the next preference.
    }
  }

  return undefined
}

export async function resolveRequestLocale(): Promise<Locale> {
  const localeCookie = (await cookies()).get(LOCALE_COOKIE_NAME)?.value
  if (localeCookie && isLocale(localeCookie)) return localeCookie

  const acceptLanguage = (await headers()).get("accept-language")
  return resolveAcceptLanguageLocale(acceptLanguage) ?? defaultLocale
}

export async function loadLocaleMessages(locale: Locale) {
  return (await import(`../messages/${locale}.json`)).default
}
