import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/login/page.tsx"), "utf8")

describe("page contract", () => {
  it("renders the login route without a generic lazy-page skeleton wrapper", () => {
    expect(source).toContain("export default function LoginPage")
    expect(source).toContain("import LoginPageContent from \"./content\"")
    expect(source).not.toContain("from \"@/components/common/lazy-page\"")
  })
})
