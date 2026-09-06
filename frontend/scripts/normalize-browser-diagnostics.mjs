#!/usr/bin/env node

import fs from "node:fs/promises"
import path from "node:path"

const projectRoot = process.cwd()

function usage() {
  return [
    "用法: node scripts/normalize-browser-diagnostics.mjs [input.json] [output.json]",
    "",
    "环境变量:",
    "  DIAGNOSTICS_SOURCE_KIND   默认 chrome-devtools-mcp",
    "  DIAGNOSTICS_LABEL         诊断标签",
    "  DIAGNOSTICS_ROUTE         关联 route 或场景",
    "  DIAGNOSTICS_PHASE         before|after|supporting，默认 supporting",
  ].join("\n")
}

function summarizeValue(value, depth = 0) {
  if (value === null || value === undefined) {
    return value
  }
  if (typeof value === "string") {
    return value.length > 400 ? `${value.slice(0, 400)}...[truncated]` : value
  }
  if (typeof value === "number" || typeof value === "boolean") {
    return value
  }
  if (Array.isArray(value)) {
    if (depth >= 2) {
      return {
        type: "array",
        length: value.length,
      }
    }
    return value.slice(0, 10).map((entry) => summarizeValue(entry, depth + 1))
  }
  if (typeof value === "object") {
    const entries = Object.entries(value)
    if (depth >= 2) {
      return {
        type: "object",
        keys: entries.slice(0, 20).map(([key]) => key),
      }
    }
    return Object.fromEntries(
      entries.slice(0, 20).map(([key, entry]) => [key, summarizeValue(entry, depth + 1)]),
    )
  }
  return String(value)
}

async function main() {
  const [, , inputArg, outputArg] = process.argv
  if (!inputArg) {
    console.error(usage())
    process.exit(1)
  }

  const inputPath = path.resolve(projectRoot, inputArg)
  const outputPath = path.resolve(projectRoot, outputArg || path.join("test-plan", "supporting-diagnostics.json"))
  const sourceKind = process.env.DIAGNOSTICS_SOURCE_KIND || "chrome-devtools-mcp"
  const phase = process.env.DIAGNOSTICS_PHASE || "supporting"
  const label = process.env.DIAGNOSTICS_LABEL || path.basename(inputPath)
  const route = process.env.DIAGNOSTICS_ROUTE || null

  const rawText = await fs.readFile(inputPath, "utf8")
  const raw = JSON.parse(rawText)

  const normalized = {
    schemaVersion: 1,
    generatedAt: new Date().toISOString(),
    sourceKind,
    phase,
    label,
    route,
    inputRef: path.relative(projectRoot, inputPath).replace(/\\/g, "/"),
    summary: {
      topLevelKeys: raw && typeof raw === "object" && !Array.isArray(raw) ? Object.keys(raw).slice(0, 50) : [],
      preview: summarizeValue(raw),
    },
    canonicalForAcceptance: false,
  }

  await fs.mkdir(path.dirname(outputPath), { recursive: true })
  await fs.writeFile(outputPath, `${JSON.stringify(normalized, null, 2)}\n`, "utf8")
  console.log(`normalized diagnostics written: ${path.relative(projectRoot, outputPath).replace(/\\/g, "/")}`)
}

main().catch((error) => {
  console.error(error instanceof Error ? error.message : String(error))
  process.exit(1)
})
