#!/usr/bin/env node

import fs from "node:fs/promises"
import path from "node:path"
import { spawn } from "node:child_process"

const projectRoot = process.cwd()
const testPlanDir = path.join(projectRoot, "test-plan")

const fullTodoPath = path.join(testPlanDir, "full-coverage.todo.json")
const fullProgressPath = path.join(testPlanDir, "full-coverage-progress.md")
const routeReportPath = path.join(testPlanDir, "last-run.json")

const baseUrl = process.env.SMOKE_BASE_URL || "http://192.168.5.7:3000"
const locales = process.env.SMOKE_LOCALES || "zh"
const smokeConcurrency = process.env.SMOKE_CONCURRENCY || "4"
const retryConcurrency = process.env.SMOKE_RETRY_CONCURRENCY || "2"
const matrixConcurrency = process.env.MATRIX_CONCURRENCY || "3"
const maxRounds = Number.parseInt(process.env.FULL_MAX_ROUNDS || "8", 10)
const failFast = process.env.FULL_FAIL_FAST !== "0"

function nowIso() {
  return new Date().toISOString()
}

function defaultTodo() {
  const timestamp = nowIso()
  return {
    schemaVersion: 1,
    createdAt: timestamp,
    updatedAt: timestamp,
    objective: "完整覆盖：全按钮、全功能、边界场景、危险操作安全验证",
    status: "pending",
    doneCriteria: [
      "路由巡检全绿（目标路由 100% 通过）",
      "交互清单已生成并覆盖所有可见交互控件",
      "边界用例清单已生成并进入执行状态",
      "高风险/破坏性操作已在安全策略下验证",
      "最终回归巡检通过（无新增 blocker）",
    ],
    resumeCommand: "cd /Users/yangyang/Desktop/lunafox/frontend && pnpm run test:e2e:full:loop",
    phases: [
      {
        id: "route_smoke",
        title: "路由与核心交互巡检",
        auto: true,
        status: "pending",
        attempts: 0,
        lastRunAt: null,
        notes: "",
      },
      {
        id: "interaction_inventory",
        title: "交互控件清单生成",
        auto: true,
        status: "pending",
        attempts: 0,
        lastRunAt: null,
        notes: "",
      },
      {
        id: "boundary_plan",
        title: "边界用例清单生成",
        auto: true,
        status: "pending",
        attempts: 0,
        lastRunAt: null,
        notes: "",
      },
      {
        id: "button_assertions",
        title: "按钮功能断言覆盖",
        auto: false,
        status: "pending",
        attempts: 0,
        lastRunAt: null,
        notes: "需要按模块补充断言型 e2e 用例（新增/编辑/停止/导入等）。",
      },
      {
        id: "boundary_execution",
        title: "边界用例执行与修复",
        auto: false,
        status: "pending",
        attempts: 0,
        lastRunAt: null,
        notes: "需要按控件逐项执行 boundary-cases.todo.json。",
      },
      {
        id: "destructive_safe_checks",
        title: "危险操作安全校验",
        auto: false,
        status: "pending",
        attempts: 0,
        lastRunAt: null,
        notes: "删除/停止/重置等需在 mock 或隔离数据下验证并记录回滚策略。",
      },
      {
        id: "final_regression",
        title: "最终全量回归",
        auto: true,
        status: "pending",
        attempts: 0,
        lastRunAt: null,
        notes: "所有阶段完成后，执行最终并发巡检与质量门禁。",
      },
    ],
    summary: {
      runCount: 0,
      autoPassed: 0,
      autoFailed: 0,
      manualPending: 0,
    },
  }
}

async function loadTodo() {
  try {
    const raw = await fs.readFile(fullTodoPath, "utf8")
    return JSON.parse(raw)
  } catch {
    return defaultTodo()
  }
}

async function writeTodo(todo) {
  todo.updatedAt = nowIso()
  await fs.writeFile(fullTodoPath, `${JSON.stringify(todo, null, 2)}\n`, "utf8")
}

async function ensureProgressFile() {
  try {
    await fs.access(fullProgressPath)
  } catch {
    const header = [
      "# Full Coverage 执行进度",
      "",
      "- 目标：完整覆盖（按钮/功能/边界/高风险操作）",
      "- 续跑命令：`pnpm run test:e2e:full:loop`",
      "",
    ].join("\n")
    await fs.writeFile(fullProgressPath, header, "utf8")
  }
}

async function appendProgress(lines) {
  await ensureProgressFile()
  await fs.appendFile(fullProgressPath, `\n${lines.join("\n")}\n`, "utf8")
}

function getPhase(todo, phaseId) {
  const phase = todo.phases.find((item) => item.id === phaseId)
  if (!phase) {
    throw new Error(`missing phase: ${phaseId}`)
  }
  return phase
}

function updatePhase(todo, phaseId, patch) {
  const phase = getPhase(todo, phaseId)
  Object.assign(phase, patch)
  if (patch.status) {
    phase.lastRunAt = nowIso()
    phase.attempts = (phase.attempts || 0) + 1
  }
}

function recomputeSummary(todo) {
  const autoPhases = todo.phases.filter((item) => item.auto)
  const manualPhases = todo.phases.filter((item) => !item.auto)

  const autoPassed = autoPhases.filter((item) => item.status === "passed").length
  const autoFailed = autoPhases.filter((item) => item.status === "failed").length
  const manualPending = manualPhases.filter((item) => item.status !== "passed").length

  todo.summary = {
    ...todo.summary,
    autoPassed,
    autoFailed,
    manualPending,
  }

  if (autoFailed > 0) {
    todo.status = "failed"
  } else if (autoPassed === autoPhases.length && manualPending === 0) {
    todo.status = "passed"
  } else if (autoPassed === autoPhases.length) {
    todo.status = "in_progress"
  } else {
    todo.status = "running"
  }
}

async function runCommand(command, args, extraEnv = {}) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, {
      cwd: projectRoot,
      stdio: "inherit",
      env: {
        ...process.env,
        ...extraEnv,
      },
    })

    child.on("error", reject)
    child.on("close", (code) => resolve(code ?? 0))
  })
}

async function readRouteReport() {
  try {
    const raw = await fs.readFile(routeReportPath, "utf8")
    return JSON.parse(raw)
  } catch {
    return null
  }
}

async function runRouteSmokeLoop(todo, phaseId, options = {}) {
  const rounds = options.rounds || maxRounds
  const regenerate = options.regenerate !== false

  updatePhase(todo, phaseId, {
    status: "running",
    notes: `baseUrl=${baseUrl}; locales=${locales}; rounds<=${rounds}`,
  })
  await writeTodo(todo)

  if (regenerate) {
    const code = await runCommand("node", ["scripts/build-route-todo.mjs"])
    if (code !== 0) {
      updatePhase(todo, phaseId, {
        status: "failed",
        notes: "build-route-todo.mjs failed",
      })
      recomputeSummary(todo)
      await writeTodo(todo)
      return false
    }
  }

  for (let round = 1; round <= rounds; round += 1) {
    const env = {
      SMOKE_BASE_URL: baseUrl,
      SMOKE_LOCALES: locales,
      SMOKE_CONCURRENCY: round === 1 ? smokeConcurrency : retryConcurrency,
    }

    if (round > 1) {
      env.SMOKE_ONLY_FAILED = "1"
    }

    const code = await runCommand("node", ["scripts/run-route-smoke.mjs"], env)
    const report = await readRouteReport()
    const failed = report?.failed ?? (code === 0 ? 0 : -1)
    const passed = report?.passed ?? -1
    const total = report?.total ?? -1

    await appendProgress([
      `## 轮次 ${nowIso()}`,
      `- phase: \`${phaseId}\``,
      `- round: \`${round}/${rounds}\``,
      `- routes: total=\`${total}\` passed=\`${passed}\` failed=\`${failed}\``,
      `- result: ${failed === 0 ? "`all-green`" : "`has-failures`"}`,
    ])

    if (failed === 0) {
      updatePhase(todo, phaseId, {
        status: "passed",
        notes: `round=${round}; total=${total}; passed=${passed}`,
      })
      recomputeSummary(todo)
      await writeTodo(todo)
      return true
    }
  }

  updatePhase(todo, phaseId, {
    status: "failed",
    notes: `exceeded max rounds (${rounds})`,
  })
  recomputeSummary(todo)
  await writeTodo(todo)
  return false
}

async function runPhaseCommand(todo, phaseId, command, args, env = {}, successNote = "") {
  updatePhase(todo, phaseId, {
    status: "running",
    notes: successNote,
  })
  recomputeSummary(todo)
  await writeTodo(todo)

  const code = await runCommand(command, args, env)
  if (code !== 0) {
    updatePhase(todo, phaseId, {
      status: "failed",
      notes: `${args.join(" ")} failed with code=${code}`,
    })
    recomputeSummary(todo)
    await writeTodo(todo)
    return false
  }

  updatePhase(todo, phaseId, {
    status: "passed",
    notes: successNote || `${args.join(" ")} passed`,
  })
  recomputeSummary(todo)
  await writeTodo(todo)
  return true
}

async function main() {
  const todo = await loadTodo()
  todo.summary.runCount = (todo.summary.runCount || 0) + 1
  recomputeSummary(todo)
  await writeTodo(todo)

  await appendProgress([
    `## 轮次 ${nowIso()}`,
    "- mode: `full-coverage-loop`",
    `- baseUrl: \`${baseUrl}\``,
    `- locales: \`${locales}\``,
    `- smokeConcurrency: \`${smokeConcurrency}\``,
    `- retryConcurrency: \`${retryConcurrency}\``,
    `- matrixConcurrency: \`${matrixConcurrency}\``,
    `- maxRounds: \`${maxRounds}\``,
  ])

  const routeOk = await runRouteSmokeLoop(todo, "route_smoke", {
    regenerate: true,
    rounds: maxRounds,
  })
  if (!routeOk && failFast) {
    process.exit(1)
  }

  const matrixOk = await runPhaseCommand(
    todo,
    "interaction_inventory",
    "node",
    ["scripts/build-interaction-matrix.mjs"],
    {
      SMOKE_BASE_URL: baseUrl,
      SMOKE_LOCALES: locales,
      MATRIX_CONCURRENCY: matrixConcurrency,
    },
    "interaction-matrix.json updated"
  )
  if (!matrixOk && failFast) {
    process.exit(1)
  }

  const boundaryPlanOk = await runPhaseCommand(
    todo,
    "boundary_plan",
    "node",
    ["scripts/build-boundary-todo.mjs"],
    {},
    "boundary-cases.todo.json updated"
  )
  if (!boundaryPlanOk && failFast) {
    process.exit(1)
  }

  const regressionOk = await runRouteSmokeLoop(todo, "final_regression", {
    regenerate: true,
    rounds: 3,
  })

  const manualPhaseIds = [
    "button_assertions",
    "boundary_execution",
    "destructive_safe_checks",
  ]

  for (const phaseId of manualPhaseIds) {
    const phase = getPhase(todo, phaseId)
    if (phase.status === "pending") {
      phase.notes = `${phase.notes} 待继续执行。`
    }
  }

  recomputeSummary(todo)
  await writeTodo(todo)

  const failedPhases = todo.phases.filter((item) => item.status === "failed")
  console.log(`full-coverage status=${todo.status}`)
  if (failedPhases.length > 0) {
    console.log("failed phases:")
    for (const phase of failedPhases) {
      console.log(`- ${phase.id}: ${phase.notes}`)
    }
  }

  if (!regressionOk && failFast) {
    process.exit(1)
  }
}

main().catch((error) => {
  console.error(error)
  process.exit(1)
})
