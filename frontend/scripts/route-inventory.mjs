#!/usr/bin/env node

import { readdirSync } from "node:fs"
import path from "node:path"
import { fileURLToPath } from "node:url"

const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const defaultAppDir = path.resolve(scriptDir, "../app")

export const excludedPrefixes = Object.freeze(["/prototypes"])
export const excludedPattern = /\/demo(\/|$)/i
export const serverRedirectAliases = new Map([
  [
    "/targets/[id]/relations",
    {
      redirectTarget: "/targets/[id]/websites/",
    },
  ],
  [
    "/targets/[id]/relations/[websiteId]/[[...section]]",
    {
      redirectTarget: "/targets/[id]/websites/[websiteId]/[[...section]]/",
    },
  ],
  [
    "/scan/workflow",
    {
      redirectTarget: "/scan/config/workflows/",
    },
  ],
  [
    "/tools/engines",
    {
      redirectTarget: "/scan/config/engines/",
    },
  ],
])

const p0Routes = new Set([
  "/",
  "/login",
  "/overview",
  "/scan/history",
  "/scan/config/workflows",
  "/targets",
  "/organizations",
  "/vulnerabilities",
  "/search",
])

function normalizePath(value) {
  return value.split(path.sep).join("/")
}

function collectPageFiles(directory) {
  const files = []

  for (const entry of readdirSync(directory, { withFileTypes: true }).sort((left, right) => (
    left.name.localeCompare(right.name)
  ))) {
    const fullPath = path.join(directory, entry.name)
    if (entry.isDirectory()) {
      files.push(...collectPageFiles(fullPath))
      continue
    }
    if (entry.isFile() && entry.name === "page.tsx") {
      files.push(fullPath)
    }
  }

  return files
}

function toRoutePattern(appDir, filePath) {
  const relative = normalizePath(path.relative(appDir, filePath))
  if (relative === "page.tsx") return "/"
  return `/${relative.replace(/\/page\.tsx$/, "")}`
}

function toSourcePath(frontendRoot, filePath) {
  return normalizePath(path.relative(frontendRoot, filePath))
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

function getSkipDetails(routePattern) {
  const redirectAlias = serverRedirectAliases.get(routePattern)
  if (redirectAlias) {
    return {
      skipReason: "server-redirect-alias",
      redirectTarget: redirectAlias.redirectTarget,
      notes: `Server redirect alias; smoke the canonical route ${redirectAlias.redirectTarget} instead.`,
    }
  }

  const exclusion = getExcludeReason(routePattern)
  if (exclusion) {
    return {
      skipReason: "non-production-route",
      exclusion,
      notes: `Excluded non-production route (${exclusion}).`,
    }
  }

  return null
}

function getPriority(routePattern) {
  if (p0Routes.has(routePattern)) return "P0"
  if (routePattern.includes("[") && routePattern.includes("]")) return "P1"
  if (
    routePattern.startsWith("/scan/") ||
    routePattern.startsWith("/targets/") ||
    routePattern.startsWith("/organizations/") ||
    routePattern.startsWith("/settings/")
  ) {
    return "P1"
  }
  return "P2"
}

function toRouteId(routePattern) {
  const base = routePattern
    .replace(/^\//, "")
    .replace(/^$/, "root")
    .replace(/[^a-zA-Z0-9]+/g, "_")
    .replace(/^_+|_+$/g, "")
    .toLowerCase()
  return `route_${base || "root"}`
}

/**
 * Derives route coverage from public Next page source. Runtime smoke status is
 * intentionally not part of this result; that state belongs to private
 * `test-plan/routes.todo.json` and must not be required by public validation.
 */
export function collectRouteInventory(appDir = defaultAppDir) {
  const resolvedAppDir = path.resolve(appDir)
  const frontendRoot = path.dirname(resolvedAppDir)
  const pageFiles = collectPageFiles(resolvedAppDir)
  const entries = pageFiles
    .map((filePath) => ({
      filePath,
      routePattern: toRoutePattern(resolvedAppDir, filePath),
      sourcePath: toSourcePath(frontendRoot, filePath),
    }))
    .sort((left, right) => left.routePattern.localeCompare(right.routePattern))

  const routes = []
  const skippedRoutes = []

  for (const { routePattern, sourcePath } of entries) {
    const base = {
      id: toRouteId(routePattern),
      routePattern,
      sourcePath,
    }
    const skipDetails = getSkipDetails(routePattern)
    if (skipDetails) {
      skippedRoutes.push({
        ...base,
        status: "skipped",
        ...skipDetails,
      })
      continue
    }

    routes.push({
      ...base,
      locales: ["zh", "en"],
      priority: getPriority(routePattern),
      status: "pending",
      attempts: 0,
      lastRunAt: null,
      lastResult: null,
      notes: "",
    })
  }

  return {
    schemaVersion: 2,
    include: "app/**/page.tsx",
    exclusions: [
      "route starts with /prototypes",
      "route path includes /demo/",
      "explicit server redirect aliases",
    ],
    summary: {
      discovered: entries.length,
      skipped: skippedRoutes.length,
      planned: routes.length,
    },
    routes,
    skippedRoutes,
  }
}

export function collectActiveRouteInventory(appDir = defaultAppDir) {
  return collectRouteInventory(appDir).routes
}
