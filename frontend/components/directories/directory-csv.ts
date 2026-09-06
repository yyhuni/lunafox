import { escapeCSV, formatDateForCSV } from "@/lib/csv-utils"
import type { Directory } from "@/types/directory.types"

export function buildDirectoryCSV(items: Directory[]): string {
  const headers = [
    "url",
    "status",
    "content_length",
    "content_type",
    "duration",
    "created_at",
  ]

  const rows = items.map((item) =>
    [
      escapeCSV(item.url),
      escapeCSV(item.status),
      escapeCSV(item.contentLength),
      escapeCSV(item.contentType),
      escapeCSV(item.duration),
      escapeCSV(formatDateForCSV(item.createdAt)),
    ].join(",")
  )

  return "\ufeff" + [headers.join(","), ...rows].join("\n")
}
