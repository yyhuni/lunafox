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
    file.includes("components/ui/") ||
    file.includes("components/auth/terminal-login")
  )
}

runGuardrail({
  checkId: "a11y",
  title: "Accessibility foundation check",
  rules: [
    {
      id: "icon-only-button-without-name",
      label: "icon-only button without accessible name",
      pattern: /<Button\b(?=[^>\n]*size=["']icon(?:-sm|-lg)?["'])(?![^>\n]*(?:aria-label|aria-labelledby|title=))[^>\n]*>/g,
      exclude: isExcludedPath,
    },
    {
      id: "focus-outline-removal",
      label: "focus outline removal without visible replacement",
      pattern: /(?:focus|focus-visible):outline-(?:none|hidden)/g,
      exclude: isExcludedPath,
      ignoreMatch: ({ source, match }) => {
        const lineStart = source.lastIndexOf("\n", match.index ?? 0) + 1
        const lineEnd = source.indexOf("\n", match.index ?? 0)
        const line = source.slice(lineStart, lineEnd === -1 ? source.length : lineEnd)
        return /focus(?:-visible)?:ring|focus(?:-visible)?:border/.test(line)
      },
    },
    {
      id: "clickable-non-button",
      label: "clickable non-button surface",
      pattern: /<div\b[^>\n]*onClick=/g,
      exclude: isExcludedPath,
    },
  ],
})
