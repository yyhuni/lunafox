#!/usr/bin/env node

import fs from "node:fs/promises"
import path from "node:path"

const projectRoot = process.cwd()
const appLocaleDir = path.join(projectRoot, "app", "[locale]")
const outputPath = path.join(projectRoot, "test-plan", "routes.todo.json")

const excludedPrefixes = ["/[locale]/tools", "/[locale]/prototypes"]
const excludedPattern = /\/demo(\/|$)/i

const p0Routes = new Set([
  "/[locale]/",
  "/[locale]/login",
  "/[locale]/dashboard",
  "/[locale]/scan/history",
  "/[locale]/scan/workflow",
  "/[locale]/target",
  "/[locale]/organization",
  "/[locale]/vulnerabilities",
  "/[locale]/search",
])

async function collectPageFiles(dir) {
  const entries = await fs.readdir(dir, { withFileTypes: true })
  const files = []

  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      files.push(...(await collectPageFiles(fullPath)))
      continue
    }
    if (entry.isFile() && entry.name === "page.tsx") {
      files.push(fullPath)
    }
  }

  return files
}

function toRoutePattern(filePath) {
  const relative = path.relative(appLocaleDir, filePath)
  const withoutFile = relative
    .replace(/^page\.tsx$/, "")
    .replace(/[\\/]page\.tsx$/, "")
  if (!withoutFile) return "/[locale]/"
  return `/[locale]/${withoutFile.split(path.sep).join("/")}`
}

function getExcludeReason(routePattern) {
  for (const prefix of excludedPrefixes) {
    if (routePattern === prefix || routePattern.startsWith(`${prefix}/`)) {
      return prefix
    }
  }
  if (excludedPattern.test(routePattern)) {
    return "/demo/"
  }
  return null
}

function getPriority(routePattern) {
  if (p0Routes.has(routePattern)) return "P0"
  if (routePattern.includes("[") && routePattern.includes("]")) return "P1"
  if (
    routePattern.startsWith("/[locale]/scan/") ||
    routePattern.startsWith("/[locale]/target/") ||
    routePattern.startsWith("/[locale]/organization/") ||
    routePattern.startsWith("/[locale]/settings/")
  ) {
    return "P1"
  }
  return "P2"
}

function toRouteId(routePattern) {
  const base = routePattern
    .replace(/^\/\[locale\]\//, "")
    .replace(/^\/\[locale\]\/?$/, "root")
    .replace(/[^a-zA-Z0-9]+/g, "_")
    .replace(/^_+|_+$/g, "")
    .toLowerCase()
  return `route_${base || "root"}`
}

async function main() {
  const pageFiles = await collectPageFiles(appLocaleDir)
  const routePatterns = pageFiles.map(toRoutePattern).sort((a, b) => a.localeCompare(b))

  const routes = []
  let skipped = 0

  for (const routePattern of routePatterns) {
    const reason = getExcludeReason(routePattern)
    if (reason) {
      skipped += 1
      continue
    }

    routes.push({
      id: toRouteId(routePattern),
      routePattern,
      locales: ["zh", "en"],
      priority: getPriority(routePattern),
      status: "pending",
      attempts: 0,
      lastRunAt: null,
      lastResult: null,
      notes: "",
    })
  }

  const payload = {
    schemaVersion: 1,
    generatedAt: new Date().toISOString(),
    source: "scripts/build-route-todo.mjs",
    include: "app/[locale]/**/page.tsx",
    exclusions: [
      "route starts with /[locale]/tools",
      "route starts with /[locale]/prototypes",
      "route path includes /demo/",
    ],
    summary: {
      discovered: routePatterns.length,
      skipped,
      planned: routes.length,
    },
    routes,
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
