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
  { id: "fingerprinthub", placeholder: /搜索名称|Search name/, columns: 10, facet: true },
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

function buildVirtualRowRegressionImport() {
  return JSON.stringify(Array.from({ length: 1000 }, (_, index) => ({
    id: `virtual-row-${index + 1}`,
    info: {
      name: `Virtual row ${index + 1}`,
      severity: "info",
    },
    http: [{
      matchers: [{
        type: "word",
        words: [`virtual-row-${index + 1}-${"x".repeat(1200)}`],
      }],
    }],
  })))
}

async function getRenderedVirtualRowGeometry(page) {
  return page.locator('[data-slot="table-body"] tr').evaluateAll((elements) => elements
    .map((element, index) => {
      const rowRect = element.getBoundingClientRect()
      const contentBottom = Math.max(
        rowRect.bottom,
        ...Array.from(element.querySelectorAll("td"), (cell) => cell.getBoundingClientRect().bottom)
      )
      return {
        index,
        top: rowRect.top,
        contentBottom,
        height: rowRect.height,
      }
    })
    .filter((row) => row.height > 0)
    .sort((left, right) => left.top - right.top))
}

function findOverlappingVirtualRowPair(rows) {
  return rows.slice(1).find((row, index) => row.top < rows[index].contentBottom - 0.5)
}

async function assertVirtualRowsDoNotOverlap(page, phase) {
  await page.waitForFunction(() => {
    const rows = Array.from(document.querySelectorAll('[data-slot="table-body"] tr'))
      .map((element) => {
        const rowRect = element.getBoundingClientRect()
        const contentBottom = Math.max(
          rowRect.bottom,
          ...Array.from(element.querySelectorAll("td"), (cell) => cell.getBoundingClientRect().bottom)
        )
        return { top: rowRect.top, contentBottom, height: rowRect.height }
      })
      .filter((row) => row.height > 0)
      .sort((left, right) => left.top - right.top)

    return rows.length >= 2 && rows.slice(1).every((row, index) => row.top >= rows[index].contentBottom - 0.5)
  }, undefined, { timeout: 10000 })

  const rows = await getRenderedVirtualRowGeometry(page)
  const overlap = findOverlappingVirtualRowPair(rows)
  assert(!overlap, `${phase}: virtual row content overlaps the following row`)
  console.log(`fingerprinthub/virtual-rows/${phase}: ok`)
}

async function verifyFingerprintHubVirtualRows(page) {
  await page.getByRole("button", { name: /^(文件导入|导入文件|Import File)$/ }).click()
  const dialog = page.getByRole("dialog")
  const fileInput = dialog.locator('input[type="file"]')
  await fileInput.setInputFiles({
    name: "virtual-row-regression.json",
    mimeType: "application/json",
    buffer: Buffer.from(buildVirtualRowRegressionImport()),
  })
  await dialog.getByRole("button", { name: /^(导入|Import)$/ }).click()
  await dialog.waitFor({ state: "hidden", timeout: 15000 })

  const pageSize = page.getByRole("combobox", { name: /每页显示|Rows per page/ })
  await pageSize.click()
  await page.getByRole("option", { name: "1000", exact: true }).click()
  await page.waitForFunction(() => {
    const tableBody = document.querySelector('[data-slot="table-body"]')
    return tableBody instanceof HTMLElement
      && Number.parseFloat(tableBody.style.height) > 0
      && tableBody.querySelectorAll("tr").length > 1
  }, undefined, { timeout: 15000 })

  await assertVirtualRowsDoNotOverlap(page, "collapsed")

  await page.getByRole("button", { name: /展开|Expand/ }).first().click()
  await page.getByRole("button", { name: /收起|Collapse/ }).first().waitFor({ state: "visible", timeout: 5000 })
  await assertVirtualRowsDoNotOverlap(page, "expanded")
}

async function verifyFacet(page, library) {
  const trigger = page.getByRole("button", { name: /严重程度|Severity/ }).first()
  await trigger.click()
  const content = page.locator('[data-slot="popover-content"]').last()
  await content.waitFor({ state: "visible", timeout: 5000 })
  const option = content.locator('[data-command-item]').first()
  await option.click()
  const apply = content.getByRole("button", { name: /^(应用|Apply)$/ })
  await apply.click()
  await content.waitFor({ state: "hidden", timeout: 5000 })
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
          if (library.facet) await verifyFacet(page, library)
          if (library.id === "fingerprinthub" && viewport.name === "desktop") await verifyFingerprintHubVirtualRows(page)
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
