/**
 * Date formatting utilities with locale support
 * Maps next-intl locale to toLocaleString locale
 */

/**
 * Maps next-intl locale (zh, en) to toLocaleString locale (zh-CN, en-US)
 */
export function getDateLocale(locale: string): string {
  const localeMap: Record<string, string> = {
    zh: "zh-CN",
    en: "en-US",
  }
  return localeMap[locale] || "en-US"
}

/**
 * Format date with locale support
 * @param dateString - ISO date string or Date object
 * @param locale - next-intl locale (zh, en)
 * @param options - Intl.DateTimeFormatOptions
 */
export function formatDateByLocale(
  dateString: string | Date | null | undefined,
  locale: string,
  options?: Intl.DateTimeFormatOptions
): string {
  if (!dateString) return "-"
  try {
    const date = typeof dateString === "string" ? new Date(dateString) : dateString
    if (isNaN(date.getTime())) return "-"
    
    const defaultOptions: Intl.DateTimeFormatOptions = {
      year: "numeric",
      month: "numeric",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hour12: false,
    }
    
    return date.toLocaleString(getDateLocale(locale), options || defaultOptions)
  } catch {
    return typeof dateString === "string" ? dateString : "-"
  }
}
