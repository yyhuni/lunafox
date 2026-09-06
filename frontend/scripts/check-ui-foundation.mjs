#!/usr/bin/env node

import { spawnSync } from "node:child_process"
import { fileURLToPath } from "node:url"
import { dirname, join } from "node:path"

const scriptDir = dirname(fileURLToPath(import.meta.url))
const guardrailContract = {
  modes: ["inventory", "verify"],
  ledger: "foundation-exceptions.json",
  roots: ["app", "components", "hooks", "lib"],
}
void guardrailContract

const modeFlagIndex = process.argv.indexOf("--mode")
const modeArg = process.argv.find((arg) => arg.startsWith("--mode="))
const mode =
  modeArg?.split("=")[1] ??
  (modeFlagIndex === -1 ? undefined : process.argv[modeFlagIndex + 1]) ??
  "inventory"

const scripts = [
  "check-typography-foundation.mjs",
  "check-color-foundation.mjs",
  "check-spacing-foundation.mjs",
  "check-button-foundation.mjs",
  "check-component-foundation.mjs",
  "check-a11y-foundation.mjs",
  "check-motion-foundation.mjs",
  "check-loading-foundation.mjs",
  "check-ui-boundary.mjs",
]

if (!["inventory", "verify"].includes(mode)) {
  console.error(`Unsupported mode: ${mode}`)
  process.exit(2)
}

console.log(`UI foundation aggregate mode: ${mode}`)
console.log("Ledger: foundation-exceptions.json")
console.log("Production roots:")
console.log("  app")
console.log("  components")
console.log("  hooks")
console.log("  lib")

let failed = false
for (const scriptName of scripts) {
  const result = spawnSync(process.execPath, [join(scriptDir, scriptName), "--mode", mode], {
    stdio: "inherit",
  })
  if (result.status !== 0) {
    failed = true
  }
}

if (failed) {
  process.exit(1)
}
