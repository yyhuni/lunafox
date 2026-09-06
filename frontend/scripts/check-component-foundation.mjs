#!/usr/bin/env node

import { runGuardrail } from "./ui-foundation-guardrail-lib.mjs"

const guardrailContract = {
  modes: ["inventory", "verify"],
  ledger: "foundation-exceptions.json",
  roots: ["components", "app", "hooks", "lib"],
}
void guardrailContract

const rawStableConceptIconImportPattern = /import\s*\{(?=[^}]*\b(?:Target|IconTarget|Building2|IconBuilding|Bug|IconBug|Radar|IconRadar|Globe|IconGlobe|Network|IconNetwork|Server|IconServer|Link|Link2|IconLink|Folder|FolderOpen|IconFolder|Camera|IconCamera|IconStack2|Fingerprint|IconFingerprint|Wrench|IconTool|IconWorldSearch)\b)[^}]*\}\s*from\s*["']@\/components\/icons["']/g

function isExcludedPath(file) {
  return (
    file.includes("__tests__") ||
    file.includes("prototype") ||
    file.includes("demo") ||
    file.startsWith("components/icons/")
  )
}

runGuardrail({
  checkId: "component",
  title: "Component foundation check",
  rules: [
    {
      id: "direct-icon-import",
      label: "direct icon import outside components/icons",
      pattern:
        /@(?:icon-park\/react|tabler\/icons-react|phosphor-icons\/react|fluentui\/react-icons|solar-icons\/react|lucide-react)/g,
      exclude: isExcludedPath,
    },
    {
      id: "raw-stable-concept-icon-import",
      label: "raw stable product-concept icon import outside semantic registry",
      pattern: rawStableConceptIconImportPattern,
      exclude: isExcludedPath,
    },
    {
      id: "parallel-ui-entrypoint",
      label: "legacy UI entrypoint import",
      pattern:
        /@\/components\/ui\/(?:terminal|yaml-editor|yaml-viewer|code-editor|spinner|wave-grid|mermaid-diagram|data-table(?:\/[^"')\s]+)?|overlay-styles)/g,
      exclude: isExcludedPath,
    },
    {
      id: "excluded-route-import",
      label: "production import from excluded demo/prototype boundary",
      pattern:
        /from\s+["'][^"']*(?:prototype|prototypes|demo|visual-lab|playground)[^"']*["']/g,
      exclude: isExcludedPath,
    },
  ],
  scanRoots: ["app", "components", "hooks", "lib"],
})
