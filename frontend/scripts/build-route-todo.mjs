#!/usr/bin/env node

import fs from "node:fs/promises"
import path from "node:path"
import { collectRouteInventory } from "./route-inventory.mjs"

const projectRoot = process.cwd()
const appDir = path.join(projectRoot, "app")
const outputPath = path.join(projectRoot, "test-plan", "routes.todo.json")

async function main() {
  const inventory = collectRouteInventory(appDir)
  const stripSourcePath = (route) => {
    const publicRoute = { ...route }
    delete publicRoute.sourcePath
    return publicRoute
  }
  const routes = inventory.routes.map(stripSourcePath)
  const skippedRoutes = inventory.skippedRoutes.map(stripSourcePath)

  const payload = {
    schemaVersion: 2,
    generatedAt: new Date().toISOString(),
    source: "scripts/build-route-todo.mjs",
    include: "app/**/page.tsx",
    exclusions: [
      "route starts with /prototypes",
      "route path includes /demo/",
      "explicit server redirect aliases",
    ],
    summary: inventory.summary,
    routes,
    skippedRoutes,
  }

  await fs.mkdir(path.dirname(outputPath), { recursive: true })
  await fs.writeFile(outputPath, `${JSON.stringify(payload, null, 2)}\n`, "utf8")

  console.log(`generated ${outputPath}`)
  console.log(`discovered=${payload.summary.discovered} skipped=${payload.summary.skipped} planned=${payload.summary.planned}`)
}

main().catch((error) => {
  console.error(error)
  process.exit(1)
})
