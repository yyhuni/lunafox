#!/usr/bin/env node

import { existsSync, readFileSync, readdirSync } from "node:fs"
import { join, relative, resolve } from "node:path"
import { fileURLToPath } from "node:url"
import { dirname } from "node:path"

const scriptDir = dirname(fileURLToPath(import.meta.url))
const frontendRoot = process.env.FRONTEND_ROOT
  ? resolve(process.env.FRONTEND_ROOT)
  : dirname(scriptDir)
const servicesDir = join(frontendRoot, "services")
const hooksDir = join(frontendRoot, "hooks")
const ledgerPath = join(frontendRoot, "mock", "service-mock-exceptions.json")

const modeFlagIndex = process.argv.indexOf("--mode")
const modeArg = process.argv.find((arg) => arg.startsWith("--mode="))
const mode =
  modeArg?.split("=")[1] ??
  (modeFlagIndex === -1 ? undefined : process.argv[modeFlagIndex + 1]) ??
  "verify"

if (!["inventory", "verify"].includes(mode)) {
  console.error(`Unsupported mode: ${mode}`)
  process.exit(2)
}

const requiredExceptionFields = [
  "file",
  "owner",
  "reason",
  "status",
  "reviewTrigger",
  "recoveryPath",
]

const allowedStatuses = new Set(["needs-mock", "mock-unsupported", "external-runtime"])

function toPosixPath(path) {
  return path.split("\\").join("/")
}

function readJson(path) {
  return JSON.parse(readFileSync(path, "utf8"))
}

function usesBackendClient(source) {
  return (
    /from\s+["']@\/lib\/api-client["']/.test(source) ||
    /from\s+["']axios["']/.test(source) ||
    /\bfetch\s*\(/.test(source)
  )
}

function usesMockMode(source) {
  return /\bUSE_MOCK\b/.test(source)
}

function usesDirectMockImport(source) {
  return /from\s+["']@\/mock(?:\/[^"']*)?["']/.test(source)
}

function listFilesRecursive(dir, predicate) {
  if (!existsSync(dir)) return []

  const entries = readdirSync(dir, { withFileTypes: true })
  return entries.flatMap((entry) => {
    const entryPath = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === "__tests__") return []
      return listFilesRecursive(entryPath, predicate)
    }
    return predicate(entryPath) ? [entryPath] : []
  })
}

function usesForbiddenHookDataAccess(source) {
  return (
    /from\s+["']@\/mock(?:\/[^"']*)?["']/.test(source) ||
    /from\s+["']@\/lib\/api-client["']/.test(source) ||
    /from\s+["']axios["']/.test(source)
  )
}

function validateException(entry) {
  const missing = requiredExceptionFields.filter((field) => !entry[field])
  if (missing.length > 0) {
    return `missing fields: ${missing.join(", ")}`
  }
  if (!allowedStatuses.has(entry.status)) {
    return `unsupported status: ${entry.status}`
  }
  return null
}

const ledger = readJson(ledgerPath)
const exceptionEntries = Array.isArray(ledger.entries) ? ledger.entries : []
const exceptionMap = new Map(exceptionEntries.map((entry) => [entry.file, entry]))
const findings = []

for (const entry of exceptionEntries) {
  const problem = validateException(entry)
  if (problem) {
    findings.push({
      file: entry.file ?? "(unknown)",
      type: "invalid-exception",
      detail: problem,
    })
  }
}

const serviceFiles = listFilesRecursive(
  servicesDir,
  (filePath) => filePath.endsWith(".service.ts")
)
  .sort()

for (const filePath of serviceFiles) {
  const source = readFileSync(filePath, "utf8")
  const servicePath = toPosixPath(relative(frontendRoot, filePath))
  const hasBackendClient = usesBackendClient(source)
  if (!hasBackendClient) continue

  const exception = exceptionMap.get(servicePath)
  const hasMockMode = usesMockMode(source)
  const hasDirectMockImport = usesDirectMockImport(source)

  if (hasDirectMockImport && !exception) {
    findings.push({
      file: servicePath,
      type: "service-direct-mock-import",
      detail: "production services must not import @/mock directly; mock ownership lives in the network-layer platform",
    })
  }

  if (hasMockMode && !exception) {
    findings.push({
      file: servicePath,
      type: "service-inline-mock-branch",
      detail: "production services must not keep inline USE_MOCK branching once the network-layer mock platform is active",
    })
  }

  if ((hasMockMode || hasDirectMockImport) && exception) {
    findings.push({
      file: servicePath,
      type: "stale-exception",
      detail: "remove the temporary exception after deleting the inline service mock ownership",
    })
  }
}

const hookFiles = listFilesRecursive(
  hooksDir,
  (filePath) => filePath.endsWith(".ts") || filePath.endsWith(".tsx")
).sort()

for (const filePath of hookFiles) {
  const source = readFileSync(filePath, "utf8")
  if (!usesForbiddenHookDataAccess(source)) continue

  const hookPath = toPosixPath(relative(frontendRoot, filePath))
  const exception = exceptionMap.get(hookPath)
  if (!exception) {
    findings.push({
      file: hookPath,
      type: "hook-data-layer-bypass",
      detail: "hook imports mock data or backend clients directly; route data must go through services",
    })
  }
}

for (const entry of exceptionEntries) {
  if (!entry.file) continue
  const absolutePath = join(frontendRoot, entry.file)
  if (!existsSync(absolutePath)) {
    findings.push({
      file: entry.file,
      type: "missing-exception-target",
      detail: "ledger entry points to a missing file",
    })
  }
}

console.log(`Service mock mode check mode: ${mode}`)
console.log("Ledger: mock/service-mock-exceptions.json")
console.log("Findings:")

if (findings.length === 0) {
  console.log("  none")
} else {
  for (const finding of findings) {
    console.log(`  ${finding.file} [${finding.type}] ${finding.detail}`)
  }
}

if (mode === "verify" && findings.length > 0) {
  process.exit(1)
}
