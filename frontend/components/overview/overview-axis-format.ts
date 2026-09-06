const AXIS_UNIT_THRESHOLDS = {
  wan: 10_000,
  yi: 100_000_000,
} as const

const SHORT_DATE_FORMATTERS = {
  en: new Intl.DateTimeFormat("en-US", { day: "numeric", month: "numeric", timeZone: "UTC" }),
  zh: new Intl.DateTimeFormat("zh-CN", { day: "numeric", month: "numeric", timeZone: "UTC" }),
} as const

function formatCompactValue(value: number, locale: string) {
  return new Intl.NumberFormat(locale, {
    notation: "compact",
    maximumFractionDigits: 1,
  }).format(value)
}

export function formatAssetTrendAxisTick(value: number, maxValue: number, locale = "zh") {
  const absoluteMaxValue = Math.abs(maxValue)

  if (absoluteMaxValue < AXIS_UNIT_THRESHOLDS.wan) {
    return value.toLocaleString(locale)
  }

  return formatCompactValue(value, locale)
}

export function formatOverviewCompactNumber(value: number, locale = "zh") {
  const absoluteValue = Math.abs(value)

  if (absoluteValue < AXIS_UNIT_THRESHOLDS.wan) {
    return value.toLocaleString(locale)
  }

  return formatCompactValue(value, locale)
}

export function formatOverviewSignedChange(value: number, locale = "zh") {
  const sign = value < 0 ? "-" : "+"

  return `${sign}${Math.abs(value).toLocaleString(locale)}`
}

export function formatOverviewDateTick(value: number | string, locale = "zh") {
  const date = new Date(value)

  if (Number.isNaN(date.getTime())) {
    return String(value)
  }

  return SHORT_DATE_FORMATTERS[locale === "zh" ? "zh" : "en"].format(date)
}
