import { formatDateByLocale } from "@/lib/date-utils"

export function formatAgentLicenseExpiry(expiresAt: string, locale: string): string {
  // Show the viewer's UTC offset so a fixed instant is not mistaken for an end-of-day deadline.
  return formatDateByLocale(expiresAt, locale, {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
    timeZoneName: "longOffset",
  })
}
