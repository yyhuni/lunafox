#!/usr/bin/env node

import { lstatSync, readFileSync, readdirSync, statSync } from "node:fs"
import { extname, join, relative } from "node:path"
import { fileURLToPath } from "node:url"

const root = join(fileURLToPath(new URL(".", import.meta.url)), "..")
const modeFlagIndex = process.argv.indexOf("--mode")
const modeArg = process.argv.find((arg) => arg.startsWith("--mode="))
const mode =
  modeArg?.split("=")[1] ??
  (modeFlagIndex === -1 ? undefined : process.argv[modeFlagIndex + 1]) ??
  "inventory"

const ignoredDirectories = new Set([
  ".git",
  ".next",
  "coverage",
  "node_modules",
])

const scanExtensions = new Set([".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"])

const targetedScanRoots = [
  "components/ui/table.tsx",
  "components/ui/badge.tsx",
  "components/ui/tabs.tsx",
  "components/ui/card.tsx",
  "components/ui/field.tsx",
  "components/ui/dialog.tsx",
  "components/ui/sheet.tsx",
  "components/ui/alert-dialog.tsx",
  "components/ui/sidebar.tsx",
  "components/ui/command.tsx",
  "components/vulnerabilities/vulnerabilities-vertical-table.tsx",
  "components/vulnerabilities/vulnerability-vertical-detail.tsx",
  "components/search/search-page-sections.tsx",
  "components/search/search-pagination.tsx",
  "components/shared/data-table/pagination.tsx",
  "components/shared/data-table/expandable-cell.tsx",
  "components/scan/workflow-profile-selector.tsx",
  "components/scan/workflow/workflow-create-dialog-sections.tsx",
  "components/settings/agents/architecture-dialog-sections.tsx",
  "components/settings/agents/agent-list.tsx",
  "components/target/all-targets-columns.tsx",
  "components/organization/organization-list-sections.tsx",
  "components/endpoints/endpoints-columns.tsx",
  "components/fingerprints/fingerprinthub-fingerprint-columns.tsx",
  "components/target/target-overview-sections.tsx",
]

const fullProductionRoots = [
  "components",
  "app",
]

const allowedExceptionSubstrings = [
  "__tests__",
  "prototype",
  "prototypes",
  "demo",
  "terminal",
  "system-logs",
  "agent-log-drawer",
  "codemirror",
  "yaml-editor",
  "yaml-viewer",
  "chart",
  "github-star-button.tsx",
  "auth/terminal-login-sections.tsx",

  "components/ui/card.tsx",
]

const bannedPatterns = [
  {
    id: "foreground-opacity-body",
    label: "text-foreground/80",
    pattern: /text-foreground\/80/g,
    scope: "full",
  },
  {
    id: "table-header-wide-tracking",
    label: "tracking-[0.14em]",
    pattern: /tracking-\[0\.14em\]/g,
    scope: "full",
  },
  {
    id: "filter-wide-tracking",
    label: "tracking-[0.12em]",
    pattern: /tracking-\[0\.12em\]/g,
    scope: "full",
  },
  {
    id: "caps-wide-tracking",
    label: "tracking-[0.2em] uppercase",
    pattern: /tracking-\[0\.2em\][^"\n]*uppercase|uppercase[^"\n]*tracking-\[0\.2em\]/g,
    scope: "full",
  },
  {
    id: "micro-readable-text",
    label: "text-[9px]",
    pattern: /text-\[9px\]/g,
    scope: "full",
  },
  {
    id: "tiny-readable-text",
    label: "text-[10px]",
    pattern: /text-\[10px\]/g,
    scope: "targeted",
  },
  {
    id: "decorative-uppercase",
    label: "uppercase",
    pattern: /uppercase/g,
    scope: "targeted",
  },
  {
    id: "muted-small-body",
    label: "text-muted-foreground text-sm",
    pattern: /text-muted-foreground[^"\n]*text-sm|text-sm[^"\n]*text-muted-foreground/g,
    scope: "targeted",
  },
  {
    id: "muted-small-caption",
    label: "text-muted-foreground text-xs",
    pattern: /text-muted-foreground[^"\n]*text-xs|text-xs[^"\n]*text-muted-foreground/g,
    scope: "targeted",
  },
  {
    id: "medium-small-body",
    label: "font-medium text-sm",
    pattern: /font-medium[^"\n]*text-sm|text-sm[^"\n]*font-medium/g,
    scope: "targeted",
  },
  {
    id: "semibold-small-body",
    label: "font-semibold text-sm",
    pattern: /font-semibold[^"\n]*text-sm|text-sm[^"\n]*font-semibold/g,
    scope: "full",
  },
  {
    id: "medium-large-operational-metric",
    label: "font-medium text-lg",
    pattern: /font-medium[^"\n]*text-lg|text-lg[^"\n]*font-medium/g,
    scope: "full",
  },
]

function walk(dir, files = []) {
  for (const entry of readdirSync(dir)) {
    if (ignoredDirectories.has(entry)) continue
    const fullPath = join(dir, entry)
    const linkStats = lstatSync(fullPath)
    if (linkStats.isSymbolicLink()) continue
    const stats = statSync(fullPath)
    if (stats.isDirectory()) {
      walk(fullPath, files)
      continue
    }
    if (scanExtensions.has(extname(fullPath))) {
      files.push(fullPath)
    }
  }
  return files
}

function resolveScanFiles(scanRoots) {
  const files = []
  for (const scanRoot of scanRoots) {
    const fullPath = join(root, scanRoot)
    let stats
    try {
      stats = statSync(fullPath)
    } catch (error) {
      if (error?.code === "ENOENT") {
        continue
      }
      throw error
    }
    if (stats.isDirectory()) {
      walk(fullPath, files)
    } else {
      files.push(fullPath)
    }
  }
  return files.sort()
}

function isAllowedException(relativePath) {
  return allowedExceptionSubstrings.some((value) => relativePath.includes(value))
}

function findViolations() {
  const violations = []
  const targetedFiles = new Set(resolveScanFiles(targetedScanRoots))
  const fullFiles = resolveScanFiles(fullProductionRoots)

  for (const filePath of fullFiles) {
    const relativePath = relative(root, filePath)
    if (isAllowedException(relativePath)) {
      continue
    }

    const source = readFileSync(filePath, "utf8")
    for (const banned of bannedPatterns) {
      if (banned.scope === "targeted" && !targetedFiles.has(filePath)) {
        continue
      }
      for (const match of source.matchAll(banned.pattern)) {
        const line = source.slice(0, match.index).split("\n").length
        violations.push({
          file: relativePath,
          line,
          id: banned.id,
          label: banned.label,
          snippet: match[0],
        })
      }
    }
  }

  return violations
}

function printViolations(rows) {
  if (rows.length === 0) {
    console.log("  none")
    return
  }

  for (const row of rows) {
    console.log(`  ${row.file}:${row.line} [${row.id}] ${row.label} -> ${row.snippet}`)
  }
}

if (!["inventory", "verify"].includes(mode)) {
  console.error(`Unsupported mode: ${mode}`)
  process.exit(2)
}

const violations = findViolations()

console.log(`Typography foundation check mode: ${mode}`)
console.log("Full production roots:")
for (const scanRoot of fullProductionRoots) {
  console.log(`  ${scanRoot}`)
}
console.log("Targeted strict roots:")
for (const scanRoot of targetedScanRoots) {
  console.log(`  ${scanRoot}`)
}
console.log("Allowed exceptions:")
for (const value of allowedExceptionSubstrings) {
  console.log(`  ${value}`)
}
console.log("Violations:")
printViolations(violations)

if (mode === "verify" && violations.length > 0) {
  process.exit(1)
}
