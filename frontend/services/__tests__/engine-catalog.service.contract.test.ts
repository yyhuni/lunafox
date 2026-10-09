import { beforeEach, describe, expect, it, vi } from "vitest"

const apiClientMocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }))
vi.mock("@/lib/api-client", () => ({ default: apiClientMocks }))

import { getEngineCatalog, getEngineCatalogDetail, installEngine } from "@/services/engine-catalog.service"

describe("engine-catalog.service contract", () => {
  beforeEach(() => vi.clearAllMocks())

  it("normalizes the read-only engine list", async () => {
    apiClientMocks.get.mockResolvedValue({ data: [websiteEnginePayload(false)] })
    const result = await getEngineCatalog()
    expect(apiClientMocks.get).toHaveBeenCalledWith("/engines")
    expect(result[0]).toMatchObject({ name: "engines/engine.lunafox.website_discovery", engineId: "engine.lunafox.website_discovery" })
  })

  it("loads configuration detail by exact engine ID", async () => {
    apiClientMocks.get.mockResolvedValue({ data: websiteEnginePayload(true) })
    const result = await getEngineCatalogDetail("engine.lunafox.website_discovery")
    expect(apiClientMocks.get).toHaveBeenCalledWith("/engines/engine.lunafox.website_discovery")
    expect(result.execution.configSections[0]?.id).toBe("httpx")
    expect(result.execution.configSections[0]?.requiredEnabled).toBe(true)
    expect(result.execution.configSections[0]?.params[0]?.unit).toBe("seconds")
    expect("configSchema" in result).toBe(false)
  })

  it("posts one immutable artifact reference with explicit replacement intent", async () => {
    const artifactRef = "registry.example/team/engine@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
    apiClientMocks.post.mockResolvedValue({ data: websiteEnginePayload(true) })

    const result = await installEngine({ artifactRef, allowReplacement: false })

    expect(apiClientMocks.post).toHaveBeenCalledWith("/engines:install", { artifactRef, allowReplacement: false })
    expect(result.engineId).toBe("engine.lunafox.website_discovery")
  })

  it("rejects a whitespace-padded artifact reference before issuing an HTTP request", async () => {
    await expect(installEngine({
      artifactRef: " registry.example/team/engine@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      allowReplacement: false,
    })).rejects.toThrow("canonical OCI digest reference")
    expect(apiClientMocks.post).not.toHaveBeenCalled()
  })

  it("preserves engine.v5 execution declarations", async () => {
    apiClientMocks.get.mockResolvedValue({ data: engineV5Payload(true) })
    const result = await getEngineCatalogDetail("engine.lunafox.demo")

    expect(result.execution.engineApiMajor).toBe(2)
    expect("inputs" in result.execution).toBe(false)
    expect(result.execution.executionResources).toEqual(["subfinderProviderConfig"])
    expect("runtimeRef" in result.execution).toBe(false)
    expect("inputProfile" in result.execution).toBe(false)
  })

  it("rejects stale engine.v5 catalog payloads carrying execution input membership", async () => {
    apiClientMocks.get.mockResolvedValue({
      data: {
        ...engineV5Payload(true),
        execution: { ...engineV5Payload(true).execution, inputs: ["websiteURLs"] },
      },
    })

    await expect(getEngineCatalogDetail("engine.lunafox.demo")).rejects.toThrow(
      "Engine catalog engine.v5 execution must not include retired fields"
    )
  })

  it("rejects unsupported or misplaced parameter units", async () => {
    for (const params of [
      [{ key: "timeout", type: "integer", unit: "minutes" }],
      [{ key: "mode", type: "string", unit: "seconds" }],
    ]) {
      apiClientMocks.get.mockResolvedValue({
        data: {
          ...websiteEnginePayload(true),
          execution: {
            ...websiteEnginePayload(true).execution,
            configSections: [{ id: "httpx", params }],
          },
        },
      })
      await expect(getEngineCatalogDetail("engine.lunafox.website_discovery")).rejects.toThrow(
        /unsupported parameter unit|unit on a non-integer parameter/,
      )
    }
  })

  it("rejects the retired split execution resource fields", async () => {
    const payload = engineV5Payload(true)
    apiClientMocks.get.mockResolvedValue({
      data: {
        ...payload,
        execution: { ...payload.execution, platformResources: ["subfinderProviderConfig"] },
      },
    })

    await expect(getEngineCatalogDetail("engine.lunafox.demo")).rejects.toThrow(
      "Engine catalog engine.v5 execution must not include retired fields"
    )
  })

  it("preserves supported formats and omitted metadata in detail and install responses", async () => {
    const payload = websiteEnginePayload(true)
    const data = { ...payload, execution: { ...payload.execution, configSections: [{ id: "httpx", params: [
      { key: "request-fields", type: "stringArray", format: "http-headers", default: [] },
      { key: "headers", type: "stringArray", default: [] },
    ] }] } }
    apiClientMocks.get.mockResolvedValue({ data })
    apiClientMocks.post.mockResolvedValue({ data })
    for (const result of [await getEngineCatalogDetail(payload.engineId), await installEngine({ artifactRef: payload.artifactRef, allowReplacement: false })]) {
      expect(result.execution.configSections[0].params).toEqual(data.execution.configSections[0].params)
      expect(result.execution.configSections[0].params[1]).not.toHaveProperty("format")
    }
  })

  it("rejects unknown, misplaced, null, and enum-constrained formats at both response boundaries", async () => {
    const payload = websiteEnginePayload(true)
    for (const param of [
      { key: "fields", type: "stringArray", format: "unknown" },
      { key: "fields", type: "stringArray", format: "" },
      { key: "fields", type: "stringArray", format: null },
      { key: "fields", type: "string", format: "http-headers" },
      { key: "fields", type: "integer", format: "http-headers" },
      { key: "fields", type: "stringArray", format: "http-headers", enum: ["X: v"] },
    ]) {
      const data = { ...payload, execution: { ...payload.execution, configSections: [{ id: "httpx", params: [param] }] } }
      apiClientMocks.get.mockResolvedValue({ data })
      apiClientMocks.post.mockResolvedValue({ data })
      await expect(getEngineCatalogDetail(payload.engineId)).rejects.toThrow(/format/)
      await expect(installEngine({ artifactRef: payload.artifactRef, allowReplacement: false })).rejects.toThrow(/format/)
    }
  })

  it("rejects every manifest version except engine.v5", async () => {
    apiClientMocks.get.mockResolvedValue({
      data: [{ ...engineV5Payload(false), manifestVersion: "engine.v6" }],
    })

    await expect(getEngineCatalog()).rejects.toThrow("Unsupported engine manifest version: engine.v6")
  })

  it("rejects engine.v5 payloads carrying legacy execution fields", async () => {
    const payload = engineV5Payload(false)
    apiClientMocks.get.mockResolvedValue({
      data: [
        {
          ...payload,
          execution: {
            ...payload.execution,
            runtimeRef: "runtime.demo",
            inputProfile: "target",
          },
        },
      ],
    })

    await expect(getEngineCatalog()).rejects.toThrow(
      "Engine catalog engine.v5 execution must not include retired fields"
    )
  })

  it("rejects engine.v5 payloads missing required execution fields", async () => {
    const payload = engineV5Payload(false)
    apiClientMocks.get.mockResolvedValue({
      data: [
        {
          ...payload,
          execution: {
            engineApiMajor: payload.execution.engineApiMajor,
          },
        },
      ],
    })

    await expect(getEngineCatalog()).rejects.toThrow(
      "Engine catalog item is missing required fields"
    )
  })
})

function websiteEnginePayload(detail: boolean) {
  return {
    name: "engines/engine.lunafox.website_discovery",
    engineId: "engine.lunafox.website_discovery",
    manifestVersion: "engine.v5",
    publisher: "lunafox",
    packageVersion: "1.0.0",
    artifactRef: "docker.io/lunafox/lunafox-engine-website-discovery@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    packageDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    execution: {
      engineApiMajor: 2,
      supportedTargetTypes: ["domain"],
      ...(detail ? {
        configSections: [{
          id: "httpx",
          defaultEnabled: true,
          requiredEnabled: true,
          params: [{ key: "timeout", type: "integer", unit: "seconds", default: 60 }],
        }],
      } : {}),
    },
    localeResources: { zh: { engine: { displayName: "站点发现", description: "站点发现" } } },
	}
}

function engineV5Payload(detail: boolean) {
  return {
    name: "engines/engine.lunafox.demo",
    engineId: "engine.lunafox.demo",
    manifestVersion: "engine.v5",
    publisher: "lunafox",
    packageVersion: "1.2.3",
    artifactRef: "docker.io/lunafox/lunafox-engine-demo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    packageDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    execution: {
      engineApiMajor: 2,
      supportedTargetTypes: ["domain"],
      executionResources: ["subfinderProviderConfig"],
      ...(detail ? { configSections: [{ id: "scan", defaultEnabled: true, requiredEnabled: true, params: [] }] } : {}),
    },
    localeResources: {},
  }
}
