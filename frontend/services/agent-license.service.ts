import { api } from "@/lib/api-client"
import type { AgentLicense } from "@/types/agent-license.types"

export function parseAgentLicense(value: unknown): AgentLicense {
  const invalid = (): never => { throw new Error("Invalid Agent license response") }
  if (!value || typeof value !== "object" || Array.isArray(value)) return invalid()
  const row = value as Record<string, unknown>
  if (row.name !== "agentLicenses/current" ||
      !Number.isSafeInteger(row.agentLimit) || (row.agentLimit as number) < 1 ||
      !["inactive", "active", "expired"].includes(row.state as string)) return invalid()
  if (row.state === "inactive") {
    if (row.months !== null || row.expiresAt !== null || row.licenseId !== null) return invalid()
  } else if (!Number.isSafeInteger(row.months) || (row.months as number) < 1 ||
      typeof row.expiresAt !== "string" || !Number.isFinite(Date.parse(row.expiresAt)) ||
      typeof row.licenseId !== "string" || !row.licenseId.trim()) return invalid()
  return {
    resourceName: "agentLicenses/current",
    state: row.state as AgentLicense["state"],
    agentLimit: row.agentLimit as number,
    months: row.months as number | null,
    expiresAt: row.expiresAt as string | null,
    licenseId: row.licenseId as string | null,
  }
}

export async function activateAgentLicense(key: string): Promise<AgentLicense> {
  if (!key.trim()) throw new Error("Agent license Key is required")
  // Keep the signed Key intact; its fixed entitlement is verified by the Server.
  const response = await api.post<unknown>("/admin/agentLicenses/current:activate", { key: key.trim() })
  const license = parseAgentLicense(response.data)
  if (license.state !== "active") throw new Error("Activation did not return an active Agent license")
  return license
}
