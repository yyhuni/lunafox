#!/usr/bin/env node

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

const projectRoot = process.cwd()
const baseUrl = process.env.FINGERPRINT_UI_SMOKE_BASE_URL || "http://127.0.0.1:3000"
const viewports = [
  { name: "desktop", width: 1280, height: 720, isMobile: false, hasTouch: false },
  { name: "mobile", width: 390, height: 844, isMobile: true, hasTouch: true },
]
const libraries = [
  { id: "ehole", placeholder: /搜索 CMS|Search CMS/, columns: 8, facet: true },
  { id: "goby", placeholder: /搜索名称|Search name/, columns: 5 },
  { id: "wappalyzer", placeholder: /搜索名称|Search name/, columns: 16 },
  { id: "fingers", placeholder: /搜索名称|Search name/, columns: 11, facet: true },
  { id: "fingerprinthub", placeholder: /搜索名称|Search name/, columns: 10, facet: true },
  { id: "arl", placeholder: /搜索名称|Search name/, columns: 4 },
]

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

async function openLibrary(browser, viewport, library) {
  const context = await browser.newContext({
    viewport: { width: viewport.width, height: viewport.height },
    isMobile: viewport.isMobile,
    hasTouch: viewport.hasTouch,
  })
  await primeSmokeLocaleCookie(context, baseUrl, "zh")
  await context.addInitScript(primeSmokeAuthSession, createSmokeAuthBootstrap())
  const page = await context.newPage()
  const response = await page.goto(`${baseUrl}/tools/fingerprints/${library.id}/`, { waitUntil: "domcontentloaded", timeout: 60000 })
  console.log(`loaded=${library.id}/${viewport.name} status=${response?.status()} url=${page.url()}`)
  assert(response?.ok(), `${library.id}/${viewport.name}: route response was not successful`)
  await page.locator("table:visible").first().waitFor({ state: "visible", timeout: 15000 })
  return { context, page }
}

async function verifyLibraryPage(page, viewport, library) {
  const search = page.getByPlaceholder(library.placeholder).first()
  await search.waitFor({ state: "visible", timeout: 15000 })

  const table = page.locator("table:visible").first()
  const headerCount = await table.locator("thead th").count()
  assert(headerCount >= library.columns, `${library.id}/${viewport.name}: expected at least ${library.columns} default columns, got ${headerCount}`)

  const firstDataCell = table.locator("tbody tr").first().locator("td").nth(1)
  await firstDataCell.click()
  await page.locator('[role="dialog"]').waitFor({ state: "visible", timeout: 10000 })
  await page.getByRole("button", { name: /关闭|Close/ }).click()

  if (viewport.isMobile) {
    const isHorizontallyReachable = await table.evaluate((element) => element.scrollWidth > window.innerWidth)
    assert(isHorizontallyReachable, `${library.id}/mobile: default columns are not horizontally reachable`)
  }
}

async function verifyExpandableCell(page) {
  const expander = page.getByRole("button", { name: /展开|Expand/ }).first()
  await expander.waitFor({ state: "visible", timeout: 10000 })
  await expander.click()
  await page.getByRole("button", { name: /收起|Collapse/ }).first().waitFor({ state: "visible", timeout: 5000 })
}

async function verifyFacet(page, library) {
  const trigger = page.getByRole("button", { name: /筛选|Filter/ }).first()
  await trigger.click()
  const layout = page.locator("[data-facet-layout]").first()
  await layout.waitFor({ state: "visible", timeout: 5000 })
  const option = layout.locator('[data-command-item]').first()
  await option.click()
  const apply = page.getByRole("button", { name: /^(应用|Apply)$/ }).last()
  await apply.click()
  await layout.waitFor({ state: "hidden", timeout: 5000 })
  await page.locator("table:visible").first().waitFor({ state: "visible", timeout: 5000 })
  console.log(`facet=${library.id}: ok`)
}

async function main() {
  console.log(`baseUrl=${baseUrl}`)
  const lease = await acquireSmokeDevServerLease({
    baseUrl,
    autoStartServer: process.env.FINGERPRINT_UI_SMOKE_AUTO_START !== "0",
    projectRoot,
  })
  console.log("dev-server-ready")
  let browser
  try {
    browser = await chromium.launch({ headless: true })
    console.log("browser-ready")
    for (const viewport of viewports) {
      for (const library of libraries) {
        console.log(`checking=${library.id}/${viewport.name}`)
        const { context, page } = await openLibrary(browser, viewport, library)
        try {
          await verifyLibraryPage(page, viewport, library)
          if (library.id === "goby" && viewport.name === "desktop") await verifyExpandableCell(page)
          if (library.facet) await verifyFacet(page, library)
          console.log(`${library.id}/${viewport.name}: ok`)
        } finally {
          await context.close()
        }
      }
    }
  } finally {
    await browser?.close()
    await releaseDevServerLease({ projectRoot, lease })
  }
}

main().catch((error) => {
  console.error(error)
  process.exit(1)
})
