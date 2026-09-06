import { describe, expect, it, vi, beforeEach } from "vitest"

const apiMocks = vi.hoisted(() => ({
  post: vi.fn(),
}))

vi.mock("@/lib/api-client", () => ({
  api: apiMocks,
}))

import {
  initFormValuesFromWorkflow,
  serializeFormValuesToConfig,
} from "@/components/scan/engine-config-form"
import { api } from "@/lib/api-client"
import { bulkInitiateScan } from "@/services/scan.service"

import type { EngineConfigFormValues, ScanWorkflowWithEngines } from "@/types/engine-config.types"

const subdomainWorkflow: ScanWorkflowWithEngines = {
  scanWorkflowId: "default",
  displayName: "Default Scan",
  description: "Discover subdomains",
  stages: [
    {
      stageId: "discovery",
      steps: [
        {
          stepId: "subdomain_discovery",
          engineId: "engine.lunafox.subdomain_discovery",
          profileDefaultEnabled: true,
          engine: {
            manifestVersion: "engine.v5",
            engineId: "engine.lunafox.subdomain_discovery",
            displayName: "Subdomain Discovery",
            description: "Discover subdomains",
            publisher: "lunafox",
            execution: {
              engineApiMajor: 2,
              supportedTargetTypes: ["domain"],
              configSections: [
                {
                  id: "recon",
                  name: "Reconnaissance",
                  description: "Collect subdomains from passive sources",
                  defaultEnabled: true,
                  params: [
                    { key: "timeout", type: "integer", description: "Scan timeout in seconds", default: 3600, minimum: 1 },
                    { key: "threads", type: "integer", description: "Number of concurrent threads", default: 10, minimum: 1 },
                  ],
                },
                {
                  id: "bruteforce",
                  name: "Dictionary Bruteforce",
                  description: "Bruteforce domains using dictionaries",
                  params: [
                    { key: "timeout", type: "integer", description: "Scan timeout in seconds", default: 3600, minimum: 1 },
                    {
                      key: "wordlist",
                      type: "string",
                      description: "Subdomain wordlist name",
                      default: "subdomains-top1million-110000.txt",
                      minLength: 1,
                      resource: { kind: "wordlist" },
                    },
                    {
                      key: "resolvers",
                      type: "string",
                      description: "Resolver list for PureDNS Bruteforce",
                      default: "bruteforce-resolvers.txt",
                      minLength: 1,
                      resource: { kind: "wordlist" },
                    },
                    { key: "threads", type: "integer", description: "Number of concurrent threads", default: 100, minimum: 1 },
                    { key: "rate-limit", type: "integer", description: "Rate limit per second", default: 150, minimum: 1 },
                    { key: "wildcard-probe-count", type: "integer", description: "PureDNS wildcard tests per domain level", default: 50, minimum: 1 },
                    { key: "wildcard-batch", type: "integer", description: "PureDNS wildcard filtering batch size", default: 1000000, minimum: 1 },
                  ],
                },
                {
                  id: "resolve",
                  name: "Resolution",
                  description: "Resolve discovered subdomains",
                  defaultEnabled: true,
                  requiredEnabled: true,
                  params: [
                    { key: "timeout", type: "integer", description: "Scan timeout in seconds", default: 3600, minimum: 1 },
                    {
                      key: "resolvers",
                      type: "string",
                      description: "Resolver list for PureDNS Resolve",
                      default: "resolve-resolvers.txt",
                      minLength: 1,
                      resource: { kind: "wordlist" },
                    },
                  ],
                },
              ],
            },
          },
        },
      ],
    },
  ],
}

function toggleActiveDiscovery(values: EngineConfigFormValues): EngineConfigFormValues {
  return {
    ...values,
    subdomain_discovery: {
      ...values.subdomain_discovery,
      enabled: true,
      sections: {
        ...values.subdomain_discovery.sections,
        bruteforce: {
          ...values.subdomain_discovery.sections.bruteforce,
          enabled: true,
        },
        resolve: {
          ...values.subdomain_discovery.sections.resolve,
          enabled: true,
        },
      },
    },
  }
}

function profileConfigurationFor(workflow: ScanWorkflowWithEngines) {
  return {
    steps: Object.fromEntries(workflow.stages.flatMap((stage) => stage.steps).map((step) => [
      step.stepId,
      {
        enabled: false,
        engineConfig: Object.fromEntries(step.engine.execution.configSections.map((section) => [
          section.id,
          {
            enabled: section.defaultEnabled !== false,
            ...Object.fromEntries(section.params.map((param) => [
              param.key,
              param.default ?? (param.type === "integer" ? 1 : param.type === "boolean" ? false : ""),
            ])),
          },
        ])),
      },
    ])),
  }
}

describe("subdomain active discovery configuration flow", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("sends enabled bruteforce and final resolve when the scan form toggles active discovery on", async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: { count: 1, createdCount: 1, skipped: [], failed: [], scans: [] },
    } as never)

    const formValues = initFormValuesFromWorkflow(subdomainWorkflow, profileConfigurationFor(subdomainWorkflow))
    const configuration = serializeFormValuesToConfig(toggleActiveDiscovery(formValues))
    expect(formValues.subdomain_discovery).not.toHaveProperty("permutation")
    expect(
      (configuration.steps as Record<string, { engineConfig: Record<string, unknown> }>).subdomain_discovery?.engineConfig,
    ).not.toHaveProperty("permutation")

    await bulkInitiateScan({
      targetIds: [7],
      scanWorkflow: "default",
      inputSource: "scanSnapshot",
      configuration,
    })

    expect(api.post).toHaveBeenCalledWith("/scans:batchCreate", {
      requests: [{ target: "targets/7" }],
      scanWorkflow: "scanWorkflows/default",
      inputSource: "scanSnapshot",
      configuration: expect.objectContaining({
        steps: {
          subdomain_discovery: {
            enabled: true,
            engineConfig: expect.objectContaining({
              recon: expect.objectContaining({ enabled: true, threads: 10, timeout: 3600 }),
              bruteforce: expect.objectContaining({
                enabled: true,
                wordlist: "subdomains-top1million-110000.txt",
                resolvers: "bruteforce-resolvers.txt",
                "rate-limit": 150,
              }),
              resolve: expect.objectContaining({ enabled: true, resolvers: "resolve-resolvers.txt" }),
            }),
          },
        },
      }),
    })
  })
})
