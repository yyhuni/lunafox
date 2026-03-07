import * as yaml from 'js-yaml'
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
import type { WorkflowConfiguration } from '@/types/workflow.types'

const CAPABILITY_COLOR = "bg-primary/10 text-primary border-primary/20"

export const CAPABILITY_CONFIG: Record<string, { label: string; color: string; icon: LucideIcon }> = {
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

export function getWorkflowIcon(capabilities: string[]): LucideIcon {
  const priorityOrder = [
    'vuln_scan',
    'subdomain_discovery',
    'port_scan',
    'site_scan',
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
  return Cpu
}

export function parseWorkflowConfiguration(input: WorkflowConfiguration | string | null | undefined): WorkflowConfiguration {
  if (!input) return {}
  if (typeof input !== 'string') {
    return cloneWorkflowConfiguration(input)
  }

  const parsed = yaml.load(input)
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    return {}
  }
  return cloneWorkflowConfiguration(parsed as WorkflowConfiguration)
}

export function normalizeWorkflowConfiguration(input: unknown): WorkflowConfiguration {
  try {
    return parseWorkflowConfiguration(input as WorkflowConfiguration | string | null | undefined)
  } catch {
    return {}
  }
}

export function serializeWorkflowConfiguration(configuration: WorkflowConfiguration | string | null | undefined): string {
  const normalized = normalizeWorkflowConfiguration(configuration)
  if (Object.keys(normalized).length === 0) return ''
  return String(yaml.dump(normalized, { lineWidth: -1, noRefs: true })).trim()
}

export function parseWorkflowCapabilities(configuration: WorkflowConfiguration | string | null | undefined): string[] {
  const normalized = normalizeWorkflowConfiguration(configuration)
  return Object.keys(normalized).filter((key) => key in CAPABILITY_CONFIG)
}

export function mergeWorkflowConfigurations(
  configurations: Array<WorkflowConfiguration | string | null | undefined>
): WorkflowConfiguration {
  const merged: WorkflowConfiguration = {}
  configurations.forEach((configuration) => {
    const normalized = normalizeWorkflowConfiguration(configuration)
    Object.entries(normalized).forEach(([key, value]) => {
      merged[key] = cloneWorkflowValue(value)
    })
  })
  return merged
}

export function extractWorkflowIds(configuration: WorkflowConfiguration | string | null | undefined): string[] {
  return Object.keys(normalizeWorkflowConfiguration(configuration))
}

function cloneWorkflowConfiguration(configuration: WorkflowConfiguration): WorkflowConfiguration {
  const cloned: WorkflowConfiguration = {}
  Object.entries(configuration).forEach(([key, value]) => {
    cloned[key] = cloneWorkflowValue(value)
  })
  return cloned
}

function cloneWorkflowValue(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map((item) => cloneWorkflowValue(item))
  }
  if (value && typeof value === 'object') {
    const cloned: Record<string, unknown> = {}
    Object.entries(value as Record<string, unknown>).forEach(([key, innerValue]) => {
      cloned[key] = cloneWorkflowValue(innerValue)
    })
    return cloned
  }
  return value
}
