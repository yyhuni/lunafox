import { describe, expect, it } from "vitest"

import {
  reconcileConfirmedWordlistCatalog,
  validateRequiredConfigResources,
} from "@/lib/engine-config-resource-validation"
import type {
  EngineConfigFormValues,
  ScanWorkflowWithEngines,
} from "@/types/engine-config.types"

describe("engine config resource validation", () => {
  it("reports every empty resource in enabled Steps and config sections", () => {
    const values = formValues()
    values.discovery.sections.recon.params.wordlist = ""
    values.discovery.sections.recon.params.exclude = []

    expect(validateRequiredConfigResources(workflow, values)).toEqual([
      {
        stepId: "discovery",
        sectionId: "recon",
        paramKey: "wordlist",
        key: '["discovery","recon","wordlist"]',
        fieldId: "engine-config-discovery-recon-wordlist",
      },
      {
        stepId: "discovery",
        sectionId: "recon",
        paramKey: "exclude",
        key: '["discovery","recon","exclude"]',
        fieldId: "engine-config-discovery-recon-exclude",
      },
    ])
  })

  it("ignores empty resources in disabled Steps and config sections", () => {
    const values = formValues()
    values.discovery.enabled = false
    values.discovery.sections.recon.params.wordlist = ""
    values.discovery.sections.optional.params.optional = ""
    expect(validateRequiredConfigResources(workflow, values)).toEqual([])

    values.discovery.enabled = true
    values.discovery.sections.recon.params.wordlist = "wordlists/1"
    values.discovery.sections.recon.params.exclude = "wordlists/2"
    expect(validateRequiredConfigResources(workflow, values)).toEqual([])
  })

  it("converts a unique fileName candidate only after a complete Catalog result", () => {
    const values = formValues()
    const unchanged = reconcileConfirmedWordlistCatalog(
      workflow,
      values,
      [
        { name: "wordlists/1", fileName: "dns.txt" },
        { name: "wordlists/2", fileName: "exclude.txt" },
        { name: "wordlists/3", fileName: "optional.txt" },
      ],
    )
    expect(unchanged).not.toBe(values)
    expect(unchanged.discovery.sections.recon.params).toMatchObject({
      wordlist: "wordlists/1",
      exclude: "wordlists/2",
    })

    const reconciled = reconcileConfirmedWordlistCatalog(
      workflow,
      values,
      [{ name: "wordlists/2", fileName: "exclude.txt" }],
    )
    expect(reconciled).not.toBe(values)
    expect(reconciled.discovery.sections.recon.params).toMatchObject({
      wordlist: "",
      exclude: "wordlists/2",
    })
    expect(reconciled.discovery.sections.optional.params.optional).toBe("")
    expect(values.discovery.sections.recon.params.wordlist).toBe("dns.txt")
  })

  it("keeps an unresolved candidate when duplicate fileName matches exist", () => {
    const values = formValues()
    const reconciled = reconcileConfirmedWordlistCatalog(workflow, values, [
      { name: "wordlists/1", fileName: "dns.txt" },
      { name: "wordlists/2", fileName: "dns.txt" },
      { name: "wordlists/3", fileName: "exclude.txt" },
    ])
    expect(reconciled.discovery.sections.recon.params.wordlist).toBe("dns.txt")
    expect(reconciled.discovery.sections.recon.params.exclude).toBe("wordlists/3")
  })
})

const workflow: ScanWorkflowWithEngines = {
  scanWorkflowId: "default",
  displayName: "Default",
  description: "",
  stages: [{
    stageId: "discovery",
    steps: [{
      stepId: "discovery",
      engineId: "engine.lunafox.discovery",
      profileDefaultEnabled: true,
      engine: {
        manifestVersion: "engine.v5",
        engineId: "engine.lunafox.discovery",
        publisher: "lunafox",
        displayName: "Discovery",
        description: "",
        execution: {
          engineApiMajor: 2,
          supportedTargetTypes: ["domain"],
          configSections: [
            {
              id: "recon",
              name: "Recon",
              defaultEnabled: true,
              params: [
                { key: "wordlist", type: "string", default: "dns.txt", resource: { kind: "wordlist" } },
                { key: "exclude", type: "string", default: "exclude.txt", resource: { kind: "wordlist" } },
              ],
            },
            {
              id: "optional",
              name: "Optional",
              defaultEnabled: false,
              params: [
                { key: "optional", type: "string", default: "optional.txt", resource: { kind: "wordlist" } },
              ],
            },
          ],
        },
      },
    }],
  }],
}

function formValues(): EngineConfigFormValues {
  return {
    discovery: {
      enabled: true,
      sections: {
        recon: {
          enabled: true,
          params: { wordlist: "dns.txt", exclude: "exclude.txt" },
        },
        optional: {
          enabled: false,
          params: { optional: "optional.txt" },
        },
      },
    },
  }
}
