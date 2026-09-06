#!/usr/bin/env node

import fs from "node:fs/promises"
import path from "node:path"
import { spawnSync } from "node:child_process"
import { fileURLToPath } from "node:url"
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
import {
  getBoundedGeometryExceptionValidationErrors,
  getResolvedIntrinsicTableSettlementValidationErrors,
  getRouteLoadingContract,
} from "./loading-route-contracts.mjs"
import { instantiateRoutePattern } from "./route-pattern.mjs"

const projectRoot = process.cwd()
const todoPath = path.join(projectRoot, "test-plan", "routes.todo.json")
const reportPath = path.join(projectRoot, "test-plan", "loading-handoff-smoke.json")

const smokeAuthMode = (process.env.LOADING_SMOKE_AUTH_MODE || "skip-auth").trim()
const baseUrl = process.env.LOADING_SMOKE_BASE_URL || process.env.SMOKE_BASE_URL || "http://127.0.0.1:3000"
const localeFilter = process.env.LOADING_SMOKE_LOCALES
  ? process.env.LOADING_SMOKE_LOCALES.split(",").map((item) => item.trim()).filter(Boolean)
  : process.env.SMOKE_LOCALES
    ? process.env.SMOKE_LOCALES.split(",").map((item) => item.trim()).filter(Boolean)
    : ["zh"]

const concurrency = Number.parseInt(process.env.LOADING_SMOKE_CONCURRENCY || "1", 10)
const limit = Number.parseInt(process.env.LOADING_SMOKE_LIMIT || "0", 10)
const onlyPriority = process.env.LOADING_SMOKE_PRIORITY
  ? new Set(process.env.LOADING_SMOKE_PRIORITY.split(",").map((item) => item.trim()).filter(Boolean))
  : null
const targetIdFilter = process.env.LOADING_SMOKE_TARGET_IDS
  ? new Set(process.env.LOADING_SMOKE_TARGET_IDS.split(",").map((item) => item.trim()).filter(Boolean))
  : null
const autoStartServer = process.env.LOADING_SMOKE_AUTO_START_SERVER !== "0"
const settleTimeoutMs = Number.parseInt(process.env.LOADING_SMOKE_SETTLE_TIMEOUT_MS || "8000", 10)
const navigationTimeoutMs = parsePositiveInteger(
  process.env.LOADING_SMOKE_NAVIGATION_TIMEOUT_MS,
  90_000
)
const interactionMode = process.env.LOADING_SMOKE_INTERACTIONS || "default"

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

const allowedFinalLoadingOwners = new Set([
  "initial-boot",
])

const interactionTargetDefinitions = [
  {
    id: "header-quick-scan-dialog",
    routePattern: "/overview",
    interactionName: "quick-scan-dialog",
    triggerSelector: "[data-slot='quick-scan-trigger']",
    expectedVisible: "[data-slot='quick-scan-drawer']",
    afterAction: "wait-for-sheet",
  },
  {
    id: "header-notification-drawer",
    routePattern: "/overview",
    interactionName: "notification-drawer",
    triggerSelector: "button[aria-label='通知'], button[aria-label='Notifications']",
    expectedVisible: "[data-slot='sheet-content']",
    afterAction: "wait-for-sheet",
  },
  {
    id: "distributed-architecture-dialog",
    routePattern: "/settings/agents",
    interactionName: "ArchitectureFlowCanvasSkeleton",
    triggerSelector: "button:has-text('系统架构图'), button:has-text('System Architecture')",
    expectedVisible: "[data-slot='dialog-content']",
    afterAction: "wait-for-dialog",
  },
  {
    id: "scan-workflow-config-preview",
    routePattern: "/scan/config/workflows",
    interactionName: "WorkflowConfigPreviewSkeleton",
    triggerSelector: "tr:has-text('Default Scan'), tr:has-text('Full Scan')",
    followupSelector: "button:has-text('打开编排'), button:has-text('Open Builder')",
    expectedVisible: "[data-workflow-composition-canvas]",
    afterAction: "wait-for-workflow-builder",
  },
]

function parsePositiveInteger(value, fallback) {
  const parsed = Number.parseInt(value || "", 10)
  if (!Number.isInteger(parsed) || parsed <= 0) {
    return fallback
  }
  return parsed
}

function parseTimelineSchedule(value) {
  if (!value) return [700, 1300, 2200, 4000]
  return Array.from(new Set(
    value
      .split(",")
      .map((item) => Number.parseInt(item.trim(), 10))
      .filter((item) => Number.isInteger(item) && item > 0)
  )).sort((left, right) => left - right)
}

const timelineScheduleMs = parseTimelineSchedule(process.env.LOADING_SMOKE_TIMELINE_MS)

function normalizeExpectedOwners(expectedOwners = []) {
  return expectedOwners.map((item) => {
    if (typeof item === "string") {
      return {
        owner: item,
        layer: null,
        intent: null,
      }
    }
    return {
      owner: item.owner,
      layer: item.layer ?? null,
      intent: item.intent ?? null,
    }
  })
}

export function withRouteLoadingContract(target) {
  const contractId = target.destinationContractId ?? target.id
  const contract = getRouteLoadingContract(contractId)
  if (!contract) {
    if (target.destinationContractId) {
      throw new Error(`Missing destination loading contract for ${target.id}: ${target.destinationContractId}`)
    }
    return target
  }
  return {
    ...target,
    geometry: target.geometry ?? contract.geometry,
    redirectTarget: target.redirectTarget ?? contract.redirectTarget,
    expectedOwners: normalizeExpectedOwners([
      ...(target.expectedOwners ?? []),
      ...(contract.expectedOwners ?? []),
    ]),
  }
}

function resolveViewportConfig() {
  const requestedName = (process.env.LOADING_SMOKE_VIEWPORT || "desktop").trim()
  const presetName = Object.hasOwn(viewportPresets, requestedName) ? requestedName : "desktop"
  const preset = viewportPresets[presetName]
  return {
    name: presetName,
    width: parsePositiveInteger(process.env.LOADING_SMOKE_VIEWPORT_WIDTH, preset.width),
    height: parsePositiveInteger(process.env.LOADING_SMOKE_VIEWPORT_HEIGHT, preset.height),
    isMobile: preset.isMobile,
    hasTouch: preset.hasTouch,
  }
}

const viewportConfig = resolveViewportConfig()

const loadingStructureSampleKey = "__lunafoxLoadingStructureSamples"

export function installLoadingStructureObserver() {
  const sampleKey = "__lunafoxLoadingStructureSamples"
  const maxSamples = 240
  const samples = []
  let scheduled = false
  let mutationObserver = null
  let resizeObserver = null
  const observedResizeTargets = new WeakSet()

  const toRect = (element, ownerRect, { measureIntrinsic = false } = {}) => {
    const parent = element.parentElement
    const parentStyle = parent ? window.getComputedStyle(parent) : null
    const elementStyle = window.getComputedStyle(element)
    const needsIntrinsicGridMeasurement = measureIntrinsic &&
      (parentStyle?.display === "grid" || parentStyle?.display === "inline-grid") &&
      ["auto", "normal", "stretch"].includes(elementStyle.alignSelf)

    const originalAlignSelf = element.style.getPropertyValue("align-self")
    const originalAlignSelfPriority = element.style.getPropertyPriority("align-self")
    if (needsIntrinsicGridMeasurement) {
      element.style.setProperty("align-self", "start", "important")
    }

    try {
      const rect = element.getBoundingClientRect()
      if (rect.width <= 0 || rect.height <= 0) return null

      return {
        top: Number((rect.top - ownerRect.top).toFixed(3)),
        left: Number((rect.left - ownerRect.left).toFixed(3)),
        width: Number(rect.width.toFixed(3)),
        height: Number(rect.height.toFixed(3)),
      }
    } finally {
      if (needsIntrinsicGridMeasurement) {
        if (originalAlignSelf) {
          element.style.setProperty("align-self", originalAlignSelf, originalAlignSelfPriority)
        } else {
          element.style.removeProperty("align-self")
        }
      }

      if (needsIntrinsicGridMeasurement) {
        // Ignore the probe's reversible style writes so the observer does not
        // schedule a new frame solely to measure the same grid item again.
        mutationObserver?.takeRecords()
      }
    }
  }

  const collectSlots = (owner, wrapper, ownerRect) => {
    const candidates = [wrapper, ...wrapper.querySelectorAll("[data-loading-slot]")]
    const slots = []

    for (const element of candidates) {
      if (element.closest("[data-loading-owner]") !== owner) continue
      if (element.closest("[data-loading-structure-state]") !== wrapper) continue

      const slot = element.getAttribute("data-loading-slot")
      const rect = toRect(element, ownerRect, {
        // Crossfade wrappers share one Grid track. Measure their natural block
        // size so Grid stretch cannot make unequal branches look equivalent.
        measureIntrinsic: element === wrapper && slot === "surface",
      })
      if (!slot || !rect) continue
      slots.push({ slot, ...rect })
    }

    return slots
  }

  const collect = () => {
    const owners = []

    for (const owner of document.querySelectorAll("[data-loading-owner]")) {
      const ownerRect = owner.getBoundingClientRect()
      if (ownerRect.width <= 0 || ownerRect.height <= 0) continue

      const states = {}
      for (const child of owner.children) {
        if (!(child instanceof HTMLElement)) continue
        const state = child.getAttribute("data-loading-structure-state")
        if (state !== "skeleton" && state !== "content") continue
        states[state] = collectSlots(owner, child, ownerRect)
      }

      if (Object.keys(states).length === 0) continue
      owners.push({
        owner: owner.getAttribute("data-loading-owner") || "",
        layer: owner.getAttribute("data-loading-layer"),
        phase: owner.getAttribute("data-loading-phase"),
        rect: {
          top: Number((ownerRect.top + window.scrollY).toFixed(3)),
          left: Number((ownerRect.left + window.scrollX).toFixed(3)),
          width: Number(ownerRect.width.toFixed(3)),
          height: Number(ownerRect.height.toFixed(3)),
        },
        states,
      })
    }

    const snapshot = {
      sampledAtMs: Math.round(performance.now()),
      owners,
    }
    const previous = samples[samples.length - 1]
    const fingerprint = JSON.stringify({ owners: snapshot.owners })
    if (previous?.fingerprint === fingerprint) return

    samples.push({ ...snapshot, fingerprint })
    if (samples.length > maxSamples) samples.splice(0, samples.length - maxSamples)
  }

  const schedule = () => {
    if (scheduled) return
    scheduled = true
    requestAnimationFrame(() => {
      scheduled = false
      collect()
    })
  }

  const observe = () => {
    if (!document.documentElement) return
    const observeResizeTarget = (element) => {
      if (!resizeObserver || !(element instanceof Element) || observedResizeTargets.has(element)) return
      observedResizeTargets.add(element)
      resizeObserver.observe(element)
    }
    const observeResizeTargets = () => {
      observeResizeTarget(document.documentElement)
      for (const element of document.querySelectorAll(
        "[data-loading-owner], [data-loading-structure-state], [data-loading-slot]"
      )) {
        observeResizeTarget(element)
      }
    }

    resizeObserver = typeof ResizeObserver === "undefined" ? null : new ResizeObserver(schedule)
    observeResizeTargets()
    mutationObserver = new MutationObserver(() => {
      observeResizeTargets()
      schedule()
    })
    mutationObserver.observe(document.documentElement, {
      attributes: true,
      childList: true,
      subtree: true,
    })
    window.addEventListener("resize", schedule, { passive: true })
    window.addEventListener("scroll", schedule, { passive: true, capture: true })
    schedule()
  }

  window[sampleKey] = samples
  if (document.documentElement) observe()
  else document.addEventListener("DOMContentLoaded", observe, { once: true })

  for (const delay of [0, 16, 50, 150, 300, 700, 1300, 2200, 4000]) {
    window.setTimeout(schedule, delay)
  }
}

function getTargetFilterIds(target) {
  return [target.id, ...(Array.isArray(target.filterIds) ? target.filterIds : [])]
    .filter((id) => typeof id === "string" && id.trim())
}

export function matchesTargetIdFilter(target, filter = targetIdFilter) {
  if (!filter || filter.size === 0) return true
  // Dependency metadata must not become an implicit selector: a destination
  // contract is not the same smoke target as the route that owns it.
  return getTargetFilterIds(target).some((id) => filter.has(id))
}

function shouldPrimeAuthSession(target) {
  if (smokeAuthMode !== "skip-auth") return false
  if (target.requiresAuthenticatedSession) return true
  return !/\/login\/?$/.test(target.route)
}

function nowIso() {
  return new Date().toISOString()
}

function formatElapsed(ms) {
  const seconds = Math.max(0, Math.floor(ms / 1000))
  const minutes = Math.floor(seconds / 60)
  const restSeconds = seconds % 60
  return `${String(minutes).padStart(2, "0")}:${String(restSeconds).padStart(2, "0")}`
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
      if (visited.has(childPid)) continue
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
  for (const pid of initialPids) {
    if (isPidAlive(pid)) {
      signalPid(pid, "SIGTERM")
    }
  }
  await waitMs(500)
  for (const pid of initialPids) {
    if (isPidAlive(pid)) {
      signalPid(pid, "SIGKILL")
    }
  }
}

async function loadTodo() {
  return JSON.parse(await fs.readFile(todoPath, "utf8"))
}

function buildTargets(todo) {
  const targets = []
  for (const route of todo.routes) {
    if (smokeAuthMode === "public" && route.id !== "route_login") continue
    if (smokeAuthMode === "skip-auth" && route.id === "route_login") continue
    if (onlyPriority && !onlyPriority.has(route.priority)) continue
    const locales = localeFilter.length
      ? route.locales.filter((locale) => localeFilter.includes(locale))
      : route.locales
    for (const locale of locales) {
      targets.push({
        id: route.id,
        priority: route.priority,
        routePattern: route.routePattern,
        locale,
        route: instantiateRoutePattern(route.routePattern),
      })
    }
  }
  targets.push(...buildExtraTargets())
  targets.push(...buildAuthenticatedLoginRedirectTargets())
  const filteredTargets = targets
    .map(withRouteLoadingContract)
    .filter((target) => matchesTargetIdFilter(target))
  if (limit > 0) return filteredTargets.slice(0, limit)
  return filteredTargets
}

function shouldRunInteractionTargets() {
  return interactionMode !== "0" && interactionMode !== "false" && interactionMode !== "off"
}

function buildInteractionTargets() {
  if (!shouldRunInteractionTargets()) return []
  const requestedIds = interactionMode === "default"
    ? null
    : new Set(interactionMode.split(",").map((item) => item.trim()).filter(Boolean))

  const targets = []
  for (const definition of interactionTargetDefinitions) {
    if (requestedIds && !requestedIds.has(definition.id)) continue
    for (const locale of localeFilter) {
      targets.push({
        ...definition,
        locale,
        route: instantiateRoutePattern(definition.routePattern),
      })
    }
  }
  return targets.filter((target) => matchesTargetIdFilter(target))
}

function buildExtraTargets() {
  if (smokeAuthMode !== "skip-auth") return []

  return localeFilter.map((locale) => ({
    id: "search-query-results",
    priority: "P1",
    routePattern: "/search?q=statusCode%3D%3D%22200%22",
    locale,
    route: "/search?q=statusCode%3D%3D%22200%22",
    expectedOwners: [
      {
        owner: "search-results-content",
        layer: "section",
        intent: "data",
      },
    ],
  }))
}

function buildAuthenticatedLoginRedirectTargets() {
  if (smokeAuthMode !== "skip-auth") return []

  return localeFilter.map((locale) => ({
    id: "smoke-login-authenticated-redirect",
    sourceRouteId: "route_login",
    // This is a distinct scenario that begins at the public login route and
    // settles against the real overview handoff after client navigation.
    destinationContractId: "route_overview",
    filterIds: ["route_login"],
    requiresAuthenticatedSession: true,
    priority: "P0",
    routePattern: "/login",
    route: "/login/",
    locale,
    redirectTarget: "/overview/",
  }))
}

function normalizeRedirectLocation(value) {
  const url = new URL(value, baseUrl)
  const pathname = url.pathname.replace(/\/+$/, "") || "/"
  return `${pathname}${url.search}${url.hash}`
}

/**
 * The final destination is the correctness boundary. `waitForURL` remains
 * useful diagnostic evidence, but client-side same-document redirects can
 * update the URL after its commit observer has timed out.
 *
 * @param {string} finalUrl
 * @param {{ redirectTarget?: string }} target
 * @param {{ status?: string } | null | undefined} redirect
 */
export function getRedirectTargetError(finalUrl, target, redirect = null) {
  if (!target.redirectTarget) return ""

  try {
    const finalLocation = normalizeRedirectLocation(finalUrl)
    const expectedLocation = normalizeRedirectLocation(target.redirectTarget)
    if (finalLocation === expectedLocation) return ""
    return redirect?.status === "reached"
      ? "unexpected-redirect-target"
      : "redirect-target-not-reached"
  } catch {
    return "redirect-target-contract-invalid"
  }
}

export async function waitForExpectedRedirect(page, target) {
  if (!target.redirectTarget) return null

  let expectedLocation = ""
  try {
    expectedLocation = normalizeRedirectLocation(target.redirectTarget)
  } catch (error) {
    return {
      status: "invalid-contract",
      expectedLocation: target.redirectTarget,
      finalLocation: "",
      waitedMs: 0,
      failure: error instanceof Error ? error.message : String(error),
    }
  }

  const startedAtMs = Date.now()
  try {
    await page.waitForURL(
      (url) => normalizeRedirectLocation(url.href) === expectedLocation,
      {
        timeout: settleTimeoutMs,
        waitUntil: "commit",
      }
    )
    return {
      status: "reached",
      expectedLocation,
      finalLocation: normalizeRedirectLocation(page.url()),
      waitedMs: Date.now() - startedAtMs,
    }
  } catch (error) {
    let finalLocation = ""
    try {
      finalLocation = normalizeRedirectLocation(page.url())
    } catch {
      finalLocation = page.url()
    }
    return {
      status: "not-reached",
      expectedLocation,
      finalLocation,
      waitedMs: Date.now() - startedAtMs,
      failure: error instanceof Error ? error.message : String(error),
    }
  }
}

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
      if (shuttingDown) return
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

export async function waitForLoadingSettle(page, target = {}) {
  const expectedOwners = normalizeExpectedOwners(target.expectedOwners ?? [])

  await page.waitForFunction(
    ({ allowedOwners, expectedOwners }) => {
      const owners = Array.from(document.querySelectorAll("[data-loading-owner]"))
      const allCurrentOwnersSettled = owners.every((owner) => {
        const ownerName = owner.getAttribute("data-loading-owner") || ""
        const phase = owner.getAttribute("data-loading-phase")
        if (allowedOwners.includes(ownerName)) return true
        return phase === null || phase === "content"
      })

      if (!allCurrentOwnersSettled) return false

      // A settled shell can precede a lazily mounted workspace. Do not treat
      // that shell alone as settled before the route contract owner commits.
      return expectedOwners.every((expectedOwner) => owners.some((owner) => (
        owner.getAttribute("data-loading-owner") === expectedOwner.owner &&
        owner.getAttribute("data-loading-phase") === "content" &&
        (!expectedOwner.layer || owner.getAttribute("data-loading-layer") === expectedOwner.layer)
      )))
    },
    {
      allowedOwners: Array.from(allowedFinalLoadingOwners),
      expectedOwners,
    },
    { timeout: settleTimeoutMs }
  ).catch(() => {})
}

async function waitForExpectedVisible(page, selector) {
  if (!selector) return true
  await page.locator(selector).first().waitFor({
    state: "visible",
    timeout: settleTimeoutMs,
  })
  return true
}

async function inspectLoadingDom(page, options = {}) {
  const {
    includeGeometry = false,
    includeHeading = false,
  } = options

  return page.evaluate(({ allowedOwners, includeGeometry, includeHeading }) => {
    const visible = (element) => {
      const style = window.getComputedStyle(element)
      const rect = element.getBoundingClientRect()
      return style.display !== "none" &&
        style.visibility !== "hidden" &&
        Number(style.opacity) > 0.01 &&
        rect.width > 0 &&
        rect.height > 0
    }

    const loadingOwners = Array.from(document.querySelectorAll("[data-loading-owner]")).map((element) => ({
      owner: element.getAttribute("data-loading-owner") || "",
      phase: element.getAttribute("data-loading-phase"),
      layer: element.getAttribute("data-loading-layer"),
      intent: element.getAttribute("data-loading-intent"),
      slot: element.getAttribute("data-slot"),
      bootExitStarted: element.getAttribute("data-boot-exit-started"),
      ariaHidden: element.getAttribute("aria-hidden"),
      hidden: element.hidden,
      className: typeof element.className === "string" ? element.className : "",
      opacity: Number(window.getComputedStyle(element).opacity),
      visibility: window.getComputedStyle(element).visibility,
      visible: visible(element),
      text: (element.textContent || "").replace(/\s+/g, " ").trim().slice(0, 160),
      rect: includeGeometry ? {
        top: Number(element.getBoundingClientRect().top.toFixed(2)),
        height: Number(element.getBoundingClientRect().height.toFixed(2)),
        width: Number(element.getBoundingClientRect().width.toFixed(2)),
      } : undefined,
    }))

    const visiblePageSectionSkeletons = Array.from(document.querySelectorAll("[data-slot='page-section-skeleton']")).filter(visible)
    const visibleDataSlots = Array.from(document.querySelectorAll("[data-slot]"))
      .filter(visible)
      .map((element) => ({
        slot: element.getAttribute("data-slot") || "",
        owner: element.closest("[data-loading-owner]")?.getAttribute("data-loading-owner") || "",
        layer: element.closest("[data-loading-owner]")?.getAttribute("data-loading-layer") || "",
      }))
    const visibleSkeletonSlashSeparators = Array.from(document.querySelectorAll("[data-loading-owner] span"))
      .filter((element) => {
        const slot = element.closest("[data-slot]")?.getAttribute("data-slot") || ""
        return slot.includes("skeleton") && element.textContent?.trim() === "/" && visible(element)
      })
      .map((element) => ({
        owner: element.closest("[data-loading-owner]")?.getAttribute("data-loading-owner") || "",
        slot: element.closest("[data-slot]")?.getAttribute("data-slot") || "",
      }))

    const stuckOwners = loadingOwners.filter((item) => {
      if (allowedOwners.includes(item.owner)) return false
      return item.visible && item.phase !== null && item.phase !== "content"
    })

    return {
      headingText: includeHeading ? document.querySelector("h1")?.textContent?.trim() || null : undefined,
      loadingOwners,
      stuckOwners,
      visibleDataSlots,
      visiblePageSectionSkeletonCount: visiblePageSectionSkeletons.length,
      visibleSkeletonSlashSeparators,
    }
  }, {
    allowedOwners: Array.from(allowedFinalLoadingOwners),
    includeGeometry,
    includeHeading,
  })
}

function isTransientNavigationInspectionError(error) {
  const message = error instanceof Error ? error.message : String(error)
  return message.includes("Execution context was destroyed") ||
    message.includes("Cannot find context with specified id") ||
    message.includes("most likely because of a navigation")
}

function createInspectionErrorDiagnostics(error) {
  return {
    headingText: null,
    inspectionError: error instanceof Error ? error.message : String(error),
    loadingOwners: [],
    stuckOwners: [],
    visiblePageSectionSkeletonCount: 0,
    visibleSkeletonSlashSeparators: [],
  }
}

async function inspectLoadingDomDuringTimeline(page, options = {}) {
  try {
    return await inspectLoadingDom(page, options)
  } catch (error) {
    if (!isTransientNavigationInspectionError(error)) {
      return createInspectionErrorDiagnostics(error)
    }

    await page.waitForTimeout(50)
    return inspectLoadingDom(page, options).catch(createInspectionErrorDiagnostics)
  }
}

async function captureRouteTimeline(page, target, navigationStartedAtMs) {
  if (timelineScheduleMs.length === 0) return []

  const timeline = []
  for (const requestedAtMs of timelineScheduleMs) {
    const waitForMs = requestedAtMs - (Date.now() - navigationStartedAtMs)
    if (waitForMs > 0) {
      await page.waitForTimeout(waitForMs)
    }

    const sampledAtMs = Date.now() - navigationStartedAtMs
    const diagnostics = await inspectLoadingDomDuringTimeline(page, {
      includeGeometry: true,
      includeHeading: true,
    })

    timeline.push({
      targetId: target.id,
      requestedAtMs,
      sampledAtMs,
      lateByMs: Math.max(0, sampledAtMs - requestedAtMs),
      diagnostics,
    })
  }

  return timeline
}

async function inspectLoadingStructureSamples(page) {
  return page.evaluate((sampleKey) => {
    const samples = window[sampleKey]
    if (!Array.isArray(samples)) return []
    return samples.map(({ sampledAtMs, owners }) => ({ sampledAtMs, owners }))
  }, loadingStructureSampleKey)
}

function indexLoadingStructureSlots(slots, requiredSlots) {
  const slotsByName = new Map()
  for (const slot of slots ?? []) {
    const current = slotsByName.get(slot.slot) ?? []
    current.push(slot)
    slotsByName.set(slot.slot, current)
  }

  const indexed = {}
  for (const slot of requiredSlots) {
    const matches = slotsByName.get(slot) ?? []
    if (matches.length !== 1) return null
    indexed[slot] = matches[0]
  }

  return indexed
}

function collectOwnerStructureSamples(samples, owner) {
  return samples.flatMap((sample) => {
    const ownerSample = sample.owners?.find((item) => item.owner === owner)
    return ownerSample ? [{ sample, owner: ownerSample }] : []
  })
}

function findComparableStructurePair(ownerSamples, requiredSlots) {
  const states = ownerSamples.map(({ sample, owner }) => ({
    sampledAtMs: sample.sampledAtMs,
    ownerRect: owner.rect,
    skeleton: indexLoadingStructureSlots(owner.states?.skeleton, requiredSlots),
    content: indexLoadingStructureSlots(owner.states?.content, requiredSlots),
  }))
  const overlapping = states.filter((item) => item.skeleton && item.content)
  if (overlapping.length > 0) {
    const pair = overlapping[overlapping.length - 1]
    return {
      mode: "overlap",
      skeleton: pair.skeleton,
      content: pair.content,
      owner: pair.ownerRect,
      sampledAtMs: {
        skeleton: pair.sampledAtMs,
        content: pair.sampledAtMs,
      },
    }
  }

  const skeletonSamples = states.filter((item) => item.skeleton)
  const contentSamples = states.filter((item) => item.content)
  if (skeletonSamples.length === 0 || contentSamples.length === 0) return null

  const skeleton = skeletonSamples[skeletonSamples.length - 1]
  const content = contentSamples.find((item) => item.sampledAtMs >= skeleton.sampledAtMs) ?? contentSamples[0]
  return {
    mode: "replacement",
    skeleton: skeleton.skeleton,
    content: content.content,
    owner: {
      skeleton: skeleton.ownerRect,
      content: content.ownerRect,
    },
    sampledAtMs: {
      skeleton: skeleton.sampledAtMs,
      content: content.sampledAtMs,
    },
  }
}

function findLastSkeletonToContentOnlyPairs(ownerSamples, requiredSlots) {
  const states = ownerSamples.map(({ sample, owner }) => ({
    sampledAtMs: sample.sampledAtMs,
    ownerRect: owner.rect,
    phase: owner.phase,
    skeleton: indexLoadingStructureSlots(owner.states?.skeleton, requiredSlots),
    content: indexLoadingStructureSlots(owner.states?.content, requiredSlots),
  }))
  const lastSkeletonIndex = states.reduce((lastIndex, state, index) => (
    state.skeleton ? index : lastIndex
  ), -1)
  if (lastSkeletonIndex < 0) return null

  const skeleton = states[lastSkeletonIndex]
  if (!skeleton.ownerRect) return null

  const contentOnlyStates = states.slice(lastSkeletonIndex + 1).filter((state) => (
    state.phase === "content" && !state.skeleton && state.content && state.ownerRect
  ))
  if (contentOnlyStates.length === 0) return null

  return contentOnlyStates.map((contentOnly) => ({
    mode: "content-only",
    skeleton: skeleton.skeleton,
    content: contentOnly.content,
    owner: {
      skeleton: skeleton.ownerRect,
      content: contentOnly.ownerRect,
    },
    sampledAtMs: {
      skeleton: skeleton.sampledAtMs,
      content: contentOnly.sampledAtMs,
    },
  }))
}

function getPairOwnerRects(pair) {
  if (!pair?.owner) return null

  if (pair.mode === "overlap") {
    return pair.owner ? { skeleton: pair.owner, content: pair.owner } : null
  }

  if (!pair.owner.skeleton || !pair.owner.content) return null
  return {
    skeleton: pair.owner.skeleton,
    content: pair.owner.content,
  }
}

function getRectDelta(skeleton, content) {
  return {
    top: Number(Math.abs(skeleton.top - content.top).toFixed(3)),
    left: Number(Math.abs(skeleton.left - content.left).toFixed(3)),
    width: Number(Math.abs(skeleton.width - content.width).toFixed(3)),
    height: Number(Math.abs(skeleton.height - content.height).toFixed(3)),
  }
}

function getExceededGeometryFields(delta, tolerance) {
  return Object.entries(tolerance)
    .filter(([key, value]) => delta[key] > value)
    .map(([key]) => key)
}

function getDirectionalGeometryExceededFields(skeleton, content, tolerance, options = {}) {
  const delta = getRectDelta(skeleton, content)
  return getExceededGeometryFields(delta, tolerance).filter((field) => {
    if (field === "height" && options.allowHeightShrink && content.height < skeleton.height) {
      return false
    }
    if (field === "height" && options.allowHeightGrowth && content.height > skeleton.height) {
      return false
    }
    if (field === "top" && options.allowUpwardTopShift && content.top < skeleton.top) {
      return false
    }
    if (field === "top" && options.allowDownwardTopShift && content.top > skeleton.top) {
      return false
    }
    return true
  })
}

function getResolvedIntrinsicTableSettlementOptions(settlement, slot) {
  if (!settlement) return {}

  const allowsContentGrowth = settlement.allowContentGrowth === true

  if (slot === undefined || slot === "surface" || slot === settlement.bodySlot) {
    return {
      allowHeightShrink: true,
      ...(allowsContentGrowth ? { allowHeightGrowth: true } : {}),
    }
  }

  if (slot === settlement.paginationSlot) {
    return {
      allowUpwardTopShift: true,
      ...(allowsContentGrowth ? { allowDownwardTopShift: true } : {}),
    }
  }

  return {}
}

function getResolvedIntrinsicTableSettlementPaginationError(
  settlement,
  skeletonSlots,
  contentSlots,
  tolerance
) {
  if (!settlement?.paginationSlot) return ""

  const skeletonBody = skeletonSlots[settlement.bodySlot]
  const contentBody = contentSlots[settlement.bodySlot]
  const skeletonPagination = skeletonSlots[settlement.paginationSlot]
  const contentPagination = contentSlots[settlement.paginationSlot]

  // A pagination settlement is valid only when natural table flow explains it;
  // otherwise an independently shifted footer could evade the strict contract.
  const bodyHeightDelta = contentBody.height - skeletonBody.height
  const paginationTopDelta = contentPagination.top - skeletonPagination.top
  const naturalFlowTolerance = Math.max(tolerance.top, tolerance.height)

  return Math.abs(bodyHeightDelta - paginationTopDelta) > naturalFlowTolerance
    ? "pagination-not-following-body-settlement"
    : ""
}

function getOwnerGeometryComparison(
  owner,
  pair,
  tolerance,
  measurement = "owner-document",
  settlement
) {
  const ownerRects = getPairOwnerRects(pair)
  if (!ownerRects) return null

  const delta = getRectDelta(ownerRects.skeleton, ownerRects.content)
  return {
    owner,
    slot: null,
    mode: pair.mode,
    sampledAtMs: pair.sampledAtMs,
    measurement,
    skeleton: ownerRects.skeleton,
    content: ownerRects.content,
    delta,
    tolerance,
    exceeded: getDirectionalGeometryExceededFields(
      ownerRects.skeleton,
      ownerRects.content,
      tolerance,
      getResolvedIntrinsicTableSettlementOptions(settlement)
    ),
  }
}

function getBoundedGeometryContract(geometry) {
  if (geometry.disposition !== "bounded") return { value: [], error: "" }

  if (!Array.isArray(geometry.owners) || geometry.owners.length === 0 ||
    !Array.isArray(geometry.boundedExceptions) || geometry.boundedExceptions.length === 0) {
    return { value: [], error: "bounded-contract-invalid" }
  }

  for (const exception of geometry.boundedExceptions) {
    if (getBoundedGeometryExceptionValidationErrors(exception).length > 0) {
      return { value: [], error: "bounded-contract-invalid" }
    }

    const owner = geometry.owners.find((item) => item?.owner === exception.owner)
    if (!owner || exception.affectedSlots.some((slot) => owner.requiredSlots?.includes(slot))) {
      return { value: [], error: "bounded-contract-invalid" }
    }
  }

  return { value: geometry.boundedExceptions, error: "" }
}

export function getLoadingGeometryDiagnostics(target, samples) {
  const geometry = target.geometry
  const comparisons = []

  if (!geometry || typeof geometry !== "object") {
    return { error: "loading-geometry-contract-missing", comparisons }
  }

  if (geometry.disposition === "not-applicable") {
    return { error: "", comparisons }
  }

  const boundedGeometry = getBoundedGeometryContract(geometry)
  if (boundedGeometry.error) {
    return { error: `loading-geometry-${boundedGeometry.error}`, comparisons }
  }

  if (!["verify", "bounded"].includes(geometry.disposition) ||
    !Array.isArray(geometry.owners) || geometry.owners.length === 0) {
    return { error: "loading-geometry-contract-invalid", comparisons }
  }

  for (const contract of geometry.owners) {
    const owner = contract.owner
    if (contract.preparedExternalSurface !== undefined) {
      return {
        error: `loading-geometry-prepared-surface-removed:${owner}`,
        comparisons,
      }
    }

    const settlement = contract.resolvedIntrinsicTableSettlement
    if (settlement !== undefined) {
      const settlementErrors = getResolvedIntrinsicTableSettlementValidationErrors(
        settlement,
        contract.requiredSlots
      )
      if (settlementErrors.length > 0) {
        return {
          error: `loading-geometry-${settlementErrors[0]}`,
          comparisons,
        }
      }
    }

    const ownerSamples = collectOwnerStructureSamples(samples, owner)
    if (ownerSamples.length === 0) {
      return {
        error: `loading-geometry-owner-not-observed:${owner}`,
        comparisons,
      }
    }

    const pair = findComparableStructurePair(ownerSamples, contract.requiredSlots)
    if (!pair) {
      return {
        error: `loading-geometry-pair-not-observed:${owner}`,
        comparisons,
      }
    }

    const contentOnlyPairs = findLastSkeletonToContentOnlyPairs(ownerSamples, contract.requiredSlots)
    if (!contentOnlyPairs) {
      return {
        error: `loading-geometry-content-only-pair-not-observed:${owner}`,
        comparisons,
      }
    }

    for (const contentOnlyPair of contentOnlyPairs) {
      const ownerComparison = getOwnerGeometryComparison(
        owner,
        contentOnlyPair,
        contract.tolerance,
        "owner-document",
        settlement
      )
      if (ownerComparison) {
        comparisons.push(ownerComparison)
        if (ownerComparison.exceeded.length > 0) {
          return {
            error: `loading-geometry-owner-drift:${owner}:${ownerComparison.exceeded.join(",")}`,
            comparisons,
          }
        }
      }
    }

    for (const slot of contract.requiredSlots) {
      const intrinsicSkeleton = pair.skeleton[slot]
      const intrinsicContent = pair.content[slot]
      const intrinsicDelta = getRectDelta(intrinsicSkeleton, intrinsicContent)
      const intrinsicExceeded = getDirectionalGeometryExceededFields(
        intrinsicSkeleton,
        intrinsicContent,
        contract.tolerance,
        getResolvedIntrinsicTableSettlementOptions(settlement, slot)
      )
      comparisons.push({
        owner,
        slot,
        mode: pair.mode,
        sampledAtMs: pair.sampledAtMs,
        measurement: "intrinsic",
        skeleton: intrinsicSkeleton,
        content: intrinsicContent,
        delta: intrinsicDelta,
        tolerance: contract.tolerance,
        exceeded: intrinsicExceeded,
      })

      if (intrinsicExceeded.length > 0) {
        return {
          error: `loading-geometry-drift:${owner}:${slot}:${intrinsicExceeded.join(",")}`,
          comparisons,
        }
      }
    }

    const intrinsicPaginationSettlementError = getResolvedIntrinsicTableSettlementPaginationError(
      settlement,
      pair.skeleton,
      pair.content,
      contract.tolerance
    )
    if (intrinsicPaginationSettlementError) {
      return {
        error: `loading-geometry-intrinsic-table-${intrinsicPaginationSettlementError}:${owner}`,
        comparisons,
      }
    }

    for (const contentOnlyPair of contentOnlyPairs) {
      for (const slot of contract.requiredSlots) {
        const contentOnlySkeleton = contentOnlyPair.skeleton[slot]
        const contentOnlyContent = contentOnlyPair.content[slot]
        const contentOnlyDelta = getRectDelta(contentOnlySkeleton, contentOnlyContent)
        const contentOnlyExceeded = getDirectionalGeometryExceededFields(
          contentOnlySkeleton,
          contentOnlyContent,
          contract.tolerance,
          getResolvedIntrinsicTableSettlementOptions(settlement, slot)
        )
        comparisons.push({
          owner,
          slot,
          mode: contentOnlyPair.mode,
          sampledAtMs: contentOnlyPair.sampledAtMs,
          measurement: "content-only",
          skeleton: contentOnlySkeleton,
          content: contentOnlyContent,
          delta: contentOnlyDelta,
          tolerance: contract.tolerance,
          exceeded: contentOnlyExceeded,
        })

        if (contentOnlyExceeded.length > 0) {
          return {
            error: `loading-geometry-content-only-drift:${owner}:${slot}:${contentOnlyExceeded.join(",")}`,
            comparisons,
          }
        }
      }

      const contentOnlyPaginationSettlementError = getResolvedIntrinsicTableSettlementPaginationError(
        settlement,
        contentOnlyPair.skeleton,
        contentOnlyPair.content,
        contract.tolerance
      )
      if (contentOnlyPaginationSettlementError) {
        return {
          error: `loading-geometry-content-only-intrinsic-table-${contentOnlyPaginationSettlementError}:${owner}`,
          comparisons,
        }
      }
    }
  }

  return { error: "", comparisons }
}

/**
 * Keeps the route's declared geometry contract beside the measured evidence.
 * The diagnostics alone identify a drift, but the contract is what tells a
 * reviewer which owner and slots the comparison was expected to cover.
 */
export function getLoadingGeometryReport(target, diagnostics) {
  return {
    error: diagnostics.error,
    disposition: target.geometry?.disposition ?? null,
    owners: target.geometry?.owners ?? [],
    comparisons: diagnostics.comparisons,
  }
}

export function getLoadingGeometryFailureDetails(result) {
  if (typeof result.error !== "string" || !result.error.startsWith("loading-geometry-")) {
    return null
  }

  return result.geometry ?? {
    error: result.error,
    disposition: null,
    owners: [],
    comparisons: [],
  }
}

function getLoadingDiagnosticsError(diagnostics, target) {
  if (!diagnostics) return "loading-diagnostics-missing"
  if (diagnostics.stuckOwners.length > 0) return "loading-owner-not-settled"
  if (diagnostics.loadingOwners.some((item) => item.visible && !item.layer)) {
    return "loading-owner-missing-layer"
  }
  const duplicatedControlledLayers = ["boot", "auth-shell", "app-shell", "route", "workspace"]
  for (const layer of duplicatedControlledLayers) {
    const visibleOwners = diagnostics.loadingOwners.filter((item) => (
      item.visible &&
      item.layer === layer &&
      !allowedFinalLoadingOwners.has(item.owner)
    ))
    if (visibleOwners.length > 1) {
      return "same-layer-visible-owner-overlap"
    }
  }
  if (diagnostics.visibleSkeletonSlashSeparators.length > 0) return "legacy-skeleton-slash-visible"
  if (
    diagnostics.visiblePageSectionSkeletonCount > 0 &&
    diagnostics.loadingOwners.some((item) => item.phase === "content" && item.visible)
  ) {
    return "page-section-skeleton-visible-after-content"
  }
  if (target.expectedOwners?.some((expectedOwner) => {
    if (expectedOwner.layer !== "workspace") return false
    const workspaceOwner = diagnostics.loadingOwners.find((loadingOwner) => loadingOwner.owner === expectedOwner.owner)
    const visibleRouteOwner = diagnostics.loadingOwners.some((loadingOwner) => (
      loadingOwner.visible &&
      loadingOwner.layer === "route" &&
      loadingOwner.phase === "content"
    ))
    return visibleRouteOwner && !workspaceOwner
  })) {
    return "shell-visible-workspace-empty"
  }
  if (target.expectedOwners?.some((expectedOwner) => {
    if (expectedOwner.layer !== "route") return false
    const item = diagnostics.loadingOwners.find((loadingOwner) => loadingOwner.owner === expectedOwner.owner)
    return !item || item.phase !== "content"
  })) {
    return "route-owner-not-content"
  }
  if (target.expectedOwners?.some((expectedOwner) => {
    const item = diagnostics.loadingOwners.find((loadingOwner) => loadingOwner.owner === expectedOwner.owner)
    return !item || item.phase !== "content"
  })) {
    return "expected-owner-not-content"
  }
  if (target.expectedOwners?.some((expectedOwner) => {
    if (!expectedOwner.layer) return false
    const item = diagnostics.loadingOwners.find((loadingOwner) => loadingOwner.owner === expectedOwner.owner)
    return item && item.layer !== expectedOwner.layer
  })) {
    return "expected-owner-layer-mismatch"
  }
  return ""
}

function matchesExpectedOwnerObservation(observedOwner, expectedOwner) {
  if (observedOwner?.owner !== expectedOwner.owner) return false
  return !expectedOwner.layer || observedOwner.layer === expectedOwner.layer
}

function hasExpectedOwnerSettled(diagnostics, expectedOwner) {
  const item = diagnostics?.loadingOwners?.find((loadingOwner) => (
    matchesExpectedOwnerObservation(loadingOwner, expectedOwner)
  ))
  if (!item) return false
  if (item.phase !== "content") return false
  return true
}

function hasExpectedOwnerObserved(timeline, structureSamples, expectedOwner) {
  const observedInTimeline = timeline.some((sample) => (
    sample.diagnostics?.loadingOwners?.some((item) => (
      matchesExpectedOwnerObservation(item, expectedOwner)
    ))
  ))
  if (observedInTimeline) return true

  // Structure samples come from the rendered DOM across the entire navigation.
  // They fill a coarse timeline gap only for the same declared owner and layer;
  // final settlement is still enforced by getLoadingDiagnosticsError.
  return structureSamples.some((sample) => (
    sample.owners?.some((item) => matchesExpectedOwnerObservation(item, expectedOwner))
  ))
}

export function getTimelineDiagnosticsError(timeline, target, finalDiagnostics, structureSamples = []) {
  if (!Array.isArray(timeline) || timeline.length === 0) return ""

  for (const sample of timeline) {
    const owners = sample.diagnostics?.loadingOwners ?? []
    const hasBlockingVisibleBoot = owners.some((item) => (
      item.owner === "initial-boot" &&
      item.layer === "boot" &&
      item.visible &&
      item.bootExitStarted !== "true"
    ))
    const hasVisibleNonBootOwner = owners.some((item) => (
      item.owner !== "initial-boot" &&
      item.visible &&
      item.layer &&
      item.layer !== "boot"
    ))
    if (sample.requestedAtMs >= 1300 && hasBlockingVisibleBoot && hasVisibleNonBootOwner) {
      return "boot-handoff-overlap-too-long"
    }
  }

  if (target.expectedOwners?.some((expectedOwner) => (
    hasExpectedOwnerSettled(finalDiagnostics, expectedOwner) ? false :
    !hasExpectedOwnerObserved(timeline, structureSamples, expectedOwner)
  ))) {
    return "expected-owner-not-observed-in-timeline"
  }

  return ""
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
  await context.addInitScript(installLoadingStructureObserver)
  if (shouldPrimeAuthSession(target)) {
    await context.addInitScript(primeSmokeAuthSession, createSmokeAuthBootstrap())
  }
  const page = await context.newPage()

  let error = ""
  let status = 0
  let finalUrl = ""
  let diagnostics = null
  let timeline = []
  let geometry = null
  let redirect = null

  try {
    const navigationStartedAtMs = Date.now()
    // Start before the source route navigates so a fast client redirect still
    // counts as observed rather than being inferred from the final URL.
    const redirectPromise = waitForExpectedRedirect(page, target)
    const responsePromise = page.goto(`${baseUrl}${target.route}`, {
      waitUntil: "domcontentloaded",
      timeout: navigationTimeoutMs,
    })
    timeline = await captureRouteTimeline(page, target, navigationStartedAtMs)
    const response = await responsePromise
    status = response?.status() ?? 0
    redirect = await redirectPromise
    await page.waitForLoadState("networkidle", { timeout: 10_000 }).catch(() => {})
    await waitForLoadingSettle(page, target)
    await page.waitForTimeout(250)
    finalUrl = page.url()
    diagnostics = await inspectLoadingDom(page)
    const structureSamples = await inspectLoadingStructureSamples(page)
    geometry = getLoadingGeometryReport(
      target,
      getLoadingGeometryDiagnostics(target, structureSamples)
    )

    error = status >= 500
      ? `http-${status}`
      : getRedirectTargetError(finalUrl, target, redirect) ||
        getTimelineDiagnosticsError(timeline, target, diagnostics, structureSamples) ||
        getLoadingDiagnosticsError(diagnostics, target) ||
        geometry.error
  } catch (err) {
    error = err instanceof Error ? err.message : String(err)
  } finally {
    await context.close()
  }

  return {
    ...target,
    passed: !error,
    error,
    status,
    finalUrl,
    diagnostics,
    timeline,
    geometry,
    redirect,
  }
}

async function runInteractionTarget(browser, target) {
  const context = await browser.newContext({
    viewport: {
      width: viewportConfig.width,
      height: viewportConfig.height,
    },
    isMobile: viewportConfig.isMobile,
    hasTouch: viewportConfig.hasTouch,
  })
  await primeSmokeLocaleCookie(context, baseUrl, target.locale)
  if (shouldPrimeAuthSession(target)) {
    await context.addInitScript(primeSmokeAuthSession, createSmokeAuthBootstrap())
  }
  const page = await context.newPage()

  let error = ""
  let status = 0
  let finalUrl = ""
  let immediateDiagnostics = null
  let diagnostics = null
  let expectedVisibleFound = false

  try {
    const response = await page.goto(`${baseUrl}${target.route}`, {
      waitUntil: "domcontentloaded",
      timeout: navigationTimeoutMs,
    })
    status = response?.status() ?? 0
    await page.waitForLoadState("networkidle", { timeout: 10_000 }).catch(() => {})
    await waitForLoadingSettle(page)

    if (status >= 500) {
      error = `http-${status}`
    } else {
      const trigger = page.locator(target.triggerSelector).first()
      await trigger.waitFor({ state: "visible", timeout: settleTimeoutMs })
      await trigger.click({ timeout: settleTimeoutMs })
      await page.waitForTimeout(100)
      if (target.followupSelector) {
        const followup = page.locator(target.followupSelector).first()
        await followup.waitFor({ state: "visible", timeout: settleTimeoutMs })
        await followup.click({ timeout: settleTimeoutMs })
        await page.waitForTimeout(100)
      }
      immediateDiagnostics = await inspectLoadingDom(page)
      expectedVisibleFound = await waitForExpectedVisible(page, target.expectedVisible)
      await page.waitForLoadState("networkidle", { timeout: 10_000 }).catch(() => {})
      await waitForLoadingSettle(page)
      await page.waitForTimeout(250)
      diagnostics = await inspectLoadingDom(page)
      error = getLoadingDiagnosticsError(immediateDiagnostics, target) ||
        getLoadingDiagnosticsError(diagnostics, target)
      if (!error && !expectedVisibleFound) {
        error = "expected-visible-not-found"
      }
    }

    finalUrl = page.url()
  } catch (err) {
    error = err instanceof Error ? err.message : String(err)
  } finally {
    await context.close()
  }

  return {
    ...target,
    kind: "interaction",
    passed: !error,
    error,
    status,
    finalUrl,
    afterAction: target.afterAction,
    expectedVisibleFound,
    immediateDiagnostics,
    diagnostics,
  }
}

async function runPool(items, worker, size, onResult) {
  const results = []
  let index = 0

  async function next() {
    const itemIndex = index
    index += 1
    if (itemIndex >= items.length) return
    const item = items[itemIndex]
    const result = await worker(item)
    results.push(result)
    onResult?.(result, results.length, items.length)
    await next()
  }

  await Promise.all(Array.from({ length: Math.max(1, size) }, () => next()))
  return results
}

async function main() {
  if (!["skip-auth", "public"].includes(smokeAuthMode)) {
    throw new Error(`Unsupported LOADING_SMOKE_AUTH_MODE: ${smokeAuthMode}`)
  }

  if (smokeAuthMode === "public" && autoStartServer) {
    throw new Error("Public auth loading smoke requires an externally managed NEXT_PUBLIC_SKIP_AUTH=false server; set LOADING_SMOKE_AUTO_START_SERVER=0.")
  }

  const startedAt = nowIso()
  const startedAtMs = Date.now()
  const todo = await loadTodo()
  const targets = buildTargets(todo)
  const interactionTargets = buildInteractionTargets()
  if (targets.length === 0 && interactionTargets.length === 0) {
    console.error("no loading smoke targets")
    process.exit(1)
  }

  console.log(`baseUrl=${baseUrl} authMode=${smokeAuthMode}`)
  console.log(`targets=${targets.length} interactionTargets=${interactionTargets.length} concurrency=${concurrency} viewport=${viewportConfig.name}:${viewportConfig.width}x${viewportConfig.height}`)
  console.log(`targetIds=${targetIdFilter ? Array.from(targetIdFilter).join(",") : "all"} timeline=${timelineScheduleMs.length > 0 ? timelineScheduleMs.join(",") : "off"}`)

  let browser = null
  let browserRootPid = null
  let devServerLease = null

  const closeResources = async () => {
    if (browser) {
      const activeBrowser = browser
      browser = null
      await activeBrowser.close().catch(() => {})
    }
    if (Number.isInteger(browserRootPid) && browserRootPid > 0) {
      await cleanupProcessTree(browserRootPid)
      browserRootPid = null
    }
    if (devServerLease) {
      const activeLease = devServerLease
      devServerLease = null
      await releaseDevServerLease({ projectRoot, lease: activeLease })
    }
  }

  const removeShutdownCleanup = installShutdownCleanup(closeResources)
  let results = []
  let interactionResults = []

  try {
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
    if (interactionTargets.length > 0) {
      interactionResults = await runPool(
        interactionTargets,
        (target) => runInteractionTarget(browser, target),
        1,
        (result, completed, total) => {
          const elapsed = formatElapsed(Date.now() - startedAtMs)
          const status = result.passed ? "ok" : "failed"
          console.log(`[interaction] ${completed}/${total} elapsed=${elapsed} route=${result.route} interaction=${result.interactionName} status=${status}`)
        }
      )
    }
  } finally {
    removeShutdownCleanup()
    await closeResources()
  }

  const allResults = [...results, ...interactionResults]
  const failed = allResults.filter((item) => !item.passed)
  const report = {
    schemaVersion: 1,
    startedAt,
    finishedAt: nowIso(),
    durationMs: Date.now() - startedAtMs,
    baseUrl,
    localeFilter,
    targetIdFilter: targetIdFilter ? Array.from(targetIdFilter) : null,
    timelineScheduleMs,
    viewport: viewportConfig,
    total: allResults.length,
    routeTotal: results.length,
    interactionTotal: interactionResults.length,
    passed: allResults.length - failed.length,
    failed: failed.length,
    failures: failed.map((item) => {
      const geometry = getLoadingGeometryFailureDetails(item)
      return {
        kind: item.kind || "route",
        route: item.route,
        locale: item.locale,
        interactionName: item.interactionName,
        error: item.error,
        diagnostics: item.diagnostics,
        immediateDiagnostics: item.immediateDiagnostics,
        redirect: item.redirect,
        ...(geometry ? { geometry } : {}),
      }
    }),
    routes: results,
    interactions: interactionResults,
  }

  await fs.writeFile(reportPath, `${JSON.stringify(report, null, 2)}\n`, "utf8")
  console.log(`passed=${report.passed} failed=${report.failed}`)
  if (failed.length > 0) {
    for (const item of failed.slice(0, 10)) {
      console.log(`- ${item.route} [${item.locale}] => ${item.error}`)
    }
    process.exitCode = 1
  }
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  main().catch((error) => {
    console.error(error)
    process.exit(1)
  })
}
