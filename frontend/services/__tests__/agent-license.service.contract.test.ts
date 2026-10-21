import { beforeEach, describe, expect, it, vi } from "vitest"
const mocks = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock("@/lib/api-client", () => ({ api: mocks }))
import { activateAgentLicense, parseAgentLicense } from "@/services/agent-license.service"

const dto = { name: "agentLicenses/current", state: "active", agentLimit: 10, months: 2, expiresAt: "2026-12-19T00:00:00.000Z", licenseId: "license-example" }

describe("Agent license service contract", () => {
  beforeEach(() => vi.clearAllMocks())
  it("uses the fixed resource custom method and normalizes resource identity", async () => {
    mocks.post.mockResolvedValue({ data: dto })
    const result = await activateAgentLicense(" example-key ")
    expect(mocks.post).toHaveBeenCalledWith("/admin/agentLicenses/current:activate", { key: "example-key" })
    expect(result).toEqual({ resourceName: "agentLicenses/current", state: "active", agentLimit: 10, months: 2, expiresAt: dto.expiresAt, licenseId: "license-example" })
    expect(result).not.toHaveProperty("name")
  })
  it("fails before transport for an empty Key", async () => {
    await expect(activateAgentLicense(" \n ")).rejects.toThrow("required")
    expect(mocks.post).not.toHaveBeenCalled()
  })
  it.each([
    { ...dto, name: "wrong" }, { ...dto, state: "unknown" }, { ...dto, agentLimit: undefined },
    { ...dto, expiresAt: "bad-date" }, { ...dto, licenseId: "" }, { ...dto, months: 0 }, null,
    { ...dto, state: "inactive" },
  ])("rejects malformed authorization data %#", (value) => {
    expect(() => parseAgentLicense(value)).toThrow("Invalid Agent license")
  })
  it("rejects an inactive response to an activation command", async () => {
    mocks.post.mockResolvedValue({ data: { ...dto, state: "inactive", months: null, expiresAt: null, licenseId: null } })
    await expect(activateAgentLicense("example-key")).rejects.toThrow("active Agent license")
  })
})
