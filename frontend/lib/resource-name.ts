export function organizationName(id: number): string {
  return `organizations/${id}`
}

export function targetName(id: number): string {
  return `targets/${id}`
}

export function blacklistPolicyName(): string {
  return "blacklistPolicy"
}

export function targetBlacklistPolicyName(targetId: number): string {
  if (!Number.isSafeInteger(targetId) || targetId <= 0) {
    throw new Error(`Invalid target blacklistPolicy resource id: ${targetId}`)
  }
  return `targets/${targetId}/blacklistPolicy`
}

export function parseTargetName(name: string): number {
  return parseResourceNameId(name, "targets")
}

export function scanName(id: number): string {
  return `scans/${id}`
}

export function agentName(id: number): string {
  return `agents/${id}`
}

export function parseAgentName(name: string): number {
  return parseResourceNameId(name, "agents")
}

export function agentRegistrationTokenName(id: number): string {
  if (!Number.isSafeInteger(id) || id <= 0) {
    throw new Error(`Invalid agentRegistrationTokens resource id: ${id}`)
  }
  return `agentRegistrationTokens/${id}`
}

export function parseAgentRegistrationTokenName(name: string): number {
  return parseResourceNameId(name, "agentRegistrationTokens")
}

export function scanWorkflowName(id: string): string {
  return id.startsWith("scanWorkflows/") ? id : `scanWorkflows/${id}`
}

export function parseScanWorkflowName(name: string): string {
  const parts = name.split("/")
  if (parts.length !== 2 || parts[0] !== "scanWorkflows" || !parts[1]) {
    throw new Error(`Invalid scan workflow resource name: ${name}`)
  }
  return parts[1]
}

export function scheduledScanName(id: number): string {
  return `scheduledScans/${id}`
}

export function vulnerabilityName(id: number): string {
  return `vulnerabilities/${id}`
}

export function websiteName(targetId: number, id: number): string {
  return `targets/${targetId}/websites/${id}`
}

export function parseWebsiteName(name: string): { targetId: number; websiteId: number } {
  const parts = name.split("/")
  if (parts.length !== 4 || parts[0] !== "targets" || parts[2] !== "websites") {
    throw new Error(`Invalid website resource name: ${name}`)
  }

  const targetId = parseCanonicalPositiveInteger(parts[1], "target", name)
  const websiteId = parseCanonicalPositiveInteger(parts[3], "website", name)
  return { targetId, websiteId }
}

export function subdomainName(targetId: number, id: number): string {
  return `targets/${targetId}/subdomains/${id}`
}

export function subdomainSnapshotName(scanId: number, id: number): string {
  return `scans/${scanId}/subdomainSnapshots/${id}`
}

export function endpointName(targetId: number, id: number): string {
  return `targets/${targetId}/endpoints/${id}`
}

export function directoryName(targetId: number, id: number): string {
  return `targets/${targetId}/directories/${id}`
}

export function screenshotName(targetId: number, id: number): string {
  return `targets/${targetId}/screenshots/${id}`
}

function parseResourceNameId(name: string, collection: string): number {
  const parts = name.split("/")
  if (parts.length !== 2 || parts[0] !== collection) {
    throw new Error(`Invalid ${collection} resource name: ${name}`)
  }

  const id = Number.parseInt(parts[1], 10)
  if (!Number.isSafeInteger(id) || id <= 0 || String(id) !== parts[1]) {
    throw new Error(`Invalid ${collection} resource id: ${name}`)
  }

  return id
}

function parseCanonicalPositiveInteger(value: string, label: string, resourceName: string): number {
  const id = Number.parseInt(value, 10)
  if (!Number.isSafeInteger(id) || id <= 0 || String(id) !== value) {
    throw new Error(`Invalid ${label} resource id: ${resourceName}`)
  }
  return id
}
