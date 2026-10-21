import { getMockScenario } from "@/mock/scenarios"

type LicenseDto = {
  name: "agentLicenses/current"
  state: "inactive" | "active" | "expired"
  agentLimit: number
  months: number | null
  expiresAt: string | null
  licenseId: string | null
}

let activated: LicenseDto | null = null
let activatedScenario: string | null = null

// These fixtures were issued on 2026-10-19 UTC. A reload must not reissue a Key.
const DEMO_EXPIRY_BY_MONTHS: Record<number, string> = {
  1: "2026-11-19T00:00:00.000Z",
  2: "2026-12-19T00:00:00.000Z",
  3: "2027-01-19T00:00:00.000Z",
}

function licenseForMonths(months: number): LicenseDto {
  return {
    name: "agentLicenses/current", state: "active", agentLimit: 10,
    months, expiresAt: DEMO_EXPIRY_BY_MONTHS[months], licenseId: `LF-DEMO-${months}M`,
  }
}

export function resetMockAgentLicense() {
  activated = null
  activatedScenario = null
}

export function getMockAgentLicense(): LicenseDto {
  const scenario = getMockScenario()
  if (activatedScenario !== scenario) resetMockAgentLicense()
  if (activated) {
    const expired = Date.parse(activated.expiresAt!) <= Date.now()
    return { ...activated, state: expired ? "expired" : "active", agentLimit: expired ? 3 : activated.agentLimit }
  }
  if (scenario === "edge") return { ...licenseForMonths(1), state: "expired", agentLimit: 3, expiresAt: "2020-01-01T00:00:00.000Z" }
  if (scenario === "stress") {
    activated = licenseForMonths(3)
    activatedScenario = scenario
    return getMockAgentLicense()
  }
  return { name: "agentLicenses/current", state: "inactive", agentLimit: 3, months: null, expiresAt: null, licenseId: null }
}

export function activateMockAgentLicense(body: unknown): { license: LicenseDto } | { reason: string } {
  if (!body || typeof body !== "object" || Array.isArray(body) ||
      Object.keys(body).length !== 1 || !("key" in body) || typeof body.key !== "string" || !body.key.trim()) {
    return { reason: "LICENSE_KEY_REQUIRED" }
  }
  const key = body.key.trim().toUpperCase()
  if (key === "LF-DEMO-EXPIRED") return { reason: "LICENSE_KEY_EXPIRED" }
  if (key === "LF-DEMO-NOT-YET-VALID") return { reason: "LICENSE_KEY_NOT_YET_VALID" }
  const match = /^LF-DEMO-([123])M$/.exec(key)
  if (!match) return { reason: "LICENSE_KEY_INVALID" }
  const current = getMockAgentLicense()
  // Replaying the same demonstration Key must not restart its expiry clock.
  if (current.licenseId === key) {
    return current.state === "active" ? { license: current } : { reason: "LICENSE_KEY_EXPIRED" }
  }
  const candidate = licenseForMonths(Number(match[1]))
  if (Date.parse(candidate.expiresAt!) <= Date.now()) return { reason: "LICENSE_KEY_EXPIRED" }
  activated = candidate
  activatedScenario = getMockScenario()
  return { license: { ...activated } }
}
