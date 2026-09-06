import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/agents/architecture-dialog-state.ts"), "utf8")
const architectureSources = [
  "components/settings/agents/architecture-dialog-state.ts",
  "components/settings/agents/architecture-dialog.tsx",
  "components/settings/agents/architecture-dialog-sections.tsx",
  "components/settings/agents/architecture-flow-state.ts",
].map((filePath) => readFileSync(path.resolve(process.cwd(), filePath), "utf8"))

function extractTranslationKeys(sourceText: string) {
  return Array.from(sourceText.matchAll(/\bt\("([^"]+)"\)/g), (match) => match[1])
}

function readDistributedMessages(locale: "en" | "zh") {
  const messages = JSON.parse(readFileSync(path.resolve(process.cwd(), `messages/${locale}.json`), "utf8"))
  return messages.pages.agents as Record<string, string>
}

describe("architecture-dialog-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useArchitectureDialogState")
    expect(source).toContain("from \"react\"")
  })

  it("keeps architecture dialog translation keys registered in zh and en messages", () => {
    const keys = Array.from(new Set(architectureSources.flatMap(extractTranslationKeys))).sort()
    const zhMessages = readDistributedMessages("zh")
    const enMessages = readDistributedMessages("en")

    expect(keys).toContain("flowEngineTitle")
    expect(keys.some((key) => key.includes("Worker"))).toBe(false)

    for (const key of keys) {
      expect(zhMessages, `zh missing pages.agents.${key}`).toHaveProperty(key)
      expect(enMessages, `en missing pages.agents.${key}`).toHaveProperty(key)
    }
  })

  it("keeps Engine result return copy available to architecture surfaces", () => {
    const zhMessages = readDistributedMessages("zh")
    const enMessages = readDistributedMessages("en")

    expect(zhMessages.flowDiagramNote).toContain("Engine → Agent → Server")
    expect(zhMessages.flowEngineComms).toContain("Agent")
    expect(zhMessages.flowEngineComms).not.toContain("Server")

    expect(enMessages.flowDiagramNote).toContain("Engine → Agent → Server")
    expect(enMessages.flowEngineComms).toContain("reports results to Agent")
    expect(enMessages.flowEngineComms).not.toContain("Server")
    expect(JSON.stringify(zhMessages)).not.toContain("Worker")
    expect(JSON.stringify(enMessages)).not.toContain("Worker")
  })
})
