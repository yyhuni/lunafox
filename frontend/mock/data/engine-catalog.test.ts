import { beforeEach, describe, expect, it } from "vitest"

import { getMockEngineCatalog, getMockEngineCatalogDetail, installMockEngine, resetMockEngineCatalog } from "./engine-catalog"

const firstRef = "registry.example/team/engine@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const replacementRef = "registry.example/team/engine@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"

describe("mock Engine catalog installation", () => {
  beforeEach(resetMockEngineCatalog)

  it("models generic Engine installation, idempotency, and explicit replacement", () => {
    const installed = installMockEngine({ artifactRef: firstRef, allowReplacement: false })
    expect(installed).toMatchObject({ engine: { engineId: "engine.example.operator_scanner", publisher: "example", artifactRef: firstRef } })
    expect(getMockEngineCatalog()).toContainEqual(expect.objectContaining({ engineId: "engine.example.operator_scanner" }))
    expect(installMockEngine({ artifactRef: firstRef, allowReplacement: false })).toEqual(installed)

    expect(installMockEngine({ artifactRef: replacementRef, allowReplacement: false })).toMatchObject({
      conflict: {
        engineId: "engine.example.operator_scanner",
        proposedPackageDigest: replacementRef.slice(replacementRef.indexOf("@") + 1),
      },
    })
    expect(installMockEngine({ artifactRef: replacementRef, allowReplacement: true })).toMatchObject({
      engine: { artifactRef: replacementRef },
    })
  })

  it("rejects sources that the real custom method rejects before pull", () => {
    expect(installMockEngine({ artifactRef: "registry.example/team/engine:latest", allowReplacement: false })).toEqual({
      error: "A canonical OCI digest reference is required",
    })
    expect(installMockEngine({ artifactRef: firstRef })).toEqual({
      error: "artifactRef and allowReplacement are required",
    })
  })
})

describe("mock Screenshot Engine contract", () => {
  it("exposes the production Screenshot configuration and user-facing limits", () => {
    const engine = getMockEngineCatalogDetail("engine.lunafox.screenshot")
    expect("inputs" in (engine?.execution ?? {})).toBe(false)
    expect(engine?.execution.supportedTargetTypes).toEqual(["domain", "ip", "cidr"])
    expect(engine?.execution.configSections).toEqual([{
      id: "capture",
      defaultEnabled: true,
      params: [
        { key: "page-timeout", type: "integer", default: 15, minimum: 1, maximum: 120 },
        { key: "concurrency", type: "integer", default: 5, minimum: 1, maximum: 20 },
        { key: "retries", type: "integer", default: 1, minimum: 0, maximum: 3 },
      ],
    }])
  })
})

describe("mock Directory Scan Engine contract", () => {
  it("exposes the production FFUF configuration surface", () => {
    const engine = getMockEngineCatalogDetail("engine.lunafox.directory_scan")
    expect(engine?.execution).toEqual({
      engineApiMajor: 2,
      supportedTargetTypes: ["domain", "ip", "cidr"],
      configSections: [{
        id: "ffuf",
        defaultEnabled: true,
        params: [
          { key: "wordlist", type: "string", default: "dir_default.txt", resource: { kind: "wordlist" } },
          { key: "recursion", type: "boolean", default: false },
          { key: "recursion-depth", type: "integer", default: 1, minimum: 1, maximum: 5 },
          { key: "recursion-strategy", type: "string", default: "default", enum: ["default", "greedy"] },
          { key: "auto-calibration", type: "boolean", default: true },
          { key: "auto-calibration-mode", type: "string", default: "ac", enum: ["ac", "ach"] },
          { key: "match-codes", type: "string", default: "200-299,301,302,307,401,403,405,500" },
          { key: "concurrency", type: "integer", default: 5, minimum: 1, maximum: 20 },
          { key: "threads", type: "integer", default: 10, minimum: 1, maximum: 100 },
          { key: "rate", type: "integer", default: 0, minimum: 0, maximum: 1000 },
          { key: "delay", type: "string", default: "0.1-2.0" },
          { key: "request-timeout", type: "integer", default: 10, minimum: 1, maximum: 120 },
          { key: "timeout", type: "integer", default: 86400, minimum: 60, maximum: 604800 },
          { key: "follow-redirects", type: "boolean", default: false },
          { key: "http2", type: "boolean", default: false },
        ],
      }],
    })
  })
})

describe("mock Nuclei vulnerability Engine contract", () => {
  it("exposes the Website and Endpoint scope configuration without unsupported controls", () => {
    const engine = getMockEngineCatalogDetail("engine.lunafox.nuclei_vulnerability")
    expect(engine?.execution.supportedTargetTypes).toEqual(["domain", "ip", "cidr"])
    expect(engine?.execution.configSections).toEqual([{
      id: "nuclei",
      defaultEnabled: true,
      params: [
        { key: "scan-targets", type: "stringArray", default: ["website"], enum: ["website", "endpoint"], minItems: 1, maxItems: 2 },
        { key: "timeout", type: "integer", default: 3600, minimum: 60, maximum: 604800 },
        { key: "concurrency", type: "integer", default: 25, minimum: 1, maximum: 100 },
        { key: "rate-limit", type: "integer", default: 150, minimum: 1, maximum: 1000 },
        { key: "request-timeout", type: "integer", default: 5, minimum: 1, maximum: 120 },
        { key: "bulk-size", type: "integer", default: 25, minimum: 1, maximum: 100 },
        { key: "retries", type: "integer", default: 1, minimum: 0, maximum: 5 },
        { key: "severity", type: "stringArray", default: ["medium", "high", "critical"], enum: ["info", "low", "medium", "high", "critical"] },
        { key: "tags", type: "stringArray", default: [] },
        { key: "exclude-tags", type: "stringArray", default: [] },
      ],
    }])
  })
})

describe("mock Subdomain Discovery Engine contract", () => {
  it("exposes independently owned resolver Wordlists and required Resolve", () => {
    const engine = getMockEngineCatalogDetail("engine.lunafox.subdomain_discovery")
    expect(engine?.execution.configSections).toEqual(expect.arrayContaining([
      expect.objectContaining({
        id: "bruteforce",
        params: expect.arrayContaining([
          expect.objectContaining({ key: "wordlist", resource: { kind: "wordlist" } }),
          expect.objectContaining({ key: "resolvers", default: "resolvers.txt", resource: { kind: "wordlist" } }),
          expect.objectContaining({ key: "wildcard-filter", type: "boolean", default: false }),
        ]),
      }),
      expect.objectContaining({
        id: "resolve",
        defaultEnabled: true,
        requiredEnabled: true,
        params: expect.arrayContaining([
          expect.objectContaining({ key: "resolvers", default: "resolvers.txt", resource: { kind: "wordlist" } }),
          expect.objectContaining({ key: "wildcard-filter", type: "boolean", default: false }),
        ]),
      }),
    ]))
    expect(engine?.execution.configSections.some((section) => section.id === "dns")).toBe(false)
  })
})
