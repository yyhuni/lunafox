import { describe, expect, it } from "vitest"
import {
  buildWorkflowEngineLibrary,
  buildWorkflowWithEngineCatalog,
  getLocalizedExecutedEngineDisplays,
  getLocalizedExecutedEngineNames,
  localizeEngineCatalogDetail,
} from "@/lib/engine-catalog"
import { getMockEngineCatalog, getMockEngineCatalogDetail } from "@/mock/data/engine-catalog"

describe("engine catalog projection", () => {
  it("builds workflow library entries for arbitrary installed engines", () => {
    const base = getMockEngineCatalog()[0]!
    const custom = {
      ...base,
      engineId: "engine.example.custom",
      localeResources: {
        zh: { engine: { displayName: "自定义引擎", description: "来自安装目录" } },
        en: { engine: { displayName: "Custom Engine", description: "Loaded from the installed catalog" } },
      },
    }

    expect(buildWorkflowEngineLibrary([custom], "zh")).toEqual([{
      engineId: "engine.example.custom",
      displayName: "自定义引擎",
      description: "来自安装目录",
    }])
    expect(buildWorkflowEngineLibrary([custom], "en")[0]?.displayName).toBe("Custom Engine")
  })

  it("resolves mock catalog data without changing saved identities", () => {
    const detail = getMockEngineCatalogDetail("engine.lunafox.website_discovery")!
    expect(localizeEngineCatalogDetail(detail, "zh").displayName).toBe("Website Discovery")
    expect(localizeEngineCatalogDetail(detail, "en").displayName).toBe("Website Discovery")
  })

  it("preserves required section metadata through catalog localization", () => {
    const detail = getMockEngineCatalogDetail("engine.lunafox.subdomain_discovery")!
    const localized = localizeEngineCatalogDetail(detail, "en")
    expect(localized.execution.configSections.find((section) => section.id === "resolve")).toMatchObject({
      defaultEnabled: true,
      requiredEnabled: true,
    })
  })

  it("keeps built-in Engine mock names and descriptions English across locales", () => {
    const expected = {
      "subdomain_discovery": { displayName: "Subdomain Discovery", description: "Discover target-domain subdomains through reconnaissance, optional dictionary bruteforce, and DNS resolution" },
      "port_scan": { displayName: "Port Scan", description: "Scan Target baselines plus finalized subdomain facts for open network ports and report host-port findings." },
      "website_discovery": { displayName: "Website Discovery", description: "Probe target-scoped URL lists for live websites and report website findings." },
      "url_collection": { displayName: "URL Collection", description: "Collect target-scoped URLs from historical sources and crawling, then report endpoint findings." },
      "screenshot": { displayName: "Screenshot", description: "Capture viewport screenshots for finalized website URLs." },
      "directory_scan": { displayName: "Directory Scan", description: "Discover target-scoped directory paths with FFUF and a selected wordlist." },
    }

    for (const [engineId, display] of Object.entries(expected)) {
      const detail = getMockEngineCatalogDetail(`engine.lunafox.${engineId}`)!
      expect(localizeEngineCatalogDetail(detail, "zh")).toMatchObject(display)
      expect(localizeEngineCatalogDetail(detail, "en")).toMatchObject(display)
    }
  })

  it("strictly joins workflow steps to catalog details", () => {
    const workflow = buildWorkflowWithEngineCatalog({
      name: "default",
      displayName: "Default",
      description: "",
      stages: [{ stageId: "stage", steps: [{ stageId: "stage", stepId: "website_discovery", engineId: "engine.lunafox.website_discovery", profileDefaultEnabled: true }] }],
      steps: [{ stageId: "stage", stepId: "website_discovery", engineId: "engine.lunafox.website_discovery", profileDefaultEnabled: true }],
      isBuiltin: true,
      isExecutable: true,
      etag: "etag",
      createTime: "",
      updateTime: "",
    }, [getMockEngineCatalogDetail("engine.lunafox.website_discovery")!], "zh")
    expect(workflow.stages[0]?.steps[0]?.engine.displayName).toBe("Website Discovery")
    expect(workflow.stages[0]?.steps[0]).not.toHaveProperty("engineConfig")
  })

  it("fails closed when a workflow engine is unavailable", () => {
    expect(() => buildWorkflowWithEngineCatalog({
      name: "default",
      displayName: "Default",
      description: "",
      stages: [{ stageId: "stage", steps: [{ stageId: "stage", stepId: "missing", engineId: "engine.lunafox.missing", profileDefaultEnabled: true }] }],
      steps: [{ stageId: "stage", stepId: "missing", engineId: "engine.lunafox.missing", profileDefaultEnabled: true }],
      isBuiltin: true,
      isExecutable: false,
      etag: "etag",
      createTime: "",
      updateTime: "",
    }, [], "zh")).toThrow("references unavailable engine engine.lunafox.missing")
  })

  it("resolves executed engine IDs into mock names without raw-ID fallback", () => {
    expect(getLocalizedExecutedEngineNames([
      "engine.lunafox.subdomain_discovery",
      "engine.lunafox.port_scan",
      "engine.lunafox.subdomain_discovery",
    ], getMockEngineCatalog(), "zh")).toEqual(["Subdomain Discovery", "Port Scan"])

    expect(getLocalizedExecutedEngineNames([
      "engine.lunafox.subdomain_discovery",
      "engine.lunafox.port_scan",
    ], getMockEngineCatalog(), "en")).toEqual(["Subdomain Discovery", "Port Scan"])
  })

  it("resolves executed engine IDs into English mock names and descriptions", () => {
    expect(getLocalizedExecutedEngineDisplays([
      "engine.lunafox.subdomain_discovery",
      "engine.lunafox.port_scan",
    ], getMockEngineCatalog(), "zh")).toEqual([
      {
        engineId: "engine.lunafox.subdomain_discovery",
        displayName: "Subdomain Discovery",
        description: "Discover target-domain subdomains through reconnaissance, optional dictionary bruteforce, and DNS resolution",
      },
      {
        engineId: "engine.lunafox.port_scan",
        displayName: "Port Scan",
        description: "Scan Target baselines plus finalized subdomain facts for open network ports and report host-port findings.",
      },
    ])
  })

  it("fails closed when an executed engine ID is absent from the catalog", () => {
    expect(() => getLocalizedExecutedEngineNames([
      "engine.lunafox.missing",
    ], getMockEngineCatalog(), "zh")).toThrow("Executed engine engine.lunafox.missing is unavailable")
  })
})
