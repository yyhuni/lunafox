#!/usr/bin/env node

import { lstatSync, readdirSync, statSync } from "node:fs"
import { join, relative } from "node:path"
import { fileURLToPath } from "node:url"
import { readFileSync } from "node:fs"

const root = join(fileURLToPath(new URL(".", import.meta.url)), "..")
const modeFlagIndex = process.argv.indexOf("--mode")
const modeArg = process.argv.find((arg) => arg.startsWith("--mode="))
const mode =
  modeArg?.split("=")[1] ??
  (modeFlagIndex === -1 ? undefined : process.argv[modeFlagIndex + 1]) ??
  "inventory"

const foundationAllowlist = new Set([
  "README.md",
  "alert-dialog.tsx",
  "alert.tsx",
  "avatar.tsx",
  "badge.tsx",
  "button.tsx",
  "calendar.tsx",
  "card.tsx",
  "chart.tsx",
  "checkbox.tsx",
  "collapsible.tsx",
  "command.tsx",
  "dialog.tsx",
  "drawer.tsx",
  "dropdown-menu.tsx",
  "field.tsx",
  "form.tsx",
  "hover-card.tsx",
  "input.tsx",
  "label.tsx",
  "popover.tsx",
  "progress.tsx",
  "polymorphic.tsx",
  "radio-group.tsx",
  "resizable.tsx",
  "scroll-area.tsx",
  "select.tsx",
  "separator.tsx",
  "sheet.tsx",
  "sidebar.tsx",
  "skeleton.tsx",
  "sonner.tsx",
  "switch.tsx",
  "table.tsx",
  "tabs.tsx",
  "textarea.tsx",
  "toggle-group.tsx",
  "toggle.tsx",
  "tooltip.tsx",
  "number-stepper-input.tsx",
])

const allowedUiDirectories = new Set(["__tests__"])

const legacyFoundationSegment = "shad" + "cn-io"
const disallowedImportPattern = new RegExp(
  `@/components/ui/(dropzone|terminal-login|terminal-login-sections|terminal-login-types|yaml-editor|yaml-viewer|mermaid-diagram|data-table(?:/[^"')\\\\s]+)?|data-table-skeleton|confirm-dialog|confirm-dialog-sections|confirm-dialog-state|spinner|wave-grid|${legacyFoundationSegment}(?:/[^"')\\\\s]+)?|code-editor|codemirror-theme|datetime-picker|copyable-popover-content|card-grid-skeleton|master-detail-skeleton|terminal|overlay-styles)\\b`,
  "g"
)

const ignoredDirectories = new Set([
  ".git",
  ".next",
  ".factory",
  "coverage",
  "node_modules",
])

const scanExtensions = new Set([
  ".js",
  ".jsx",
  ".mjs",
  ".cjs",
  ".ts",
  ".tsx",
  ".json",
  ".md",
  ".sh",
])

function walk(dir, files = []) {
  for (const entry of readdirSync(dir)) {
    if (ignoredDirectories.has(entry)) continue
    const fullPath = join(dir, entry)
    const linkStats = lstatSync(fullPath)
    if (linkStats.isSymbolicLink()) {
      continue
    }
    const stats = statSync(fullPath)
    if (stats.isDirectory()) {
      walk(fullPath, files)
      continue
    }
    files.push(fullPath)
  }
  return files
}

function extensionOf(filePath) {
  const lastDot = filePath.lastIndexOf(".")
  return lastDot === -1 ? "" : filePath.slice(lastDot)
}

function findDisallowedUiFiles() {
  const uiRoot = join(root, "components", "ui")
  const violations = []

  for (const entry of readdirSync(uiRoot)) {
    const fullPath = join(uiRoot, entry)
    const stats = statSync(fullPath)
    if (stats.isDirectory()) {
      if (!allowedUiDirectories.has(entry)) {
        violations.push(relative(root, fullPath))
      }
      continue
    }
    if (!foundationAllowlist.has(entry)) {
      violations.push(relative(root, fullPath))
    }
  }

  return violations.sort()
}

function findDisallowedImports() {
  const violations = []
  for (const filePath of walk(root)) {
    if (!scanExtensions.has(extensionOf(filePath))) continue
    const content = readFileSync(filePath, "utf8")
    for (const match of content.matchAll(disallowedImportPattern)) {
      const line =
        content.slice(0, match.index).split("\n").length
      violations.push({
        file: relative(root, filePath),
        line,
        importPath: match[0],
      })
    }
  }
  return violations
}

function printSection(title, rows) {
  console.log(`\n${title}`)
  if (rows.length === 0) {
    console.log("  none")
    return
  }
  for (const row of rows) {
    if (typeof row === "string") {
      console.log(`  ${row}`)
    } else {
      console.log(`  ${row.file}:${row.line} ${row.importPath}`)
    }
  }
}

if (!["inventory", "verify"].includes(mode)) {
  console.error(`Unsupported mode: ${mode}`)
  process.exit(2)
}

const disallowedFiles = findDisallowedUiFiles()
const disallowedImports = findDisallowedImports()

console.log(`UI boundary check mode: ${mode}`)
printSection("Disallowed files under components/ui", disallowedFiles)
printSection("Disallowed legacy imports", disallowedImports)

if (mode === "verify" && (disallowedFiles.length > 0 || disallowedImports.length > 0)) {
  process.exit(1)
}
