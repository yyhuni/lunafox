import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/providers/mock-provider.tsx"), "utf8")

describe("mock-provider contract", () => {
  it("reads the mock switch from the side-effect-light config module", () => {
    expect(source).toContain('from "@/mock/config"')
    expect(source).not.toContain('from "@/mock"')
  })
})
