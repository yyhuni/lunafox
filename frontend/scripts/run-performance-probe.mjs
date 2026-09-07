#!/usr/bin/env node

import fs from "node:fs/promises"
import path from "node:path"
import { chromium } from "@playwright/test"
import {
  createSmokeAuthBootstrap,
  primeSmokeAuthSession,
  primeSmokeLocaleCookie,
} from "../lib/auth-runtime.mjs"
import { instantiateRoutePattern } from "./route-pattern.mjs"

const projectRoot = process.cwd()
const todoPath = path.join(projectRoot, "test-plan", "routes.todo.json")

const phase = process.env.PERF_PHASE || "before"
const baseUrl = process.env.PERF_BASE_URL || process.env.SMOKE_BASE_URL || "http://192.168.5.7:3000"
const localeFilter = process.env.PERF_LOCALES
  ? process.env.PERF_LOCALES.split(",").map((item) => item.trim()).filter(Boolean)
  : process.env.SMOKE_LOCALES
    ? process.env.SMOKE_LOCALES.split(",").map((item) => item.trim()).filter(Boolean)
    : ["zh"]
const routeFilter = process.env.PERF_ROUTE_FILTER
  ? process.env.PERF_ROUTE_FILTER.split(",").map((item) => item.trim()).filter(Boolean)
  : []
const limit = Number.parseInt(process.env.PERF_LIMIT || "3", 10)
const outputPath = process.env.PERF_OUTPUT_PATH || path.join(projectRoot, "test-plan", `performance-probe.${phase}.json`)
const environmentMarker = process.env.PERF_ENV_MARKER || "local-playwright"
const browserProfile = process.env.PERF_BROWSER_PROFILE || "desktop-chromium"

const browserProfiles = {
  "desktop-chromium": {
    viewport: { width: 1440, height: 1000 },
    deviceScaleFactor: 1,
    isMobile: false,
    hasTouch: false,
  },
  "mobile-chromium": {
    viewport: { width: 390, height: 844 },
    deviceScaleFactor: 2,
    isMobile: true,
    hasTouch: true,
  },
}

const selectedBrowserProfile = browserProfiles[browserProfile] ?? browserProfiles["desktop-chromium"]

function nowIso() {
  return new Date().toISOString()
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

function buildTargets(todo) {
  const targets = []
  for (const route of todo.routes || []) {
    if (routeFilter.length > 0 && !routeFilter.includes(route.id) && !routeFilter.includes(route.routePattern)) {
      continue
    }
    for (const locale of route.locales || []) {
      if (localeFilter.length > 0 && !localeFilter.includes(locale)) {
        continue
      }
      targets.push({
        id: route.id,
        locale,
        routePattern: route.routePattern,
        route: instantiateRoutePattern(route.routePattern),
      })
    }
  }
  return limit > 0 ? targets.slice(0, limit) : targets
}

async function collectMetrics(page) {
  return page.evaluate(async () => {
    await new Promise((resolve) => window.requestAnimationFrame(() => window.requestAnimationFrame(resolve)))
    const navigation = performance.getEntriesByType("navigation")[0]
    const paints = performance.getEntriesByType("paint")
    const resources = performance.getEntriesByType("resource")
    const probe = window.__lunafoxPerformanceProbe ?? {}
    const byName = new Map(paints.map((entry) => [entry.name, entry.startTime]))
    const resourceSummary = resources.reduce((summary, entry) => {
      summary.count += 1
      summary.transferSize += entry.transferSize || 0
      summary.encodedBodySize += entry.encodedBodySize || 0
      summary.decodedBodySize += entry.decodedBodySize || 0
      summary.byInitiatorType[entry.initiatorType] = (summary.byInitiatorType[entry.initiatorType] || 0) + 1
      return summary
    }, {
      count: 0,
      transferSize: 0,
      encodedBodySize: 0,
      decodedBodySize: 0,
      byInitiatorType: {},
    })
    const lcpEntries = probe.largestContentfulPaintEntries ?? []
    const lastLcp = lcpEntries[lcpEntries.length - 1] ?? null
    const layoutShiftEntries = probe.layoutShiftEntries ?? []
    const cumulativeLayoutShift = layoutShiftEntries.reduce((sum, entry) => {
      return entry.hadRecentInput ? sum : sum + (entry.value || 0)
    }, 0)
    const longTaskEntries = probe.longTaskEntries ?? []
    const longTaskTotalMs = longTaskEntries.reduce((sum, entry) => sum + (entry.duration || 0), 0)

    return {
      navigationTiming: navigation
        ? {
            domContentLoadedMs: navigation.domContentLoadedEventEnd,
            loadEventMs: navigation.loadEventEnd,
            responseEndMs: navigation.responseEnd,
            transferSize: navigation.transferSize,
            encodedBodySize: navigation.encodedBodySize,
            decodedBodySize: navigation.decodedBodySize,
          }
        : null,
      paints: {
        firstPaintMs: byName.get("first-paint") ?? null,
        firstContentfulPaintMs: byName.get("first-contentful-paint") ?? null,
      },
      webVitals: {
        largestContentfulPaintMs: lastLcp?.startTime ?? null,
        largestContentfulPaintSize: lastLcp?.size ?? null,
        cumulativeLayoutShift: Number(cumulativeLayoutShift.toFixed(4)),
      },
      longTasks: {
        count: longTaskEntries.length,
        totalDurationMs: Number(longTaskTotalMs.toFixed(2)),
        maxDurationMs: Number(Math.max(0, ...longTaskEntries.map((entry) => entry.duration || 0)).toFixed(2)),
      },
      resourceSummary,
      memory: performance.memory
        ? {
            usedJSHeapSize: performance.memory.usedJSHeapSize,
            totalJSHeapSize: performance.memory.totalJSHeapSize,
            jsHeapSizeLimit: performance.memory.jsHeapSizeLimit,
          }
        : null,
      timestamp: new Date().toISOString(),
    }
  })
}

async function runOne(browser, target) {
  const context = await browser.newContext(selectedBrowserProfile)
  await primeSmokeLocaleCookie(context, baseUrl, target.locale)
  if (shouldAuth(target.route)) {
    await context.addInitScript(primeSmokeAuthSession, createSmokeAuthBootstrap())
  }

  const page = await context.newPage()
  await page.addInitScript(() => {
    window.__lunafoxPerformanceProbe = {
      largestContentfulPaintEntries: [],
      layoutShiftEntries: [],
      longTaskEntries: [],
    }

    const observe = (type, key, mapper) => {
      try {
        const observer = new PerformanceObserver((list) => {
          for (const entry of list.getEntries()) {
            window.__lunafoxPerformanceProbe[key].push(mapper(entry))
          }
        })
        observer.observe({ type, buffered: true })
      } catch {}
    }

    observe("largest-contentful-paint", "largestContentfulPaintEntries", (entry) => ({
      startTime: entry.startTime,
      renderTime: entry.renderTime,
      loadTime: entry.loadTime,
      size: entry.size,
      element: entry.element?.tagName ?? null,
      url: entry.url || null,
    }))
    observe("layout-shift", "layoutShiftEntries", (entry) => ({
      startTime: entry.startTime,
      value: entry.value,
      hadRecentInput: entry.hadRecentInput,
    }))
    observe("longtask", "longTaskEntries", (entry) => ({
      startTime: entry.startTime,
      duration: entry.duration,
      name: entry.name,
    }))
  })
  let status = 0
  let finalUrl = ""
  let metrics = null
  let error = ""

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
    } else {
      metrics = await collectMetrics(page)
    }
  } catch (caught) {
    error = summarizeError(caught)
  } finally {
    await context.close()
  }

  return {
    ...target,
    status,
    finalUrl,
    metrics,
    error,
    passed: !error,
  }
}

function averageMetric(results, selector) {
  const values = results
    .map(selector)
    .filter((value) => Number.isFinite(value))
  if (values.length === 0) {
    return null
  }
  return Number((values.reduce((sum, value) => sum + value, 0) / values.length).toFixed(2))
}

async function main() {
  const todo = await loadTodo()
  const targets = buildTargets(todo)
  if (targets.length === 0) {
    console.log("no targets for performance probe")
    return
  }

  console.log(`phase=${phase} baseUrl=${baseUrl}`)
  console.log(`targets=${targets.length} browserProfile=${browserProfile}`)

  const browser = await chromium.launch({ headless: true })
  const results = []
  for (const target of targets) {
    results.push(await runOne(browser, target))
  }
  await browser.close()

  const successful = results.filter((entry) => entry.passed)
  const failed = results.filter((entry) => !entry.passed)
  const summary = {
    generatedAt: nowIso(),
    phase,
    baseUrl,
    environmentMarker,
    browserProfile,
    routeCount: results.length,
    successCount: successful.length,
    failedCount: failed.length,
    averages: {
      domContentLoadedMs: averageMetric(successful, (entry) => entry.metrics?.navigationTiming?.domContentLoadedMs),
      loadEventMs: averageMetric(successful, (entry) => entry.metrics?.navigationTiming?.loadEventMs),
      responseEndMs: averageMetric(successful, (entry) => entry.metrics?.navigationTiming?.responseEndMs),
      firstPaintMs: averageMetric(successful, (entry) => entry.metrics?.paints?.firstPaintMs),
      firstContentfulPaintMs: averageMetric(successful, (entry) => entry.metrics?.paints?.firstContentfulPaintMs),
      largestContentfulPaintMs: averageMetric(successful, (entry) => entry.metrics?.webVitals?.largestContentfulPaintMs),
      cumulativeLayoutShift: averageMetric(successful, (entry) => entry.metrics?.webVitals?.cumulativeLayoutShift),
      longTaskCount: averageMetric(successful, (entry) => entry.metrics?.longTasks?.count),
      longTaskTotalMs: averageMetric(successful, (entry) => entry.metrics?.longTasks?.totalDurationMs),
      resourceCount: averageMetric(successful, (entry) => entry.metrics?.resourceSummary?.count),
      resourceTransferSize: averageMetric(successful, (entry) => entry.metrics?.resourceSummary?.transferSize),
      usedJSHeapSize: averageMetric(successful, (entry) => entry.metrics?.memory?.usedJSHeapSize),
    },
  }

  await fs.writeFile(outputPath, `${JSON.stringify({
    schemaVersion: 1,
    summary,
    routes: results,
  }, null, 2)}\n`, "utf8")

  console.log(`performance probe written: ${outputPath}`)
  console.log(`success=${summary.successCount} failed=${summary.failedCount}`)

  if (failed.length > 0) {
    process.exitCode = 1
  }
}

main().catch((error) => {
  console.error(error)
  process.exit(1)
})
