#!/usr/bin/env node

import { runGuardrail } from "./ui-foundation-guardrail-lib.mjs"

const guardrailContract = {
  modes: ["inventory", "verify"],
  ledger: "foundation-exceptions.json",
  roots: ["app", "components", "hooks", "lib"],
}
void guardrailContract

const tokenOwnerPaths = [
  "app/globals.css",
  "components/ui/chart.tsx",
  "components/shared/editors/codemirror-theme.ts",
  "lib/color-themes.ts",
]

const excludedVisualPaths = [
  "mock/",
]

const strictThemeAdaptedPaths = [
  "components/overview/agent-globe-card.tsx",
]
void strictThemeAdaptedPaths

function isExcludedPath(file) {
  return (
    file.includes("__tests__") ||
    file.includes("prototype") ||
    file.includes("demo") ||
    tokenOwnerPaths.some((scope) => file === scope) ||
    excludedVisualPaths.some((scope) => file.includes(scope))
  )
}

runGuardrail({
  checkId: "color",
  title: "Color foundation check",
  rules: [
    {
      id: "raw-color-literal",
      label: "raw color literal",
      pattern: /#[0-9A-Fa-f]{3,8}\b/g,
      exclude: isExcludedPath,
    },
    {
      id: "raw-color-function",
      label: "raw color function",
      pattern: /\b(?:rgb|rgba|hsl|hsla|oklch|oklab)\(/g,
      exclude: isExcludedPath,
    },
    {
      id: "raw-var-fallback",
      label: "raw CSS variable fallback",
      pattern: /var\(--[^,)]+,\s*[^)]+\)/g,
      exclude: isExcludedPath,
    },
    {
      id: "theme-locked-tailwind-color",
      label: "theme-locked Tailwind color",
      pattern: /\b(?:(?:bg|text|border|from|via|to|ring|fill|stroke|decoration|shadow)-(?:white|black)(?:\/\d{1,3})?|(?:bg|text|border|from|via|to|ring|fill|stroke|decoration|shadow)-(?:slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)-\d{2,3}(?:\/\d{1,3})?)\b/g,
      exclude: isExcludedPath,
    },
  ],
  scanRoots: ["app", "components", "hooks", "lib"],
})
