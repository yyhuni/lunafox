#!/usr/bin/env node

import fs from "node:fs/promises"
import path from "node:path"

function parseArgs(argv) {
  const options = {
    output: path.join(process.cwd(), "reports", "refactor-visual-baseline.json"),
    routes: [],
  }

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index]
    if (arg === "--") {
      continue
    }
    if (arg === "--output") {
      const next = argv[index + 1]
      if (!next) {
        throw new Error("--output 需要一个值")
      }
      options.output = path.resolve(process.cwd(), next)
      index += 1
      continue
    }
    if (arg === "--route") {
      const next = argv[index + 1]
      if (!next) {
        throw new Error("--route 需要一个值")
      }
      options.routes.push(next)
      index += 1
      continue
    }
    throw new Error(`未知参数: ${arg}`)
  }

  return options
}

async function ensureDir(targetPath) {
  await fs.mkdir(targetPath, { recursive: true })
}

async function main() {
  const options = parseArgs(process.argv.slice(2))
  const routes = options.routes.length > 0
    ? options.routes
    : [
      "/login",
      "/overview",
      "/data-table",
      "/detail",
      "/dialog",
      "/theme",
    ]

  const payload = {
    generated_at: new Date().toISOString(),
    mode: "mock",
    routes,
    source_script: "frontend/scripts/build-refactor-visual-baseline.mjs",
  }

  await ensureDir(path.dirname(options.output))
  await fs.writeFile(options.output, JSON.stringify(payload, null, 2) + "\n", "utf8")
  process.stdout.write(`${options.output}\n`)
}

main().catch((error) => {
  console.error(error instanceof Error ? error.message : String(error))
  process.exitCode = 1
})
