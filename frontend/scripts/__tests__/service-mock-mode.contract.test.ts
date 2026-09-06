import { describe, expect, it } from "vitest"
import { mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs"
import { execFileSync } from "node:child_process"
import os from "node:os"
import path from "node:path"

const scriptSource = readFileSync(
  path.resolve(process.cwd(), "scripts/check-service-mock-mode.mjs"),
  "utf8"
)
const packageJson = JSON.parse(
  readFileSync(path.resolve(process.cwd(), "package.json"), "utf8")
) as { scripts: Record<string, string> }
const ledger = JSON.parse(
  readFileSync(path.resolve(process.cwd(), "mock/service-mock-exceptions.json"), "utf8")
) as {
  entries: Array<Record<string, string>>
}
const mockReadme = readFileSync(path.resolve(process.cwd(), "mock/README.md"), "utf8")
const servicesReadme = readFileSync(path.resolve(process.cwd(), "services/README.md"), "utf8")

describe("service mock mode guardrail", () => {
  it("is exposed as a package verification script", () => {
    expect(packageJson.scripts["check:service-mock-mode"]).toBe(
      "node scripts/check-service-mock-mode.mjs --mode verify"
    )
  })

  it("checks that services stay free of inline mock ownership and hooks do not bypass the data layer", () => {
    expect(scriptSource).toContain("service-mock-exceptions.json")
    expect(scriptSource).toContain("service-inline-mock-branch")
    expect(scriptSource).toContain("service-direct-mock-import")
    expect(scriptSource).toContain("hook-data-layer-bypass")
    expect(scriptSource).toContain("listFilesRecursive")
    expect(scriptSource).toContain("stale-exception")
  })

  it("fails when a hook bypasses the service mock boundary", () => {
    const tempRoot = mkdtempSync(path.join(os.tmpdir(), "service-mock-mode-"))
    const scriptsDir = path.join(tempRoot, "scripts")
    const mockDir = path.join(tempRoot, "mock")
    const servicesDir = path.join(tempRoot, "services", "nested")
    const hooksDir = path.join(tempRoot, "hooks")

    try {
      mkdirSync(scriptsDir, { recursive: true })
      mkdirSync(mockDir, { recursive: true })
      mkdirSync(servicesDir, { recursive: true })
      mkdirSync(hooksDir, { recursive: true })

      writeFileSync(
        path.join(mockDir, "service-mock-exceptions.json"),
        JSON.stringify({ entries: [] })
      )
      writeFileSync(
        path.join(servicesDir, "nested.service.ts"),
        'import { api } from "@/lib/api-client"\nimport { USE_MOCK } from "@/mock"\nexport const load = () => { if (USE_MOCK) return Promise.resolve([]); return api.get("/nested") }\n'
      )
      writeFileSync(
        path.join(hooksDir, "use-bad.ts"),
        'import { getMockCommands } from "@/mock/data/commands"\nexport function useBad(){ return getMockCommands() }\n'
      )

      const output = execFileSync(
        "node",
        [path.resolve(process.cwd(), "scripts/check-service-mock-mode.mjs"), "--mode", "inventory"],
        {
          cwd: tempRoot,
          env: {
            ...process.env,
            FRONTEND_ROOT: tempRoot,
          },
          encoding: "utf8",
        }
      )

      expect(output).toContain("services/nested/nested.service.ts [service-inline-mock-branch]")
      expect(output).toContain("hooks/use-bad.ts [hook-data-layer-bypass]")
    } finally {
      rmSync(tempRoot, { recursive: true, force: true })
    }
  })

  it("requires actionable metadata for every current exception when exceptions exist", () => {
    for (const entry of ledger.entries) {
      expect(entry.file).toMatch(/^(services\/.+\.service\.ts|hooks\/.+\.(ts|tsx))$/)
      expect(entry.owner).toBeTruthy()
      expect(entry.reason).toBeTruthy()
      expect(entry.status).toMatch(/^(needs-mock|mock-unsupported|external-runtime)$/)
      expect(entry.reviewTrigger).toBeTruthy()
      expect(entry.recoveryPath).toBeTruthy()
    }
  })

  it("documents network-layer mock ownership near the frontend code", () => {
    expect(mockReadme).toContain("network")
    expect(mockReadme).toContain("MSW")
    expect(servicesReadme).toContain("dev:mock")
    expect(servicesReadme).toContain("network-layer mock contract")
  })
})
