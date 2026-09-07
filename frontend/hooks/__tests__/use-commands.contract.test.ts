import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-commands.ts"), "utf8")

describe("use-commands contract", () => {
  it("keeps command hooks behind the service layer", () => {
    expect(source).toContain("export function useCommands")
    expect(source).toContain("from \"@tanstack/react-query\"")
    expect(source).toContain("CommandService.getCommands")
    expect(source).toContain("CommandService.getCommandById")
    expect(source).toContain("CommandService.batchDeleteCommands")
    expect(source).not.toContain("@/mock")
    expect(source).not.toContain("@/mock/data/commands")
  })
})
