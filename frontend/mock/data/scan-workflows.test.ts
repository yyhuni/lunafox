import { describe, expect, it } from "vitest"

import { buildWorkflowWithEngineCatalog } from "@/lib/engine-catalog"
import { serializeCanonicalWorkflowConfiguration } from "@/lib/workflow-config"
import { getMockEngineCatalogDetail } from "./engine-catalog"
import {
  getMockCanonicalWorkflowConfiguration,
  getMockScanWorkflowByName,
  getMockScanWorkflowProfile,
} from "./scan-workflows"

describe("mock default Workflow contract", () => {
  it("keeps Directory Scan before the final Nuclei vulnerability Stage", () => {
    const workflow = getMockScanWorkflowByName("scanWorkflows/default")
    expect(workflow?.stages.map((stage) => stage.stageId)).toEqual([
      "discovery",
      "ports",
      "websites",
      "url_collection",
      "screenshot",
      "directory_scan",
      "nuclei_vulnerability",
    ])
    expect(workflow?.stages.at(-1)?.steps).toEqual([{
      stageId: "nuclei_vulnerability",
      stepId: "nuclei_vulnerability",
      engineId: "engine.lunafox.nuclei_vulnerability",
      profileDefaultEnabled: false,
    }])
  })

  it("enables every default Profile and request Step, including Screenshot and Directory", () => {
    const profile = getMockScanWorkflowProfile("scanWorkflows/default")
    expect(profile?.configuration.steps).toMatchObject({
      screenshot: {
        enabled: true,
        engineConfig: {
          capture: {
            enabled: true,
            "page-timeout": 15,
            concurrency: 5,
            retries: 1,
          },
        },
      },
      directory_scan: {
        enabled: true,
        engineConfig: {
          ffuf: {
            enabled: true,
            wordlist: "dir_default.txt",
            "match-codes": "200-299,301,302,307,401,403,405,500",
            delay: "0.1-2.0",
            timeout: 86400,
          },
        },
      },
      nuclei_vulnerability: {
        enabled: false,
        engineConfig: {
          nuclei: {
            enabled: true,
            "scan-targets": ["website"],
            timeout: 3600,
            concurrency: 25,
            "rate-limit": 150,
            "request-timeout": 5,
            "bulk-size": 25,
            retries: 1,
            severity: ["medium", "high", "critical"],
            tags: [],
            "exclude-tags": [],
          },
        },
      },
    })

    const request = getMockCanonicalWorkflowConfiguration()
    for (const step of Object.values(request.steps)) {
      expect(step.enabled).toBe(true)
      expect(step.engineConfig).toBeTypeOf("object")
    }
    expect(request.steps.screenshot).toMatchObject({ enabled: true })
    expect(request.steps.directory_scan).toMatchObject({ enabled: true })
    expect(request.steps.nuclei_vulnerability).toMatchObject({ enabled: true })
  })

  it("materializes an omitted Engine section default as disabled", () => {
    const profile = getMockScanWorkflowProfile("scanWorkflows/default")
    if (!profile) throw new Error("mock default Workflow Profile is unavailable")

    const profileSteps = profile.configuration.steps as Record<string, { engineConfig: Record<string, unknown> }>
    expect(profileSteps.subdomain_discovery.engineConfig.bruteforce).toEqual({ enabled: false })
  })

  it("materializes wildcard filtering as disabled in both active subdomain stages", () => {
    const profile = getMockScanWorkflowProfile("scanWorkflows/default")
    if (!profile) throw new Error("mock default Workflow Profile is unavailable")

    const profileSteps = profile.configuration.steps as Record<string, { engineConfig: Record<string, Record<string, unknown>> }>
    expect(profileSteps.subdomain_discovery.engineConfig.recon).toBeDefined()
    expect(profileSteps.subdomain_discovery.engineConfig.resolve).toMatchObject({
      enabled: true,
      "wildcard-filter": false,
    })
  })

  it("preserves opaque Directory matcher and delay strings in canonical Frontend serialization", () => {
    const workflow = getMockScanWorkflowByName("scanWorkflows/default")
    const profile = getMockScanWorkflowProfile("scanWorkflows/default")
    if (!workflow || !profile) throw new Error("mock default Workflow is unavailable")
    const details = workflow.steps.map((step) => {
      const detail = getMockEngineCatalogDetail(step.engineId)
      if (!detail) throw new Error(`mock Engine ${step.engineId} is unavailable`)
      return detail
    })
    const workflowWithEngines = buildWorkflowWithEngineCatalog(workflow, details, "en")
    const profileSteps = profile.configuration.steps as Record<string, Record<string, unknown>>
    const directoryConfig = profileSteps.directory_scan.engineConfig as Record<string, Record<string, unknown>>
    const matchCodes = "200, 201"
    const delay = "1 - 2"
    directoryConfig.ffuf["match-codes"] = matchCodes
    directoryConfig.ffuf.delay = delay
    const requestSteps: Record<string, { enabled: boolean; engineConfig?: Record<string, unknown> }> =
      Object.fromEntries(workflow.steps.map((step) => [step.stepId, { enabled: false }]))
    requestSteps.directory_scan = { enabled: true, engineConfig: directoryConfig }

    const serialized = serializeCanonicalWorkflowConfiguration(
      { steps: requestSteps },
      workflow,
      workflowWithEngines,
    )

    expect(serialized.steps.directory_scan).toEqual({
      enabled: true,
      engineConfig: expect.objectContaining({
        ffuf: expect.objectContaining({
          "match-codes": matchCodes,
          delay,
        }),
      }),
    })
  })
})
