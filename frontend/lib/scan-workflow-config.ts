import * as yaml from 'js-yaml'
import { Search, semanticIcons, Shield } from "@/components/icons"
import type { Icon } from "@/components/icons"
import type { ScanWorkflowConfiguration } from '@/types/scan-workflow.types'

const CAPABILITY_COLOR = "bg-primary/10 text-primary border-primary/20"
const VulnerabilityIcon = semanticIcons.concept.vulnerability

export const CAPABILITY_CONFIG: Record<string, { label: string; color: string; icon: Icon }> = {
  subdomain_discovery: { label: "Subdomain Discovery", color: CAPABILITY_COLOR, icon: semanticIcons.concept.subdomain },
  port_scan: { label: "Port Scan", color: CAPABILITY_COLOR, icon: semanticIcons.concept.scan },
  website_discovery: { label: "Website Discovery", color: CAPABILITY_COLOR, icon: semanticIcons.concept.website },
  fingerprint_detect: { label: "Fingerprint Detection", color: CAPABILITY_COLOR, icon: semanticIcons.concept.fingerprint },
  directory_scan: { label: "Directory Scan", color: CAPABILITY_COLOR, icon: semanticIcons.concept.directory },
  url_fetch: { label: "URL Fetch", color: CAPABILITY_COLOR, icon: semanticIcons.concept.endpoint },
  vuln_scan: { label: "Vulnerability Scan", color: CAPABILITY_COLOR, icon: VulnerabilityIcon },
  waf_detection: { label: "WAF Detection", color: CAPABILITY_COLOR, icon: Shield },
  screenshot: { label: "Screenshot", color: CAPABILITY_COLOR, icon: semanticIcons.concept.screenshot },
  osint: { label: "OSINT", color: CAPABILITY_COLOR, icon: Search },
}

export function getScanWorkflowIcon(capabilities: string[]): Icon {
  const priorityOrder = [
    'vuln_scan',
    'subdomain_discovery',
    'port_scan',
    'website_discovery',
    'directory_scan',
    'url_fetch',
    'waf_detection',
    'screenshot',
    'osint',
  ]

  for (const key of priorityOrder) {
    if (capabilities.includes(key)) {
      return CAPABILITY_CONFIG[key].icon
    }
  }
  return semanticIcons.concept.engine
}

export function parseScanWorkflowConfiguration(input: ScanWorkflowConfiguration | string | null | undefined): ScanWorkflowConfiguration {
  if (!input) return {}
  if (typeof input !== 'string') {
    return cloneScanWorkflowConfiguration(input)
  }

  const parsed = yaml.load(input)
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    return {}
  }
  return cloneScanWorkflowConfiguration(parsed as ScanWorkflowConfiguration)
}

export function normalizeScanWorkflowConfiguration(input: unknown): ScanWorkflowConfiguration {
  try {
    return parseScanWorkflowConfiguration(input as ScanWorkflowConfiguration | string | null | undefined)
  } catch {
    return {}
  }
}

export function serializeScanWorkflowConfiguration(configuration: ScanWorkflowConfiguration | string | null | undefined): string {
  const normalized = normalizeScanWorkflowConfiguration(configuration)
  if (Object.keys(normalized).length === 0) return ''
  return String(yaml.dump(normalized, { lineWidth: -1, noRefs: true })).trim()
}

export function parseScanWorkflowCapabilities(configuration: ScanWorkflowConfiguration | string | null | undefined): string[] {
  const normalized = normalizeScanWorkflowConfiguration(configuration)
  const steps = extractScanWorkflowSteps(normalized)
  return Object.keys(steps).filter((key) => key in CAPABILITY_CONFIG)
}

export function mergeScanWorkflowConfigurations(
  configurations: Array<ScanWorkflowConfiguration | string | null | undefined>
): ScanWorkflowConfiguration {
  const mergedSteps: Record<string, unknown> = {}
  configurations.forEach((configuration) => {
    const normalized = normalizeScanWorkflowConfiguration(configuration)
    Object.entries(extractScanWorkflowSteps(normalized)).forEach(([key, value]) => {
      mergedSteps[key] = cloneScanWorkflowValue(value)
    })
  })
  if (Object.keys(mergedSteps).length === 0) {
    return {}
  }
  return { steps: mergedSteps }
}

export function extractScanWorkflowIds(configuration: ScanWorkflowConfiguration | string | null | undefined): string[] {
  return Object.keys(extractScanWorkflowSteps(normalizeScanWorkflowConfiguration(configuration)))
}

function extractScanWorkflowSteps(configuration: ScanWorkflowConfiguration): Record<string, unknown> {
  const steps = configuration.steps
  if (steps && typeof steps === 'object' && !Array.isArray(steps)) {
    return steps as Record<string, unknown>
  }
  return {}
}

function cloneScanWorkflowConfiguration(configuration: ScanWorkflowConfiguration): ScanWorkflowConfiguration {
  const clonedConfiguration: ScanWorkflowConfiguration = {}
  Object.entries(configuration).forEach(([key, value]) => {
    clonedConfiguration[key] = cloneScanWorkflowValue(value)
  })
  return clonedConfiguration
}

function cloneScanWorkflowValue(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map((item) => cloneScanWorkflowValue(item))
  }
  if (value && typeof value === 'object') {
    const clonedValue: Record<string, unknown> = {}
    Object.entries(value as Record<string, unknown>).forEach(([key, innerValue]) => {
      clonedValue[key] = cloneScanWorkflowValue(innerValue)
    })
    return clonedValue
  }
  return value
}
