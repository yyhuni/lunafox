import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const zh = JSON.parse(readFileSync(path.resolve(process.cwd(), "messages/zh.json"), "utf8"))
const en = JSON.parse(readFileSync(path.resolve(process.cwd(), "messages/en.json"), "utf8"))

describe("workflow naming contract", () => {
  it("uses workflow as the navigation and management page name", () => {
    expect(zh.navigation.scanWorkflow).toBe("工作流")
    expect(zh.scan.workflow.title).toBe("工作流")
    expect(zh.scan.workflow.create.title).toBe("新建工作流")
    expect(en.navigation.scanWorkflow).toBe("Workflow")
    expect(en.scan.workflow.title).toBe("Workflow")
    expect(en.scan.workflow.create.title).toBe("Create Workflow")
  })

  it("keeps scan workflow wording out of visible workflow route labels", () => {
    expect(zh.navigation.scanWorkflow).not.toContain("扫描")
    expect(zh.scan.workflow.title).not.toContain("扫描")
    expect(zh.scan.workflow.create.title).not.toContain("扫描")
    expect(en.navigation.scanWorkflow).not.toContain("Scan")
    expect(en.scan.workflow.title).not.toContain("Scan")
    expect(en.scan.workflow.create.title).not.toContain("Scan")
  })
})
