import {
  Globe,
  Network,
  Monitor,
  Fingerprint,
  FolderSearch,
  Link,
  ShieldAlert,
  Shield,
  Camera,
  Search,
  Cpu,
} from "@/components/icons"
import type { LucideIcon } from "@/components/icons"

/** Unified capability tag color (using global CSS variables) */
const CAPABILITY_COLOR = "bg-primary/10 text-primary border-primary/20"

/**
 * Engine capability configuration (using global CSS colors)
 * Used for scan initiation, quick scan and other engine selection interfaces
 */
export const CAPABILITY_CONFIG: Record<string, { 
  label: string
  color: string
  icon: LucideIcon 
}> = {
  subdomain_discovery: { label: "Subdomain Discovery", color: CAPABILITY_COLOR, icon: Globe },
  port_scan: { label: "Port Scan", color: CAPABILITY_COLOR, icon: Network },
  site_scan: { label: "Site Scan", color: CAPABILITY_COLOR, icon: Monitor },
  fingerprint_detect: { label: "Fingerprint Detection", color: CAPABILITY_COLOR, icon: Fingerprint },
  directory_scan: { label: "Directory Scan", color: CAPABILITY_COLOR, icon: FolderSearch },
  url_fetch: { label: "URL Fetch", color: CAPABILITY_COLOR, icon: Link },
  vuln_scan: { label: "Vulnerability Scan", color: CAPABILITY_COLOR, icon: ShieldAlert },
  waf_detection: { label: "WAF Detection", color: CAPABILITY_COLOR, icon: Shield },
  screenshot: { label: "Screenshot", color: CAPABILITY_COLOR, icon: Camera },
  osint: { label: "OSINT", color: CAPABILITY_COLOR, icon: Search },
}

/**
 * Get main icon based on engine capabilities
 * Returns the first matching capability icon by priority
 */
export function getEngineIcon(capabilities: string[]): LucideIcon {
  const priorityOrder = [
    'vuln_scan', 
    'subdomain_discovery', 
    'port_scan', 
    'site_scan', 
    'directory_scan', 
    'url_fetch', 
    'waf_detection', 
    'screenshot', 
    'osint'
  ]
  
  for (const key of priorityOrder) {
    if (capabilities.includes(key)) {
      return CAPABILITY_CONFIG[key].icon
    }
  }
  return Cpu
}

/**
 * Parse engine configuration to get capability list
 * Only matches top-level YAML keys (not comments or nested content)
 */
export function parseEngineCapabilities(configuration: string): string[] {
  if (!configuration) return []

  try {
    const capabilities: string[] = []

    // Pre-compile RegExp patterns for each capability to avoid recreation in loop
    const patterns = Object.keys(CAPABILITY_CONFIG).map(key => ({
      key,
      pattern: new RegExp(`^${key}\\s*:`, 'm')
    }))

    // Test each pattern against the configuration
    for (const { key, pattern } of patterns) {
      if (pattern.test(configuration)) {
        capabilities.push(key)
      }
    }

    return capabilities
  } catch {
    return []
  }
}

/**
 * Merge multiple engine configurations into a single YAML string
 * Simply concatenates configurations with separators
 */
export function mergeEngineConfigurations(configurations: string[]): string {
  const validConfigs = configurations.filter(c => c && c.trim())
  if (validConfigs.length === 0) return ""
  if (validConfigs.length === 1) return validConfigs[0]
  return validConfigs.join("\n\n# ---\n\n")
}
