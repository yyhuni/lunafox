import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"
import {
  IconAutomation,
  IconBan,
  IconBug,
  IconBuilding,
  IconCircleX,
  IconPlayerStop,
  IconRadar,
  IconTarget,
  IconX,
  semanticIcons,
} from "@/components/icons"

const source = readFileSync(path.resolve(process.cwd(), "components/icons/semantic-icons.tsx"), "utf8")
const readme = readFileSync(path.resolve(process.cwd(), "components/icons/README.md"), "utf8")
const navigationSource = source.slice(source.indexOf("navigation:"), source.indexOf("action:"))
const metricSource = source.slice(source.indexOf("metric:"), source.indexOf("} as const"))
const auditedConcepts = [
  "organization",
  "target",
  "vulnerability",
  "scan",
  "scheduledScan",
  "workflow",
  "agent",
  "agentTopology",
  "automaticAssignment",
  "server",
  "tool",
  "engine",
  "database",
  "domain",
  "ip",
  "cidr",
  "website",
  "subdomain",
  "endpoint",
  "directory",
  "screenshot",
  "asset",
  "fingerprint",
  "wordlist",
  "nucleiRepository",
  "notification",
  "mcp",
  "apiKey",
  "blacklist",
  "systemLog",
] as const
const vulnerabilityIconConsumers = [
  "app/targets/[id]/layout.tsx",
  "app/scan/history/[id]/layout.tsx",
  "components/overview/overview-activity-tabs.tsx",
  "components/overview/overview-stat-cards.tsx",
  "components/scan/history/scan-runtime-detail-drawer.tsx",
  "components/scan/initiate-scan-dialog-sections.tsx",
  "components/target/target-overview-sections.tsx",
  "components/vulnerabilities/vulnerability-stat-cards.tsx",
  "lib/scan-workflow-config.ts",
]

describe("semantic-icons contract", () => {
  it("keeps one canonical category for stable product concepts", () => {
    expect(source).toContain("export const semanticIcons")
    expect(source).toContain("concept:")
    expect(source).toContain("navigation:")
    expect(source).toContain("action:")
    expect(source).toContain("status:")
    expect(source).toContain("metric:")
    expect(source).not.toContain("entity:")
  })

  it("binds core product concepts to one stable Tabler-backed glyph at runtime", () => {
    expect(semanticIcons.concept.organization).toBe(IconBuilding)
    expect(semanticIcons.concept.target).toBe(IconTarget)
    expect(semanticIcons.concept.vulnerability).toBe(IconBug)
    expect(semanticIcons.concept.scan).toBe(IconRadar)
    expect(semanticIcons.concept.scheduledScan).toBeDefined()
    expect(semanticIcons.concept.workflow).toBeDefined()
    expect(semanticIcons.concept.agent).toBeDefined()
    expect(semanticIcons.concept.agentTopology).toBeDefined()
    expect(semanticIcons.concept.automaticAssignment).toBe(IconAutomation)
    expect(semanticIcons.concept.engine).toBeDefined()
    expect(semanticIcons.concept).not.toHaveProperty("worker")
    expect(semanticIcons.concept.tool).toBeDefined()
    expect(semanticIcons.triggerSource.manual).toBeDefined()
    expect(semanticIcons.triggerSource.scheduled).toBeDefined()
    expect(semanticIcons.triggerSource.ai).toBeDefined()
    expect(source).toContain("manual: IconUser")
    expect(source).toContain("scheduled: IconCalendarClock")
    expect(source).toContain("ai: IconRobot")
    expect(semanticIcons.action.stop).toBe(IconPlayerStop)
    expect(semanticIcons.action.cancel).toBe(IconX)
    expect(semanticIcons.status.cancelled).toBe(IconBan)
    expect(semanticIcons.status.failed).toBe(IconCircleX)
    expect(source).toContain("stop: IconPlayerStop")
    expect(source).toContain("cancel: IconX")
    expect(source).toContain("cancelled: IconBan")
    expect(source).toContain("failed: IconCircleX")
    expect(semanticIcons.status.unknown).toBeDefined()
    expect(semanticIcons.metric.cpu).toBeDefined()
    expect(semanticIcons.metric.memory).toBeDefined()
    expect(semanticIcons.metric.disk).toBeDefined()
    expect(semanticIcons.metric.taskSlots).toBeDefined()
  })

  it("covers every audited stable product concept through the canonical registry", () => {
    expect(Object.keys(semanticIcons.concept)).toEqual(
      expect.arrayContaining([...auditedConcepts])
    )
  })

  it("keeps navigation and metric groups free of duplicate product concept bindings", () => {
    expect(navigationSource).toContain("overview:")
    expect(navigationSource).toContain("search:")
    expect(navigationSource).not.toContain("target:")
    expect(navigationSource).not.toContain("organization:")
    expect(navigationSource).not.toContain("vulnerability:")
    expect(navigationSource).not.toContain("scan:")
    expect(metricSource).not.toContain("totalTargets:")
    expect(metricSource).not.toContain("vulnerabilities:")
    expect(metricSource).not.toContain("scheduledScans:")
  })

  it("uses a heart icon only for the support navigation utility", () => {
    expect(source).toContain("IconHeart")
    expect(navigationSource).toContain("support: IconHeart")
    expect(navigationSource).not.toContain("support: IconShieldCheck")
  })

  it("routes vulnerability concept consumers through the canonical registry", () => {
    for (const filePath of vulnerabilityIconConsumers) {
      const consumer = readFileSync(path.resolve(process.cwd(), filePath), "utf8")
      expect(consumer, filePath).toContain("semanticIcons.concept.vulnerability")
      expect(consumer, filePath).not.toContain("ShieldAlert")
      expect(consumer, filePath).not.toContain("IconBug")
    }
  })

  it("documents concept bindings before raw icon selection", () => {
    expect(readme).toContain("Semantic Map First")
    expect(readme).toContain("semanticIcons.concept")
    expect(readme).toMatch(/same concept\s+entry/)
    expect(readme).toContain("Direct third-party icon imports remain limited to this folder")
  })
})
