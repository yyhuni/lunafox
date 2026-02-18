#!/usr/bin/env node

import fs from "node:fs/promises"
import path from "node:path"

const projectRoot = process.cwd()
const matrixPath = path.join(projectRoot, "test-plan", "interaction-matrix.json")
const outputPath = path.join(projectRoot, "test-plan", "boundary-cases.todo.json")

function nowIso() {
  return new Date().toISOString()
}

function slugify(value) {
  return value
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 64)
}

function normalizeLabel(label) {
  if (!label || label === "<unnamed>") return "unnamed-control"
  return label
}

function isBoundaryTarget(item) {
  const tags = new Set(["input", "textarea", "select"])
  const roles = new Set(["textbox", "combobox", "searchbox", "spinbutton"])
  return tags.has(item.tag) || roles.has(item.role)
}

function buildCaseTemplates() {
  return [
    {
      id: "normal_ascii",
      title: "正常输入",
      value: "example-input-01",
      expected: "accept-or-normalize",
    },
    {
      id: "empty_value",
      title: "空值",
      value: "",
      expected: "validation-message-or-safe-default",
    },
    {
      id: "spaces_only",
      title: "仅空格",
      value: "     ",
      expected: "trim-or-reject",
    },
    {
      id: "long_513",
      title: "超长输入(513)",
      value: "a".repeat(513),
      expected: "truncate-or-validation-error",
    },
    {
      id: "special_chars",
      title: "特殊字符",
      value: "!@#$%^&*()_+-=[]{}|;':,./<>?",
      expected: "escaped-or-rejected",
    },
    {
      id: "script_like",
      title: "脚本注入样式",
      value: "<script>alert(1)</script>",
      expected: "escaped-or-rejected",
    },
    {
      id: "sql_like",
      title: "SQL 注入样式",
      value: "' OR '1'='1' --",
      expected: "escaped-or-rejected",
    },
    {
      id: "unicode_mix",
      title: "多语言混合",
      value: "测试-test-123",
      expected: "consistent-validation",
    },
  ]
}

async function main() {
  const raw = await fs.readFile(matrixPath, "utf8")
  const matrix = JSON.parse(raw)

  const controlTasks = []

  for (const route of matrix.routes || []) {
    if (route.error || route.status >= 500) continue

    let localIndex = 0
    for (const item of route.items || []) {
      if (!isBoundaryTarget(item)) continue

      localIndex += 1
      const label = normalizeLabel(item.label)
      const controlKey = slugify(`${route.route}-${label}-${item.type || item.tag}-${localIndex}`)

      controlTasks.push({
        id: `${route.id || "route"}-${controlKey}`,
        route: route.route,
        locale: route.locale,
        control: {
          tag: item.tag,
          role: item.role,
          type: item.type || null,
          label,
          disabled: !!item.disabled,
          danger: !!item.danger,
        },
        status: "pending",
        attempts: 0,
        lastRunAt: null,
        cases: buildCaseTemplates().map((testCase) => ({
          ...testCase,
          status: "pending",
          lastError: "",
        })),
      })
    }
  }

  const payload = {
    schemaVersion: 1,
    generatedAt: nowIso(),
    source: "scripts/build-boundary-todo.mjs",
    summary: {
      totalControls: controlTasks.length,
      totalCases: controlTasks.reduce((acc, item) => acc + item.cases.length, 0),
      statusCounts: {
        pending: controlTasks.length,
        running: 0,
        passed: 0,
        failed: 0,
        skipped: 0,
      },
    },
    controls: controlTasks,
  }

  await fs.writeFile(outputPath, `${JSON.stringify(payload, null, 2)}\n`, "utf8")

  console.log(`boundary todo written: ${outputPath}`)
  console.log(`controls=${payload.summary.totalControls} cases=${payload.summary.totalCases}`)
}

main().catch((error) => {
  console.error(error)
  process.exit(1)
})
