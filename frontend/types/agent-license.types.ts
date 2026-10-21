export type AgentLicense = {
  resourceName: "agentLicenses/current"
  state: "inactive" | "active" | "expired"
  agentLimit: number
  months: number | null
  expiresAt: string | null
  licenseId: string | null
}
