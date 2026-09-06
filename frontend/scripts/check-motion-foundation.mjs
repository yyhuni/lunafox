#!/usr/bin/env node

import { runGuardrail } from "./ui-foundation-guardrail-lib.mjs"

const guardrailContract = {
  modes: ["inventory", "verify"],
  ledger: "foundation-exceptions.json",
  roots: ["components", "app"],
}
void guardrailContract

const motionOwnerPaths = [
  "app/globals.css",
  "app/layout.tsx",
  "components/pixel-blast.tsx",

  "components/auth/terminal-login",
  "components/shared/visualization/",
  "components/animate-ui/",
  "components/ui/",
]

function isExcludedPath(file) {
  return (
    file.includes("__tests__") ||
    file.includes("prototype") ||
    file.includes("demo") ||
    motionOwnerPaths.some((scope) => file.includes(scope))
  )
}

runGuardrail({
  checkId: "motion",
  title: "Motion foundation check",
  rules: [
    {
      id: "layout-shifting-hover-motion",
      label: "layout shifting hover motion",
      pattern: /\b(?:hover|group-hover):(?:-?translate-[xy]|-?(?:top|right|bottom|left|m|mt|mr|mb|ml|p|pt|pr|pb|pl|w|h)-)/g,
      exclude: isExcludedPath,
    },
    {
      id: "transition-all",
      label: "transition-all without bounded property owner",
      pattern: /transition-all/g,
      exclude: isExcludedPath,
    },
    {
      id: "prefers-reduced-motion",
      label: "raw animation declaration outside reduced-motion owner",
      pattern: /animation:\s*[^;]+(?:infinite|linear|ease)/g,
      exclude: isExcludedPath,
    },
  ],
})
