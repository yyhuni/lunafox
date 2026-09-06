import { describe, expect, it } from "vitest"
import { existsSync, readFileSync } from "node:fs"
import path from "node:path"

function readSource(relativePath: string) {
  const absolutePath = path.resolve(process.cwd(), relativePath)
  expect(existsSync(absolutePath), `${relativePath} should exist`).toBe(true)
  return readFileSync(absolutePath, "utf8")
}

describe("app router error surfaces contract", () => {
  it("owns framework not-found, locale error, and global error surfaces", () => {
    const notFoundSource = readSource("app/not-found.tsx")
    const localeErrorSource = readSource("app/error.tsx")
    const globalErrorSource = readSource("app/global-error.tsx")

    expect(notFoundSource).toContain("export default function NotFound")
    expect(notFoundSource).toContain("notFound")

    expect(localeErrorSource).toContain('"use client"')
    expect(localeErrorSource).toContain("export default function LocaleError")
    expect(localeErrorSource).toContain("reset")

    expect(globalErrorSource).toContain('"use client"')
    expect(globalErrorSource).toContain("export default function GlobalError")
    expect(globalErrorSource).toContain("<html")
    expect(globalErrorSource).toContain("<body")
  })
})
