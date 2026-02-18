#!/usr/bin/env node

import fs from "node:fs/promises"
import path from "node:path"
import { chromium } from "@playwright/test"

const projectRoot = process.cwd()
const routesTodoPath = path.join(projectRoot, "test-plan", "routes.todo.json")
const outputPath = path.join(projectRoot, "test-plan", "interaction-matrix.json")

const baseUrl = process.env.SMOKE_BASE_URL || "http://192.168.5.7:3000"
const localeFilter = process.env.SMOKE_LOCALES
  ? process.env.SMOKE_LOCALES.split(",").map((x) => x.trim()).filter(Boolean)
  : ["zh"]

const concurrency = Number.parseInt(process.env.MATRIX_CONCURRENCY || "3", 10)
const maxElementsPerPage = Number.parseInt(process.env.MATRIX_MAX_ELEMENTS || "120", 10)

const dangerPattern = /(删除|清空|停止|注销|退出|重置|移除|销毁|delete|remove|stop|logout|sign out|clear|reset|revoke)/i

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

async function loadRoutesTodo() {
  const raw = await fs.readFile(routesTodoPath, "utf8")
  return JSON.parse(raw)
}

function buildTargets(todo) {
  const targets = []
  for (const route of todo.routes) {
    for (const locale of route.locales) {
      if (!localeFilter.includes(locale)) continue
      targets.push({
        id: route.id,
        routePattern: route.routePattern,
        locale,
        route: normalizeRoute(route.routePattern, locale),
      })
    }
  }
  return targets
}

async function collectInteractiveItems(page) {
  return page.evaluate(({ maxElements, dangerRegexSource }) => {
    const selectors = [
      "button",
      "[role='button']",
      "a",
      "input",
      "textarea",
      "select",
      "[role='tab']",
      "[role='menuitem']",
      "[role='checkbox']",
      "[role='switch']",
      "[role='radio']",
    ].join(",")

    const dangerRegex = new RegExp(dangerRegexSource, "i")

    const nodes = Array.from(document.querySelectorAll(selectors))
      .filter((el) => {
        const style = window.getComputedStyle(el)
        if (style.display === "none" || style.visibility === "hidden") return false
        const rect = el.getBoundingClientRect()
        return rect.width > 0 && rect.height > 0
      })
      .slice(0, maxElements)

    const rows = nodes.map((el) => {
      const tag = el.tagName.toLowerCase()
      const role = el.getAttribute("role") || tag
      const text = ((el.innerText || el.textContent || "").replace(/\s+/g, " ").trim()).slice(0, 120)
      const ariaLabel = (el.getAttribute("aria-label") || "").trim()
      const placeholder = (el.getAttribute("placeholder") || "").trim()
      const nameAttr = (el.getAttribute("name") || "").trim()
      const id = (el.getAttribute("id") || "").trim()
      const typeAttr = (el.getAttribute("type") || "").trim()

      const label = [ariaLabel, text, placeholder, nameAttr, id].find(Boolean) || "<unnamed>"
      const disabled = Boolean((el).disabled) || el.getAttribute("aria-disabled") === "true"
      const signature = `${tag}|${role}|${typeAttr}|${label}`
      const danger = dangerRegex.test(`${label} ${text}`)

      return {
        tag,
        role,
        type: typeAttr || undefined,
        label,
        text: text || undefined,
        disabled,
        danger,
        signature,
      }
    })

    const dedup = []
    const seen = new Set()
    for (const row of rows) {
      if (seen.has(row.signature)) continue
      seen.add(row.signature)
      dedup.push(row)
    }

    return dedup
  }, {
    maxElements: maxElementsPerPage,
    dangerRegexSource: dangerPattern.source,
  })
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
  let status = 0
  let finalUrl = ""
  let error = ""
  let items = []

  try {
    const response = await page.goto(`${baseUrl}${target.route}`, {
      waitUntil: "domcontentloaded",
      timeout: 45000,
    })
    status = response?.status() ?? 0
    await page.waitForTimeout(800)
    finalUrl = page.url()

    if (status < 500) {
      items = await collectInteractiveItems(page)
    } else {
      error = `http-${status}`
    }
  } catch (err) {
    error = summarizeError(err)
  } finally {
    await context.close()
  }

  return {
    ...target,
    status,
    finalUrl,
    error,
    items,
  }
}

async function runPool(items, worker, size) {
  const results = []
  let index = 0

  async function next() {
    const i = index
    index += 1
    if (i >= items.length) return
    const result = await worker(items[i])
    results.push(result)
    await next()
  }

  await Promise.all(Array.from({ length: Math.max(1, size) }, () => next()))
  return results
}

async function main() {
  const startedAt = nowIso()
  const todo = await loadRoutesTodo()
  const targets = buildTargets(todo)

  if (targets.length === 0) {
    console.log("no targets for interaction matrix")
    return
  }

  console.log(`baseUrl=${baseUrl}`)
  console.log(`targets=${targets.length} concurrency=${concurrency}`)

  const browser = await chromium.launch({ headless: true })
  const results = await runPool(targets, (target) => runOne(browser, target), concurrency)
  await browser.close()

  const failed = results.filter((x) => x.error)

  const summary = {
    generatedAt: nowIso(),
    startedAt,
    baseUrl,
    localeFilter,
    routeCount: results.length,
    successCount: results.length - failed.length,
    failedCount: failed.length,
    interactiveCount: results.reduce((acc, item) => acc + item.items.length, 0),
    dangerCount: results.reduce(
      (acc, item) => acc + item.items.filter((row) => row.danger).length,
      0
    ),
  }

  const payload = {
    schemaVersion: 1,
    summary,
    routes: results,
  }

  await fs.writeFile(outputPath, `${JSON.stringify(payload, null, 2)}\n`, "utf8")

  console.log(`matrix written: ${outputPath}`)
  console.log(
    `success=${summary.successCount} failed=${summary.failedCount} interactive=${summary.interactiveCount} danger=${summary.dangerCount}`
  )

  if (failed.length > 0) {
    console.log("failed samples:")
    for (const item of failed.slice(0, 10)) {
      console.log(`- ${item.route} => ${item.error}`)
    }
    process.exitCode = 1
  }
}

main().catch((error) => {
  console.error(error)
  process.exit(1)
})
