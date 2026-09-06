import { existsSync, lstatSync, readFileSync, readdirSync, statSync } from "node:fs"
import { extname, join, relative } from "node:path"
import { fileURLToPath } from "node:url"

export const root = join(fileURLToPath(new URL(".", import.meta.url)), "..")
export const ledgerPath = join(root, "foundation-exceptions.json")

const ignoredDirectories = new Set([".git", ".next", "coverage", "node_modules"])
const scanExtensions = new Set([".css", ".ts", ".tsx"])
const approvedStatuses = new Set(["keep", "narrow", "remove", "deferred", "excluded"])

export function parseMode() {
  const modeFlagIndex = process.argv.indexOf("--mode")
  const modeArg = process.argv.find((arg) => arg.startsWith("--mode="))
  const mode =
    modeArg?.split("=")[1] ??
    (modeFlagIndex === -1 ? undefined : process.argv[modeFlagIndex + 1]) ??
    "inventory"

  if (!["inventory", "verify"].includes(mode)) {
    console.error(`Unsupported mode: ${mode}`)
    process.exit(2)
  }

  return mode
}

export function loadLedger() {
  if (!existsSync(ledgerPath)) {
    return { exceptions: [] }
  }
  return JSON.parse(readFileSync(ledgerPath, "utf8"))
}

export function walk(dir, files = []) {
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

export function resolveProductionFiles(scanRoots = ["components", "app"]) {
  return scanRoots.flatMap((scanRoot) => walk(join(root, scanRoot))).sort()
}

export function toRelativePath(filePath) {
  return relative(root, filePath)
}

export function lineForIndex(source, index) {
  return source.slice(0, index).split("\n").length
}

function globToRegex(glob) {
  const escaped = glob
    .replace(/[.+^${}()|[\]\\]/g, "\\$&")
    .replace(/\*\*/g, ".*")
    .replace(/\*/g, "[^/]*")
  return new RegExp(`^${escaped}$`)
}

function matchesScope(scope, file) {
  if (scope === file) return true
  if (scope.includes("*")) return globToRegex(scope).test(file)
  return file.startsWith(scope)
}

function matchesPattern(pattern, finding) {
  if (!pattern || pattern === "*") return true
  return (
    finding.snippet.includes(pattern) ||
    finding.ruleId.includes(pattern) ||
    finding.label.includes(pattern)
  )
}

export function findApprovedException(finding, ledger, usage) {
  for (const entry of ledger.exceptions ?? []) {
    if (!approvedStatuses.has(entry.status)) continue
    if (entry.check && entry.check !== finding.checkId && entry.check !== "all") continue
    if (entry.rule && entry.rule !== finding.ruleId) continue
    if (!matchesScope(entry.scope, finding.file)) continue
    if (!matchesPattern(entry.pattern, finding)) continue

    if (typeof entry.count === "number") {
      const used = usage.get(entry.id) ?? 0
      if (used >= entry.count) continue
      usage.set(entry.id, used + 1)
    }

    return entry
  }

  return undefined
}

export function applyLedger(findings, ledger) {
  const usage = new Map()
  return findings.map((finding) => ({
    ...finding,
    exception: findApprovedException(finding, ledger, usage)?.id,
  }))
}

export function printFindings(title, findings) {
  console.log(`\n${title}`)
  if (findings.length === 0) {
    console.log("  none")
    return
  }

  for (const finding of findings) {
    const suffix = finding.exception ? ` exception=${finding.exception}` : ""
    console.log(
      `  ${finding.file}:${finding.line} [${finding.ruleId}] ${finding.label} -> ${finding.snippet}${suffix}`
    )
  }
}

export function summarizeFindings(findings) {
  const summary = new Map()
  for (const finding of findings) {
    const key = `${finding.file}|${finding.ruleId}|${finding.label}`
    const current = summary.get(key) ?? {
      file: finding.file,
      ruleId: finding.ruleId,
      label: finding.label,
      count: 0,
      approved: 0,
    }
    current.count += 1
    if (finding.exception) current.approved += 1
    summary.set(key, current)
  }
  return [...summary.values()].sort((a, b) =>
    `${a.file}|${a.ruleId}`.localeCompare(`${b.file}|${b.ruleId}`)
  )
}

export function printSummary(findings) {
  console.log("\nSummary:")
  const rows = summarizeFindings(findings)
  if (rows.length === 0) {
    console.log("  none")
    return
  }
  for (const row of rows) {
    console.log(
      `  ${row.file} [${row.ruleId}] count=${row.count} approved=${row.approved}`
    )
  }
}

export function runGuardrail({ checkId, title, rules, scanRoots = ["components", "app"] }) {
  const mode = parseMode()
  const ledger = loadLedger()
  const findings = []

  for (const filePath of resolveProductionFiles(scanRoots)) {
    const file = toRelativePath(filePath)
    const source = readFileSync(filePath, "utf8")

    for (const rule of rules) {
      if (rule.exclude?.(file, source)) continue
      for (const match of source.matchAll(rule.pattern)) {
        const snippet = match[0].replace(/\s+/g, " ").slice(0, 140)
        if (rule.ignoreMatch?.({ file, source, match, snippet })) continue
        findings.push({
          checkId,
          file,
          line: lineForIndex(source, match.index ?? 0),
          ruleId: rule.id,
          label: rule.label,
          snippet,
        })
      }
    }
  }

  const checked = applyLedger(findings, ledger)
  const unapproved = checked.filter((finding) => !finding.exception)

  console.log(`${title} mode: ${mode}`)
  console.log(`Ledger: foundation-exceptions.json`)
  console.log("Production roots:")
  for (const scanRoot of scanRoots) {
    console.log(`  ${scanRoot}`)
  }
  printFindings("Findings:", checked)
  printSummary(checked)

  if (mode === "verify" && unapproved.length > 0) {
    process.exit(1)
  }
}
