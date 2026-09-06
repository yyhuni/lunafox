import { describe, expect, it } from "vitest"
import { readdirSync, readFileSync, statSync } from "node:fs"
import path from "node:path"

const hooksRoot = path.resolve(process.cwd(), "hooks")
const sharedMutationOwner = path.resolve(hooksRoot, "_shared/create-resource-mutation.ts")

function listHookFiles(dir: string): string[] {
  return readdirSync(dir)
    .flatMap((entry) => {
      const absolutePath = path.join(dir, entry)
      const stat = statSync(absolutePath)
      if (stat.isDirectory()) {
        return listHookFiles(absolutePath)
      }
      return /\.(ts|tsx)$/.test(entry) && !absolutePath.includes(`${path.sep}__tests__${path.sep}`)
        ? [absolutePath]
        : []
    })
}

function findMatchingCall(source: string, start: number) {
  const openIndex = source.indexOf("(", start)
  if (openIndex < 0) return null

  let depth = 0
  for (let index = openIndex; index < source.length; index += 1) {
    const char = source[index]
    if (char === "(" || char === "{" || char === "[") {
      depth += 1
    } else if (char === ")" || char === "}" || char === "]") {
      depth -= 1
      if (depth === 0) {
        return source.slice(start, index + 1)
      }
    }
  }

  return null
}

function findOwnerFunction(source: string, start: number) {
  const prefix = source.slice(0, start)
  const matches = [...prefix.matchAll(/(?:export\s+)?function\s+(\w+)/g)]
  return matches.at(-1)?.[1] ?? "unknown"
}

describe("remote mutation loading feedback contract", () => {
  it("keeps user-visible remote mutations on a stable loading toast lifecycle", () => {
    const violations: string[] = []

    for (const file of listHookFiles(hooksRoot)) {
      const source = readFileSync(file, "utf8")
      let cursor = 0

      while ((cursor = source.indexOf("useResourceMutation", cursor)) >= 0) {
        const owner = findOwnerFunction(source, cursor)
        const call = findMatchingCall(source, cursor)
        cursor += "useResourceMutation".length

        if (!call) continue
        if (!/toast\.(success|error|errorFromCode)|errorFallbackKey/.test(call)) continue
        if (call.includes("loadingToast")) continue
        violations.push(`${path.relative(process.cwd(), file)}:${owner}`)
      }
    }

    expect(violations).toEqual([])
  })

  it("keeps loading and dismissal ownership inside the shared mutation helper", () => {
    const offenders = listHookFiles(hooksRoot)
      .filter((file) => file !== sharedMutationOwner)
      .filter((file) => /toast\.(loading|dismiss)\(/.test(readFileSync(file, "utf8")))
      .map((file) => path.relative(process.cwd(), file))

    expect(offenders).toEqual([])
  })

  it("requires loading owners that skip default errors to own terminal feedback", () => {
    const violations: string[] = []

    for (const file of listHookFiles(hooksRoot)) {
      const source = readFileSync(file, "utf8")
      let cursor = 0

      while ((cursor = source.indexOf("useResourceMutation", cursor)) >= 0) {
        const owner = findOwnerFunction(source, cursor)
        const call = findMatchingCall(source, cursor)
        cursor += "useResourceMutation".length

        if (!call || !call.includes("loadingToast")) continue
        if (!call.includes("skipDefaultErrorHandler: true")) continue
        if (/onSuccess|onError|errorFallbackKey/.test(call)) continue

        violations.push(`${path.relative(process.cwd(), file)}:${owner}`)
      }
    }

    expect(violations).toEqual([])
  })
})
