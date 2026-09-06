#!/usr/bin/env node

import { runGuardrail } from "./ui-foundation-guardrail-lib.mjs"

const guardrailContract = {
  modes: ["inventory", "verify"],
  ledger: "foundation-exceptions.json",
  roots: ["components", "app"],
}
void guardrailContract

function isExcludedPath(file) {
  return (
    file.includes("__tests__") ||
    file.includes("prototype") ||
    file.includes("demo") ||
    file === "components/ui/button.tsx" ||
    file.includes("components/auth/terminal-login") ||
    file.includes("components/settings/system-logs/")
  )
}

runGuardrail({
  checkId: "button",
  title: "Button foundation check",
  rules: [
    {
      id: "native-button-with-classes",
      label: "native button with production classes",
      pattern: /<button\b(?=[^>]*\bclassName=)[^>]*>/g,
      exclude: isExcludedPath,
    },
    {
      id: "adhoc-button-size",
      label: "ad-hoc Button size classes",
      pattern: /<Button\b[^>\n]*className=(?:"[^"]*\b(?:h-|w-|size-|px-|py-)[^"]*"|\{[^}\n]*\})/g,
      exclude: isExcludedPath,
    },
  ],
})
