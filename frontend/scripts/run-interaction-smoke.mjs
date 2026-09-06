#!/usr/bin/env node

import fs from "node:fs/promises"
import path from "node:path"
import { spawnSync } from "node:child_process"
import { chromium } from "@playwright/test"
import {
  createSmokeAuthBootstrap,
  primeSmokeAuthSession,
  primeSmokeLocaleCookie,
} from "../lib/auth-runtime.mjs"
import {
  acquireSmokeDevServerLease,
  releaseDevServerLease,
} from "./smoke-dev-server.mjs"
import { instantiateRoutePattern } from "./route-pattern.mjs"

const projectRoot = process.cwd()
const todoPath = path.join(projectRoot, "test-plan", "routes.todo.json")
const reportPath = path.join(projectRoot, "test-plan", "interaction-smoke.json")
const runLockPath = path.join(projectRoot, "test-plan", ".interaction-smoke.lock")

const baseUrl = process.env.INTERACTION_BASE_URL || process.env.SMOKE_BASE_URL || "http://127.0.0.1:3000"
const localeFilter = process.env.INTERACTION_LOCALES
  ? process.env.INTERACTION_LOCALES.split(",").map((item) => item.trim()).filter(Boolean)
  : process.env.SMOKE_LOCALES
    ? process.env.SMOKE_LOCALES.split(",").map((item) => item.trim()).filter(Boolean)
    : ["zh"]

const concurrency = Number.parseInt(process.env.INTERACTION_CONCURRENCY || "1", 10)
const maxClicks = Number.parseInt(process.env.INTERACTION_MAX_CLICKS || "6", 10)
const routeTimeoutMs = Number.parseInt(process.env.INTERACTION_ROUTE_TIMEOUT_MS || "60000", 10)
const limit = Number.parseInt(process.env.INTERACTION_LIMIT || "0", 10)
const onlyFailed = process.env.INTERACTION_ONLY_FAILED === "1"
const strictMode = process.env.INTERACTION_STRICT === "1"
const autoStartServer = process.env.INTERACTION_AUTO_START_SERVER !== "0"

const viewportPresets = {
  desktop: {
    width: 1280,
    height: 720,
    isMobile: false,
    hasTouch: false,
  },
  mobile: {
    width: 390,
    height: 844,
    isMobile: true,
    hasTouch: true,
  },
}

const dangerPattern = /(删除|清空|停止|注销|退出|重置|移除|销毁|delete|remove|stop|logout|sign out|clear|reset|revoke)/i
const redirectOnlyInteractionRoutePatterns = new Set([
  "/",
  "/login",
])

function nowIso() {
  return new Date().toISOString()
}

function formatElapsed(ms) {
  const seconds = Math.max(0, Math.floor(ms / 1000))
  const minutes = Math.floor(seconds / 60)
  const restSeconds = seconds % 60
  return `${String(minutes).padStart(2, "0")}:${String(restSeconds).padStart(2, "0")}`
}

function parsePositiveInteger(value, fallback) {
  const parsed = Number.parseInt(value || "", 10)
  if (!Number.isInteger(parsed) || parsed <= 0) {
    return fallback
  }
  return parsed
}

function resolveViewportConfig() {
  const requestedName = (process.env.INTERACTION_VIEWPORT || process.env.SMOKE_VIEWPORT || "desktop").trim()
  const presetName = Object.hasOwn(viewportPresets, requestedName) ? requestedName : "desktop"
  const preset = viewportPresets[presetName]
  return {
    name: presetName,
    width: parsePositiveInteger(process.env.INTERACTION_VIEWPORT_WIDTH || process.env.SMOKE_VIEWPORT_WIDTH, preset.width),
    height: parsePositiveInteger(process.env.INTERACTION_VIEWPORT_HEIGHT || process.env.SMOKE_VIEWPORT_HEIGHT, preset.height),
    isMobile: preset.isMobile,
    hasTouch: preset.hasTouch,
  }
}

const viewportConfig = resolveViewportConfig()

function installShutdownCleanup(closeResources) {
  const signalExitCodes = new Map([
    ["SIGINT", 130],
    ["SIGTERM", 143],
    ["SIGHUP", 129],
  ])
  let shuttingDown = false
  const subscriptions = []
  for (const signalName of signalExitCodes.keys()) {
    const handler = async () => {
      if (shuttingDown) {
        return
      }
      shuttingDown = true
      try {
        await closeResources()
      } finally {
        process.exit(signalExitCodes.get(signalName) || 1)
      }
    }
    process.once(signalName, handler)
    subscriptions.push([signalName, handler])
  }
  return () => {
    for (const [signalName, handler] of subscriptions) {
      process.off(signalName, handler)
    }
  }
}

function listChildPids(parentPid) {
  const result = spawnSync("pgrep", ["-P", String(parentPid)], {
    encoding: "utf8",
  })
  if (result.status !== 0 || !result.stdout) {
    return []
  }
  return result.stdout
    .split(/\s+/)
    .map((value) => Number.parseInt(value, 10))
    .filter((value) => Number.isInteger(value) && value > 0)
}

function collectProcessTreePids(rootPid) {
  const root = Number.parseInt(String(rootPid), 10)
  if (!Number.isInteger(root) || root <= 0) {
    return []
  }
  const visited = new Set([root])
  const queue = [root]
  const ordered = [root]
  while (queue.length > 0) {
    const current = queue.shift()
    const children = listChildPids(current)
    for (const childPid of children) {
      if (visited.has(childPid)) {
        continue
      }
      visited.add(childPid)
      queue.push(childPid)
      ordered.push(childPid)
    }
  }
  return ordered
}

function isPidAlive(pid) {
  try {
    process.kill(pid, 0)
    return true
  } catch {
    return false
  }
}

function signalPid(pid, signal) {
  try {
    process.kill(pid, signal)
  } catch {}
}

function waitMs(ms) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms)
  })
}

async function cleanupProcessTree(rootPid) {
  const initialPids = collectProcessTreePids(rootPid).reverse()
  if (initialPids.length === 0) {
    return
  }
  for (const pid of initialPids) {
    if (isPidAlive(pid)) {
      signalPid(pid, "SIGTERM")
    }
  }
  await waitMs(250)
  for (const pid of initialPids) {
    if (isPidAlive(pid)) {
      signalPid(pid, "SIGKILL")
    }
  }
}

function shouldAuth(route) {
  return !/\/login\/?$/.test(route)
}

function summarizeError(error) {
  if (!error) return ""
  if (typeof error === "string") return error
  if (error instanceof Error) return error.message
  try {
    return JSON.stringify(error)
  } catch {
    return String(error)
  }
}

async function loadTodo() {
  return JSON.parse(await fs.readFile(todoPath, "utf8"))
}

async function readRunLockState() {
  try {
    const raw = await fs.readFile(runLockPath, "utf8")
    let parsed = null
    try {
      parsed = JSON.parse(raw)
    } catch {}
    const pid = Number.parseInt(String(parsed?.pid ?? ""), 10)
    return {
      raw,
      pid: Number.isInteger(pid) && pid > 0 ? pid : null,
      startedAt: typeof parsed?.startedAt === "string" ? parsed.startedAt : null,
      baseUrl: typeof parsed?.baseUrl === "string" ? parsed.baseUrl : null,
    }
  } catch (error) {
    if (error?.code === "ENOENT") {
      return null
    }
    throw error
  }
}

function formatRunLockState(lockState) {
  if (!lockState) {
    return `lock=${runLockPath}`
  }
  const parts = [`lock=${runLockPath}`]
  if (lockState.pid) {
    parts.push(`pid=${lockState.pid}`)
  }
  if (lockState.startedAt) {
    parts.push(`startedAt=${lockState.startedAt}`)
  }
  if (lockState.baseUrl) {
    parts.push(`baseUrl=${lockState.baseUrl}`)
  }
  return parts.join(" ")
}

async function clearStaleRunLock() {
  const lockState = await readRunLockState()
  if (!lockState) {
    return false
  }
  if (lockState.pid && isPidAlive(lockState.pid)) {
    return false
  }
  await fs.unlink(runLockPath).catch((error) => {
    if (error?.code !== "ENOENT") {
      throw error
    }
  })
  return true
}

async function acquireRunLock() {
  await fs.mkdir(path.dirname(runLockPath), { recursive: true })
  const payload = {
    pid: process.pid,
    startedAt: nowIso(),
    baseUrl,
  }

  for (let attempt = 0; attempt < 2; attempt += 1) {
    let handle = null
    let createdLock = false
    try {
      handle = await fs.open(runLockPath, "wx")
      createdLock = true
      await handle.writeFile(`${JSON.stringify(payload, null, 2)}\n`, "utf8")
      let released = false
      return async () => {
        if (released) {
          return
        }
        released = true
        await handle.close().catch(() => {})
        await fs.unlink(runLockPath).catch((error) => {
          if (error?.code !== "ENOENT") {
            throw error
          }
        })
      }
    } catch (error) {
      if (handle) {
        await handle.close().catch(() => {})
      }
      if (createdLock) {
        await fs.unlink(runLockPath).catch((unlinkError) => {
          if (unlinkError?.code !== "ENOENT") {
            throw unlinkError
          }
        })
      }
      if (error?.code !== "EEXIST") {
        throw error
      }
      const cleared = await clearStaleRunLock()
      if (cleared) {
        console.warn(`removed stale interaction smoke lock: ${runLockPath}`)
        continue
      }
      const lockState = await readRunLockState()
      throw new Error(`interaction smoke already running: ${formatRunLockState(lockState)}`)
    }
  }

  throw new Error(`failed to acquire interaction smoke lock: ${runLockPath}`)
}

async function writeInteractionReport(payload) {
  const tempReportPath = `${reportPath}.${process.pid}.tmp`
  const serialized = `${JSON.stringify(payload, null, 2)}\n`
  try {
    await fs.writeFile(tempReportPath, serialized, "utf8")
    await fs.rename(tempReportPath, reportPath)
  } catch (error) {
    await fs.unlink(tempReportPath).catch(() => {})
    throw error
  }
}

function buildTargets(todo) {
  const selectedRoutes = (todo.routes || []).filter((route) => {
    if (!redirectOnlyInteractionRoutePatterns.has(route.routePattern)) {
      if (onlyFailed) {
        return route.status === "failed"
      }
      return route.status === "pending" || route.status === "failed" || route.status === "passed"
    }

    return false
  })

  const targets = []
  for (const route of selectedRoutes) {
    for (const locale of route.locales || []) {
      if (localeFilter.length > 0 && !localeFilter.includes(locale)) {
        continue
      }
      targets.push({
        id: route.id,
        routePattern: route.routePattern,
        locale,
        route: instantiateRoutePattern(route.routePattern),
      })
    }
  }

  return limit > 0 ? targets.slice(0, limit) : targets
}

async function clickVisibleLocators(page, selector, clicked) {
  const locator = page.locator(selector)
  const total = await locator.count().catch(() => 0)

  for (let index = 0; index < total && clicked.length < maxClicks; index += 1) {
    const target = locator.nth(index)
    const text = ((await target.innerText().catch(() => "")) || "").trim()
    const aria = ((await target.getAttribute("aria-label").catch(() => "")) || "").trim()
    const name = `${text} ${aria}`.trim()
    if (!name || dangerPattern.test(name)) {
      continue
    }

    const beforeUrl = page.url()

    try {
      await target.click({ timeout: 2000 })
      clicked.push({
        label: name,
        selector,
        navigated: page.url() !== beforeUrl,
      })
      await page.waitForTimeout(300)
      if (page.url() !== beforeUrl) {
        await page.goBack({ waitUntil: "domcontentloaded", timeout: 5000 }).catch(() => {})
        await page.waitForTimeout(200)
      } else {
        await page.keyboard.press("Escape").catch(() => {})
      }
    } catch {
      // ignore transient controls that cannot be clicked deterministically
    }
  }
}

async function runOne(browser, target) {
  const context = await browser.newContext({
    viewport: {
      width: viewportConfig.width,
      height: viewportConfig.height,
    },
    isMobile: viewportConfig.isMobile,
    hasTouch: viewportConfig.hasTouch,
  })
  await primeSmokeLocaleCookie(context, baseUrl, target.locale)
  if (shouldAuth(target.route)) {
    await context.addInitScript(primeSmokeAuthSession, createSmokeAuthBootstrap())
  }

  const page = await context.newPage()
  const clicked = []
  let status = 0
  let finalUrl = ""
  let error = ""
  let routeTimer

  try {
    await Promise.race([
      (async () => {
        const response = await page.goto(`${baseUrl}${target.route}`, {
          waitUntil: "domcontentloaded",
          timeout: Math.min(45000, routeTimeoutMs),
        })
        status = response?.status() ?? 0
        await page.waitForTimeout(1000)
        finalUrl = page.url()

        if (status >= 500) {
          error = `http-${status}`
        } else {
          await clickVisibleLocators(page, "button:visible, [role='button']:visible", clicked)
          await clickVisibleLocators(page, "a[href]:visible, [role='tab']:visible, [role='menuitem']:visible", clicked)
        }
      })(),
      new Promise((_, reject) => {
        routeTimer = setTimeout(() => {
          void context.close().catch(() => {})
          reject(new Error(`interaction-route-timeout:${target.route}:${routeTimeoutMs}ms`))
        }, routeTimeoutMs)
      }),
    ])

  } catch (caught) {
    error = summarizeError(caught)
  } finally {
    clearTimeout(routeTimer)
    await context.close()
  }

  return {
    ...target,
    status,
    finalUrl,
    clicked,
    error,
    passed: !error,
  }
}

async function runPool(items, worker, size, onResult) {
  const results = []
  let index = 0

  async function next() {
    const current = index
    index += 1
    if (current >= items.length) {
      return
    }
    const result = await worker(items[current])
    results.push(result)
    if (typeof onResult === "function") {
      onResult(result, results.length, items.length)
    }
    await next()
  }

  await Promise.all(Array.from({ length: Math.max(1, size) }, () => next()))
  return results
}

async function main() {
  const releaseRunLock = await acquireRunLock()
  let browser = null
  let browserRootPid = null
  let devServerLease = null
  let results = []
  const startedAtMs = Date.now()
  const closeBrowser = async () => {
    if (!browser) {
      return
    }
    const activeBrowser = browser
    browser = null
    await activeBrowser.close().catch(() => {})
    if (Number.isInteger(browserRootPid) && browserRootPid > 0) {
      await cleanupProcessTree(browserRootPid)
      browserRootPid = null
    }
  }
  const closeResources = async () => {
    await closeBrowser()
    if (devServerLease) {
      const activeLease = devServerLease
      devServerLease = null
      await releaseDevServerLease({ projectRoot, lease: activeLease })
    }
    await releaseRunLock()
  }
  const removeShutdownCleanup = installShutdownCleanup(closeResources)
  try {
    const todo = await loadTodo()
    const targets = buildTargets(todo)
    if (targets.length === 0) {
      console.log("no targets for interaction smoke")
      return
    }

    console.log(`baseUrl=${baseUrl}`)
    console.log(`targets=${targets.length} concurrency=${concurrency} routeTimeoutMs=${routeTimeoutMs} viewport=${viewportConfig.name}:${viewportConfig.width}x${viewportConfig.height}`)

    devServerLease = await acquireSmokeDevServerLease({
      baseUrl,
      autoStartServer,
      projectRoot,
    })
    browser = await chromium.launch({ headless: true })
    browserRootPid = browser?.process?.()?.pid ?? null
    results = await runPool(
      targets,
      (target) => runOne(browser, target),
      concurrency,
      (result, completed, total) => {
        const elapsed = formatElapsed(Date.now() - startedAtMs)
        const status = result.passed ? "ok" : "failed"
        console.log(`[progress] ${completed}/${total} elapsed=${elapsed} route=${result.route} status=${status}`)
      }
    )
  } finally {
    removeShutdownCleanup()
    await closeResources()
  }

  const failed = results.filter((entry) => !entry.passed)
  const summary = {
    generatedAt: nowIso(),
    baseUrl,
    viewport: viewportConfig,
    localeFilter,
    routeCount: results.length,
    successCount: results.length - failed.length,
    failedCount: failed.length,
    interactionCount: results.reduce((sum, entry) => sum + entry.clicked.length, 0),
  }

  await writeInteractionReport({
    schemaVersion: 1,
    summary,
    routes: results,
  })

  console.log(`interaction report written: ${reportPath}`)
  console.log(`success=${summary.successCount} failed=${summary.failedCount} interactions=${summary.interactionCount}`)

  if (failed.length > 0 && strictMode) {
    process.exitCode = 1
  }
}

main().catch((error) => {
  console.error(error)
  process.exit(1)
})
