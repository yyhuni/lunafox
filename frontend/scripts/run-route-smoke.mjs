#!/usr/bin/env node

import fs from "node:fs/promises"
import path from "node:path"
import { chromium } from "@playwright/test"

const projectRoot = process.cwd()
const todoPath = path.join(projectRoot, "test-plan", "routes.todo.json")
const progressPath = path.join(projectRoot, "test-plan", "progress.md")
const failuresPath = path.join(projectRoot, "test-plan", "failures.md")
const reportPath = path.join(projectRoot, "test-plan", "last-run.json")

const baseUrl = process.env.SMOKE_BASE_URL || "http://192.168.5.7:3000"
const localeFilter = process.env.SMOKE_LOCALES
  ? process.env.SMOKE_LOCALES.split(",").map((x) => x.trim()).filter(Boolean)
  : null

const concurrency = Number.parseInt(process.env.SMOKE_CONCURRENCY || "4", 10)
const maxClicks = Number.parseInt(process.env.SMOKE_MAX_CLICKS || "8", 10)
const maxButtonCandidates = Number.parseInt(process.env.SMOKE_MAX_BUTTON_CANDIDATES || "40", 10)
const limit = Number.parseInt(process.env.SMOKE_LIMIT || "0", 10)
const onlyFailed = process.env.SMOKE_ONLY_FAILED === "1"
const includePending = !onlyFailed

const dangerPattern = /(删除|清空|停止|注销|退出|重置|移除|销毁|delete|remove|stop|logout|sign out|clear|reset|revoke)/i
const hydrationPattern = /(hydration failed|didn't match|recoverable error|server rendered html)/i

function normalizeRoute(routePattern, locale) {
  let route = routePattern.replace("/[locale]", `/${locale}`)
  route = route.replace(/\[id\]/g, "1")
  return route
}

function shouldAuth(route) {
  return !/\/login\/?$/.test(route)
}

function nowIso() {
  return new Date().toISOString()
}

function summarizeError(err) {
  if (!err) return ""
  if (typeof err === "string") return err
  if (err instanceof Error) return err.message
  try {
    return JSON.stringify(err)
  } catch {
    return String(err)
  }
}

async function loadTodo() {
  const raw = await fs.readFile(todoPath, "utf8")
  return JSON.parse(raw)
}

async function saveTodo(payload) {
  await fs.writeFile(todoPath, `${JSON.stringify(payload, null, 2)}\n`, "utf8")
}

async function appendProgress(run) {
  const block = [
    "",
    `## 轮次 ${run.startedAt}`,
    `- 基础地址：\`${baseUrl}\``,
    `- 并发：\`${concurrency}\``,
    `- 执行项：\`${run.total}\``,
    `- 通过：\`${run.passed}\``,
    `- 失败：\`${run.failed}\``,
    `- 覆盖路由：\`${run.routeCount}\``,
    run.failed === 0 ? "- 结果：`all-green`" : "- 结果：`has-failures`",
    "",
  ].join("\n")
  await fs.appendFile(progressPath, block, "utf8")
}

async function appendFailures(failures) {
  if (!failures.length) return
  const lines = failures
    .map((f) => {
      const consoleMsg = f.hydrationLogs.join(" | ").replace(/\|/g, "/")
      const scene = `locale=${f.locale}`
      const err = (f.error || "-").replace(/\|/g, "/")
      const route = f.route
      return `| ${nowIso()} | ${route} | ${scene} | ${err} | ${consoleMsg || "-"} | pending-fix | ${f.files.join(", ")} |`
    })
    .join("\n")
  await fs.appendFile(failuresPath, `\n${lines}\n`, "utf8")
}

async function clickSafeButtons(page, clicked) {
  const locator = page.locator("button:visible, [role='button']:visible")
  const total = Math.min(await locator.count(), maxButtonCandidates)
  for (let i = 0; i < total && clicked.length < maxClicks; i += 1) {
    const btn = locator.nth(i)
    const text = ((await btn.innerText().catch(() => "")) || "").trim()
    const aria = ((await btn.getAttribute("aria-label").catch(() => "")) || "").trim()
    const name = `${text} ${aria}`.trim()
    if (!name) continue
    if (dangerPattern.test(name)) continue

    try {
      await btn.click({ timeout: 1500 })
      clicked.push(name)
      await page.waitForTimeout(250)
      await page.keyboard.press("Escape").catch(() => {})
    } catch {
      // ignore non-clickable transient elements
    }
  }
}

async function verifyDashboardQuickScan(page) {
  const header = page.locator("[data-slot='unified-header']").first()
  await header.waitFor({ state: "visible", timeout: 8000 }).catch(() => {})

  const quickBtn = page.getByRole("button", { name: /快速扫描|Quick Scan/i }).first()
  await quickBtn.waitFor({ state: "visible", timeout: 8000 }).catch(() => {})
  if ((await quickBtn.count()) === 0 || !(await quickBtn.isVisible().catch(() => false))) {
    return { ok: false, error: "missing quick-scan button" }
  }

  await quickBtn.click({ timeout: 5000 })
  await page.waitForTimeout(300)

  const dialog = page.locator("[role='dialog']").filter({ hasText: /快速扫描|Quick Scan|scanTargets|扫描目标/i }).first()
  if ((await dialog.count()) === 0 || !(await dialog.isVisible().catch(() => false))) {
    return { ok: false, error: "quick-scan dialog did not open" }
  }

  await page.keyboard.press("Escape").catch(() => {})
  return { ok: true, error: "" }
}

async function runOne(browser, target) {
  const auth = shouldAuth(target.route)
  const context = await browser.newContext()

  if (auth) {
    await context.addInitScript(() => {
      window.localStorage.setItem("accessToken", "mock-access-token")
      window.localStorage.setItem("refreshToken", "mock-refresh-token")
    })
  }

  const page = await context.newPage()
  const hydrationLogs = []
  const pageErrors = []
  const clicked = []

  page.on("console", (msg) => {
    const text = msg.text()
    if (hydrationPattern.test(text)) hydrationLogs.push(`[console:${msg.type()}] ${text}`)
  })
  page.on("pageerror", (err) => {
    if (hydrationPattern.test(err.message)) {
      hydrationLogs.push(`[pageerror] ${err.message}`)
    } else {
      pageErrors.push(err.message)
    }
  })

  let error = ""
  let finalUrl = ""
  let status = 0

  try {
    const response = await page.goto(`${baseUrl}${target.route}`, {
      waitUntil: "domcontentloaded",
      timeout: 45000,
    })
    status = response?.status() ?? 0
    await page.waitForTimeout(1200)
    finalUrl = page.url()

    if (status >= 500) {
      error = `http-${status}`
    }

    if (!error && auth && /\/login\/?$/.test(new URL(finalUrl).pathname)) {
      error = "unexpected redirect to login"
    }

    if (!error && /\/dashboard\/?$/.test(target.route)) {
      const quick = await verifyDashboardQuickScan(page)
      if (!quick.ok) error = quick.error
    }

    if (!error) {
      await clickSafeButtons(page, clicked)
    }

    if (!error && hydrationLogs.length > 0) {
      error = "hydration mismatch logs detected"
    }
  } catch (err) {
    error = summarizeError(err)
  } finally {
    await context.close()
  }

  return {
    ...target,
    passed: !error,
    error,
    status,
    finalUrl,
    hydrationLogs,
    pageErrors,
    clicked,
  }
}

async function runPool(items, worker, size) {
  const results = []
  let index = 0

  async function next() {
    const i = index
    index += 1
    if (i >= items.length) return
    const item = items[i]
    const result = await worker(item)
    results.push(result)
    await next()
  }

  await Promise.all(Array.from({ length: Math.max(1, size) }, () => next()))
  return results
}

function buildTargets(todo) {
  const candidates = todo.routes.filter((route) => {
    if (onlyFailed) return route.status === "failed"
    if (includePending) return route.status === "pending" || route.status === "failed"
    return true
  })

  const targets = []
  for (const route of candidates) {
    const locales = localeFilter?.length
      ? route.locales.filter((x) => localeFilter.includes(x))
      : route.locales

    for (const locale of locales) {
      targets.push({
        id: route.id,
        routePattern: route.routePattern,
        locale,
        route: normalizeRoute(route.routePattern, locale),
      })
    }
  }

  if (limit > 0) return targets.slice(0, limit)
  return targets
}

function groupByRoute(results) {
  const map = new Map()
  for (const result of results) {
    const key = result.routePattern
    const group = map.get(key) || []
    group.push(result)
    map.set(key, group)
  }
  return map
}

function updateTodoWithResults(todo, results) {
  const grouped = groupByRoute(results)
  const executedRoutes = new Set()
  for (const route of todo.routes) {
    const group = grouped.get(route.routePattern)
    if (!group || group.length === 0) continue

    executedRoutes.add(route.routePattern)
    route.attempts = (route.attempts || 0) + 1
    route.lastRunAt = nowIso()
    const failed = group.filter((x) => !x.passed)
    route.status = failed.length > 0 ? "failed" : "passed"
    route.lastResult = {
      testedLocales: group.map((x) => x.locale),
      passCount: group.length - failed.length,
      failCount: failed.length,
    }
    route.notes = failed[0]?.error || ""
  }

  const statusCounts = todo.routes.reduce(
    (acc, route) => {
      const s = route.status || "pending"
      acc[s] = (acc[s] || 0) + 1
      return acc
    },
    { pending: 0, running: 0, passed: 0, failed: 0, skipped: 0 }
  )

  todo.summary = {
    ...todo.summary,
    lastRunAt: nowIso(),
    executedRoutes: executedRoutes.size,
    statusCounts,
  }
}

async function main() {
  const startedAt = nowIso()
  const todo = await loadTodo()
  const targets = buildTargets(todo)

  if (targets.length === 0) {
    console.log("no targets to run")
    return
  }

  console.log(`baseUrl=${baseUrl}`)
  console.log(`targets=${targets.length} concurrency=${concurrency}`)

  const browser = await chromium.launch({ headless: true })
  const results = await runPool(targets, (target) => runOne(browser, target), concurrency)
  await browser.close()

  const failed = results.filter((x) => !x.passed)
  const passed = results.length - failed.length

  updateTodoWithResults(todo, results)
  await saveTodo(todo)

  const runReport = {
    startedAt,
    finishedAt: nowIso(),
    baseUrl,
    total: results.length,
    passed,
    failed: failed.length,
    routeCount: new Set(results.map((x) => x.routePattern)).size,
    failures: failed.map((x) => ({
      routePattern: x.routePattern,
      route: x.route,
      locale: x.locale,
      error: x.error,
      status: x.status,
      finalUrl: x.finalUrl,
      hydrationLogs: x.hydrationLogs,
    })),
  }

  await fs.writeFile(reportPath, `${JSON.stringify(runReport, null, 2)}\n`, "utf8")

  await appendProgress({
    startedAt,
    total: results.length,
    passed,
    failed: failed.length,
    routeCount: runReport.routeCount,
  })

  await appendFailures(
    failed.map((x) => ({
      route: x.route,
      locale: x.locale,
      error: x.error,
      hydrationLogs: x.hydrationLogs,
      files: ["frontend/components/auth/auth-layout.tsx", "frontend/components/scan/quick-scan-dialog.tsx"],
    }))
  )

  console.log(`passed=${passed} failed=${failed.length}`)
  if (failed.length > 0) {
    console.log("failure samples:")
    for (const item of failed.slice(0, 10)) {
      console.log(`- ${item.route} [${item.locale}] => ${item.error}`)
    }
    process.exitCode = 1
  }
}

main().catch((error) => {
  console.error(error)
  process.exit(1)
})
