#!/usr/bin/env node

import { existsSync, readFileSync } from "node:fs"
import { join } from "node:path"
import { fileURLToPath } from "node:url"

const root = join(fileURLToPath(new URL(".", import.meta.url)), "..")
const ledgerPath = join(root, "foundation-exceptions.json")

const routeGroups = [
  "overview",
  "scan/history",
  "scan/scheduled",
  "scan/workflow",
  "targets",
  "subdomains",
  "endpoints",
  "ip-addresses",
  "websites",
  "vulnerabilities",
  "fingerprints",
  "tools",
  "settings",
  "organizations",
  "search",
  "auth/login",
]

function loadLedger() {
  if (!existsSync(ledgerPath)) {
    console.error("missing foundation-exceptions.json")
    process.exit(1)
  }
  return JSON.parse(readFileSync(ledgerPath, "utf8"))
}

function groupBy(entries, getKey) {
  const grouped = new Map()
  for (const entry of entries) {
    const key = getKey(entry)
    grouped.set(key, (grouped.get(key) ?? 0) + 1)
  }
  return [...grouped.entries()].sort(([a], [b]) => a.localeCompare(b))
}

function routeGroupFor(entry) {
  const scope = String(entry.scope ?? "")
  for (const routeGroup of routeGroups) {
    const appRouteGroup = routeGroup === "auth/login" ? "login" : routeGroup
    if (
      scope.includes(`app/${appRouteGroup}`) ||
      scope.includes(`components/${routeGroup}`) ||
      scope.includes(`components/${routeGroup.split("/")[0]}`)
    ) {
      return routeGroup
    }
  }
  return "shared-or-cross-route"
}

function printGroup(title, rows) {
  console.log(`\n${title}`)
  if (rows.length === 0) {
    console.log("  none")
    return
  }
  for (const [key, count] of rows) {
    console.log(`  ${key}: ${count}`)
  }
}

const ledger = loadLedger()
const entries = Array.isArray(ledger.exceptions) ? ledger.exceptions : []

const groupByCheck = groupBy(entries, (entry) => String(entry.check ?? "all"))
const groupByStatus = groupBy(entries, (entry) => String(entry.status ?? "missing-status"))
const groupByOwner = groupBy(entries, (entry) => String(entry.owner ?? "missing-owner"))
const groupByRouteGroup = groupBy(entries, routeGroupFor)

const migrationDebt = entries.filter((entry) =>
  ["remove", "deferred"].includes(String(entry.status))
)

console.log("UI foundation ledger debt report")
console.log(`Ledger: foundation-exceptions.json`)
console.log(`Total entries: ${entries.length}`)
console.log(`Migration debt entries (remove/deferred): ${migrationDebt.length}`)

printGroup("By check", groupByCheck)
printGroup("By status", groupByStatus)
printGroup("By owner", groupByOwner)
printGroup("By route group", groupByRouteGroup)

if (migrationDebt.length > 0) {
  console.log("\nMigration debt")
  for (const entry of migrationDebt) {
    const check = entry.check ?? "all"
    const count = typeof entry.count === "number" ? ` count=${entry.count}` : ""
    console.log(
      `  ${entry.id} status=${entry.status} check=${check} route=${routeGroupFor(entry)}${count}`
    )
  }
}
