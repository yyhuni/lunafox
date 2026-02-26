import { SEVERITY_STYLES } from "@/lib/severity-config"
import type { Vulnerability } from "@/types/vulnerability.types"
import type { VulnFilter, VulnSortConfig } from "./types"

// Memoized severity configuration for UI
export const SEVERITY_CONFIG = {
  critical: { className: SEVERITY_STYLES.critical.className, label: "Critical" },
  high: { className: SEVERITY_STYLES.high.className, label: "High" },
  medium: { className: SEVERITY_STYLES.medium.className, label: "Medium" },
  low: { className: SEVERITY_STYLES.low.className, label: "Low" },
  info: { className: SEVERITY_STYLES.info.className, label: "Info" },
} as const

/**
 * Filter vulnerabilities based on search query and status filter
 */
export function filterVulnerabilities(
  items: Vulnerability[],
  filter: VulnFilter,
  search: string
): Vulnerability[] {
  let filtered = items

  // 1. Status Filter
  if (filter === "pending") {
    filtered = filtered.filter((i) => !i.isReviewed)
  } else if (filter === "reviewed") {
    filtered = filtered.filter((i) => i.isReviewed)
  }

  // 2. Search Filter
  if (search.trim()) {
    const query = search.toLowerCase().trim()
    filtered = filtered.filter((item) => {
      return (
        item.vulnType.toLowerCase().includes(query) ||
        String(item.id).includes(query) ||
        item.source.toLowerCase().includes(query) ||
        item.url.toLowerCase().includes(query) ||
        (item.description && item.description.toLowerCase().includes(query))
      )
    })
  }

  return filtered
}

/**
 * Sort vulnerabilities based on configuration
 */
export function sortVulnerabilities(
  items: Vulnerability[],
  config: VulnSortConfig
): Vulnerability[] {
  return [...items].sort((a, b) => {
    let comparison = 0
    
    switch (config.field) {
      case "severity":
        // Custom severity order: critical > high > medium > low > info
        const severityOrder = { critical: 4, high: 3, medium: 2, low: 1, info: 0 }
        comparison = severityOrder[a.severity] - severityOrder[b.severity]
        break
      case "createdAt":
        comparison = new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime()
        break
      case "vulnType":
        comparison = a.vulnType.localeCompare(b.vulnType)
        break
      default:
        comparison = 0
    }

    return config.direction === "asc" ? comparison : -comparison
  })
}
