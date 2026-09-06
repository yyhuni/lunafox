#!/usr/bin/env node

import { runGuardrail } from "./ui-foundation-guardrail-lib.mjs"

const guardrailContract = {
  modes: ["inventory", "verify"],
  ledger: "foundation-exceptions.json",
  roots: ["components", "app"],
}
void guardrailContract

const excludedPaths = [
  "components/ui/",
  "components/pixel-blast.tsx",
  "components/faulty-terminal.tsx",

  "components/shared/visualization/",
  "components/animate-ui/",
  "components/settings/system-logs/",
]

function isExcludedPath(file) {
  return (
    file.includes("__tests__") ||
    file.includes("prototype") ||
    file.includes("demo") ||
    excludedPaths.some((scope) => file.includes(scope))
  )
}

runGuardrail({
  checkId: "spacing",
  title: "Spacing foundation check",
  rules: [
    {
      id: "arbitrary-spacing-or-size",
      label: "arbitrary spacing or size utility",
      pattern:
        /\b(?:m|mt|mr|mb|ml|mx|my|p|pt|pr|pb|pl|px|py|gap|gap-x|gap-y|w|h|min-w|max-w|min-h|max-h|size|rounded|shadow|top|left|right|bottom|inset|translate-x|translate-y|grid-cols|grid-rows)-\[[^\]]+\]/g,
      exclude: isExcludedPath,
    },
    {
      id: "important-utility",
      label: "important utility",
      pattern:
        /(?:^|\s)!(?:m|mt|mr|mb|ml|mx|my|p|pt|pr|pb|pl|px|py|gap|gap-x|gap-y|w|h|min-w|max-w|min-h|max-h|size|rounded|shadow|text|bg|border|flex|grid|block|hidden|opacity|translate|top|left|right|bottom|inset|z)-[A-Za-z0-9_:\-/.[\]()]+/g,
      exclude: isExcludedPath,
    },
  ],
})
