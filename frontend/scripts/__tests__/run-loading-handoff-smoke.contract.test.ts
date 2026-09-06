import { chromium } from "@playwright/test"
import { describe, expect, it } from "vitest"
import { existsSync, readFileSync } from "node:fs"
import path from "node:path"

import {
  getLoadingGeometryFailureDetails,
  getLoadingGeometryDiagnostics,
  getLoadingGeometryReport,
  getRedirectTargetError,
  getTimelineDiagnosticsError,
  installLoadingStructureObserver,
  matchesTargetIdFilter,
  waitForLoadingSettle,
  waitForExpectedRedirect,
  withRouteLoadingContract,
} from "../run-loading-handoff-smoke.mjs"
import { getRouteLoadingContract } from "../loading-route-contracts.mjs"

const sourcePath = path.resolve(process.cwd(), "scripts/run-loading-handoff-smoke.mjs")
const source = readFileSync(sourcePath, "utf8")
const devServerSource = readFileSync(path.resolve(process.cwd(), "scripts/smoke-dev-server.mjs"), "utf8")
const packageJson = JSON.parse(
  readFileSync(path.resolve(process.cwd(), "package.json"), "utf8")
) as { scripts: Record<string, string> }

type BrowserStructureOwner = {
  owner: string
  phase?: string
  rect?: { top: number; left: number; width: number; height: number }
  states: Record<string, unknown>
}

type BrowserStructureSample = {
  sampledAtMs: number
  owners: BrowserStructureOwner[]
}

type BrowserStructureWindow = Window & typeof globalThis & {
  __lunafoxLoadingStructureSamples?: BrowserStructureSample[]
}

const geometryTolerance = {
  top: 1,
  left: 1,
  width: 1,
  height: 2,
}

const pairedGeometryTarget = {
  geometry: {
    disposition: "verify",
    owners: [
      {
        owner: "route-content",
        requiredSlots: ["surface"],
        tolerance: geometryTolerance,
      },
    ],
  },
}

function pairedStructureSample(owner = "route-content") {
  const surface = {
    slot: "surface",
    top: 0,
    left: 0,
    width: 100,
    height: 200,
  }

  return [
    {
      sampledAtMs: 1,
      owners: [
        {
          owner,
          phase: "handoff",
          rect: { top: 120, left: 24, width: 100, height: 200 },
          states: {
            skeleton: [surface],
            content: [surface],
          },
        },
      ],
    },
    {
      sampledAtMs: 2,
      owners: [
        {
          owner,
          phase: "content",
          rect: { top: 120, left: 24, width: 100, height: 200 },
          states: { content: [surface] },
        },
      ],
    },
  ]
}

const intrinsicTableSettlementTarget = {
  geometry: {
    disposition: "verify",
    owners: [
      {
        owner: "table-content",
        requiredSlots: ["surface", "toolbar", "body", "pagination"],
        tolerance: geometryTolerance,
        resolvedIntrinsicTableSettlement: {
          bodySlot: "body",
          paginationSlot: "pagination",
          reason: "The shared table releases its loading-only row reservation after sparse data resolves.",
          reviewTrigger: "When the table or pagination layout changes.",
        },
      },
    ],
  },
}

const contentGrowthTableSettlementTarget = {
  geometry: {
    ...intrinsicTableSettlementTarget.geometry,
    owners: [{
      ...intrinsicTableSettlementTarget.geometry.owners[0],
      resolvedIntrinsicTableSettlement: {
        ...intrinsicTableSettlementTarget.geometry.owners[0].resolvedIntrinsicTableSettlement,
        allowContentGrowth: true,
        reason: "Complete Badge groups may wrap after the table resolves.",
      },
    }],
  },
}

function intrinsicTableSettlementSamples({
  ownerHeight = 300,
  surfaceHeight = 300,
  toolbarHeight = 40,
  bodyHeight = 140,
  paginationTop = 220,
}: {
  ownerHeight?: number
  surfaceHeight?: number
  toolbarHeight?: number
  bodyHeight?: number
  paginationTop?: number
} = {}) {
  return [{
    sampledAtMs: 1,
    owners: [{
      owner: "table-content",
      phase: "loading",
      rect: { top: 120, left: 24, width: 600, height: 500 },
      states: {
        skeleton: [
          { slot: "surface", top: 0, left: 0, width: 600, height: 500 },
          { slot: "toolbar", top: 0, left: 0, width: 600, height: 40 },
          { slot: "body", top: 48, left: 0, width: 600, height: 340 },
          { slot: "pagination", top: 420, left: 0, width: 600, height: 32 },
        ],
      },
    }],
  }, {
    sampledAtMs: 2,
    owners: [{
      owner: "table-content",
      phase: "content",
      rect: { top: 120, left: 24, width: 600, height: ownerHeight },
      states: {
        content: [
          { slot: "surface", top: 0, left: 0, width: 600, height: surfaceHeight },
          { slot: "toolbar", top: 0, left: 0, width: 600, height: toolbarHeight },
          { slot: "body", top: 48, left: 0, width: 600, height: bodyHeight },
          { slot: "pagination", top: paginationTop, left: 0, width: 600, height: 32 },
        ],
      },
    }],
  }]
}

describe("run-loading-handoff-smoke contract", () => {
  it("exists and runs through the package script", () => {
    expect(existsSync(sourcePath)).toBe(true)
    expect(packageJson.scripts["test:e2e:loading"]).toBe(
      "LOADING_SMOKE_PRIORITY=P0,P1,P2 LOADING_SMOKE_CONCURRENCY=2 LOADING_SMOKE_INTERACTIONS=off node scripts/run-loading-handoff-smoke.mjs"
    )
    expect(packageJson.scripts["test:e2e:loading:mobile"]).toBe(
      "LOADING_SMOKE_PRIORITY=P0,P1,P2 LOADING_SMOKE_CONCURRENCY=2 LOADING_SMOKE_INTERACTIONS=off LOADING_SMOKE_VIEWPORT=mobile node scripts/run-loading-handoff-smoke.mjs"
    )
  })

  it("uses shared route inventory and smoke auth bootstrap", () => {
    expect(source).toContain("routes.todo.json")
    expect(source).toContain("from \"../lib/auth-runtime.mjs\"")
    expect(source).toContain("from \"./smoke-dev-server.mjs\"")
    expect(source).toContain("from \"./loading-route-contracts.mjs\"")
    expect(source).toContain('from "./route-pattern.mjs"')
    expect(source).toContain("getRouteLoadingContract")
    expect(source).toContain("instantiateRoutePattern(route.routePattern)")
    expect(source).toContain("instantiateRoutePattern(definition.routePattern)")
    expect(source).not.toContain("function normalizeRoute")
    expect(source).toContain("primeSmokeAuthSession")
    expect(source).toContain("LOADING_SMOKE_AUTH_MODE")
    expect(source).toContain("smoke-login-authenticated-redirect")
    expect(source).toContain("destinationContractId")
    expect(source).toContain("requiresAuthenticatedSession")
    expect(source).toContain("redirectTarget")
    expect(source).toContain("redirect-target-not-reached")
    expect(source).toContain("waitForExpectedRedirect")
    expect(source).toContain("waitForURL")
    expect(source).toContain("timeout: settleTimeoutMs")
    expect(source).toContain("primeSmokeLocaleCookie")
    expect(source).toContain("acquireSmokeDevServerLease")
    expect(source).toContain("releaseDevServerLease")
    expect(devServerSource).toContain("NEXT_PUBLIC_SKIP_AUTH: \"true\"")
    expect(devServerSource).toContain("NEXT_PUBLIC_USE_MOCK")
    expect(devServerSource).toContain("smoke-dev-server-state.json")
    expect(devServerSource).toContain("leases")
    expect(source).not.toContain("function startDevServer")
  })

  it("supports locale and viewport coverage controls", () => {
    expect(source).toContain("LOADING_SMOKE_LOCALES")
    expect(source).toContain("LOADING_SMOKE_VIEWPORT")
    expect(source).toContain("LOADING_SMOKE_VIEWPORT_WIDTH")
    expect(source).toContain("LOADING_SMOKE_VIEWPORT_HEIGHT")
    expect(source).toContain("LOADING_SMOKE_TARGET_IDS")
    expect(source).toContain("LOADING_SMOKE_TIMELINE_MS")
    expect(source).toContain("LOADING_SMOKE_NAVIGATION_TIMEOUT_MS")
    expect(source).toContain("const navigationTimeoutMs = parsePositiveInteger(")
    expect(source).toContain("90_000")
    expect(source.match(/timeout: navigationTimeoutMs/g)).toHaveLength(2)
    expect(source).toContain("viewportConfig")
    expect(source).toContain("browser.newContext({")
    expect(source).toContain("viewport:")
    expect(source).toContain("isMobile: viewportConfig.isMobile")
    expect(source).toContain("hasTouch: viewportConfig.hasTouch")
    expect(source).toContain("viewport=${viewportConfig.name}")
  })

  it("keeps authenticated login redirect selection separate from the overview route contract", () => {
    const redirectTarget = {
      id: "smoke-login-authenticated-redirect",
      sourceRouteId: "route_login",
      destinationContractId: "route_overview",
      filterIds: ["route_login"],
      redirectTarget: "/overview/",
    }

    expect(matchesTargetIdFilter(redirectTarget, new Set(["route_overview"]))).toBe(false)
    expect(matchesTargetIdFilter(redirectTarget, new Set(["route_login"]))).toBe(true)
    expect(matchesTargetIdFilter(redirectTarget, new Set(["smoke-login-authenticated-redirect"]))).toBe(true)
    expect(source).toContain(".filter((target) => matchesTargetIdFilter(target))")
    expect(source).not.toContain(".filter(matchesTargetIdFilter)")

    expect(withRouteLoadingContract({ id: "route_login" })).toMatchObject({
      id: "route_login",
      expectedOwners: [{ owner: "login-page-content", layer: "route" }],
      geometry: { disposition: "not-applicable" },
    })
    expect(withRouteLoadingContract(redirectTarget)).toMatchObject({
      id: "smoke-login-authenticated-redirect",
      destinationContractId: "route_overview",
      expectedOwners: expect.arrayContaining([
        expect.objectContaining({ owner: "overview-sections-loader", layer: "section" }),
      ]),
    })
  })

  it("waits for an observed redirect commit and reports a distinct no-arrival failure", async () => {
    let receivedOptions: { timeout?: number; waitUntil?: string } | undefined
    const reachedPage = {
      waitForURL: async (
        predicate: (url: URL) => boolean,
        options: { timeout?: number; waitUntil?: string }
      ) => {
        receivedOptions = options
        expect(predicate(new URL("http://localhost:3000/overview/"))).toBe(true)
      },
      url: () => "http://localhost:3000/overview/",
    }
    const redirectTarget = { redirectTarget: "/overview/" }

    await expect(waitForExpectedRedirect(reachedPage, redirectTarget)).resolves.toMatchObject({
      status: "reached",
      expectedLocation: "/overview",
      finalLocation: "/overview",
    })
    expect(receivedOptions).toMatchObject({
      timeout: expect.any(Number),
      waitUntil: "commit",
    })

    const notReachedPage = {
      waitForURL: async () => {
        throw new Error("Timeout waiting for redirect")
      },
      url: () => "http://localhost:3000/login/",
    }
    const notReached = await waitForExpectedRedirect(notReachedPage, redirectTarget)

    expect(notReached).toMatchObject({
      status: "not-reached",
      expectedLocation: "/overview",
      finalLocation: "/login",
      failure: "Timeout waiting for redirect",
    })
    expect(getRedirectTargetError("http://localhost:3000/login/", redirectTarget, notReached)).toBe(
      "redirect-target-not-reached"
    )
    expect(getRedirectTargetError("http://localhost:3000/overview/", redirectTarget, {
      status: "reached",
    })).toBe("")
    expect(getRedirectTargetError("http://localhost:3000/targets/", redirectTarget, {
      status: "reached",
    })).toBe("unexpected-redirect-target")
    expect(getRedirectTargetError("http://localhost:3000/overview/", redirectTarget, {
      status: "not-reached",
    })).toBe("")
  })

  it("checks loading owners, final phases, and legacy visible skeletons", () => {
    expect(source).toContain("[data-loading-owner]")
    expect(source).toContain("[data-slot]")
    expect(source).toContain("data-loading-phase")
    expect(source).toContain("data-loading-layer")
    expect(source).toContain("data-loading-intent")
    expect(source).toContain("visibleDataSlots")
    expect(source).toContain("page-section-skeleton")
    expect(source).toContain("slot.includes(\"skeleton\")")
    expect(source).toContain("legacy-skeleton-slash-visible")
    expect(source).toContain("loading-owner-not-settled")
    expect(source).toContain("loading-owner-missing-layer")
    expect(source).toContain("same-layer-visible-owner-overlap")
    expect(source).toContain("shell-visible-workspace-empty")
    expect(source).toContain("route-owner-not-content")
    expect(source).toContain("page-section-skeleton-visible-after-content")
  })

  it("captures paired loading structure slots from document start and compares geometry", () => {
    expect(source).toContain("function installLoadingStructureObserver")
    expect(source).toContain("__lunafoxLoadingStructureSamples")
    expect(source).toContain("data-loading-structure-state")
    expect(source).toContain("[data-loading-slot]")
    expect(source).toContain("getLoadingGeometryDiagnostics")
    expect(source).toContain("findComparableStructurePair")
    expect(source).toContain("loading-geometry-owner-not-observed")
    expect(source).toContain("loading-geometry-pair-not-observed")
    expect(source).toContain("loading-geometry-content-only-pair-not-observed")
    expect(source).toContain("loading-geometry-drift")
    expect(source).toContain("loading-geometry-content-only-drift")
    expect(source).toContain("loading-geometry-owner-drift")
    expect(source).toContain("findLastSkeletonToContentOnlyPairs")
    expect(source).toContain("getOwnerGeometryComparison")
    expect(source).toContain('"owner-document"')
    expect(source).toContain("ownerRect.top + window.scrollY")
    expect(source).toContain("ownerRect.left + window.scrollX")
    expect(source).toContain("ResizeObserver")
    expect(source).toContain("[data-loading-owner], [data-loading-structure-state], [data-loading-slot]")
    expect(source).toContain("boundedExceptions")
    expect(source).toContain("bounded-contract-invalid")
    expect(source).toContain("resolvedIntrinsicTableSettlement")
    expect(source).toContain("getDirectionalGeometryExceededFields")
    expect(source).toContain("await context.addInitScript(installLoadingStructureObserver)")
  })

  it("checks only declared paired geometry owners and fails missing evidence", () => {
    expect(source).toContain('geometry.disposition === "not-applicable"')
    expect(source).toContain("for (const contract of geometry.owners)")
    expect(source).toContain("if (ownerSamples.length === 0)")
    expect(source).toContain("loading-geometry-owner-not-observed:${owner}")
    expect(source).toContain("loading-geometry-pair-not-observed:${owner}")
    expect(source).not.toContain("observedOwnerNames")
    expect(source).toContain("samples.map(({ sampledAtMs, owners }) => ({ sampledAtMs, owners }))")
  })

  it("fails a declared verify owner that the structure observer never sees", () => {
    expect(getLoadingGeometryDiagnostics(pairedGeometryTarget, [])).toEqual({
      error: "loading-geometry-owner-not-observed:route-content",
      comparisons: [],
    })
  })

  it("fails a declared verify owner that never supplies both structure states", () => {
    const samples = [
      {
        sampledAtMs: 1,
        owners: [
          {
            owner: "route-content",
            states: {
              skeleton: [{
                slot: "surface",
                top: 0,
                left: 0,
                width: 100,
                height: 200,
              }],
            },
          },
        ],
      },
    ]

    expect(getLoadingGeometryDiagnostics(pairedGeometryTarget, samples)).toEqual({
      error: "loading-geometry-pair-not-observed:route-content",
      comparisons: [],
    })
  })

  it("fails an overlap-only handoff that never reaches a content-only sample", () => {
    const surface = { slot: "surface", top: 0, left: 0, width: 100, height: 200 }
    const samples = [{
      sampledAtMs: 1,
      owners: [{
        owner: "route-content",
        phase: "handoff",
        rect: { top: 120, left: 24, width: 100, height: 200 },
        states: {
          skeleton: [surface],
          content: [surface],
        },
      }],
    }]

    expect(getLoadingGeometryDiagnostics(pairedGeometryTarget, samples)).toEqual({
      error: "loading-geometry-content-only-pair-not-observed:route-content",
      comparisons: [],
    })
  })

  it("fails a replacement handoff when the owner moves even though its slots stay relative", () => {
    const surface = { slot: "surface", top: 0, left: 0, width: 100, height: 200 }
    const samples = [{
      sampledAtMs: 1,
      owners: [{
        owner: "route-content",
        phase: "loading",
        rect: { top: 120, left: 24, width: 100, height: 200 },
        states: { skeleton: [surface] },
      }],
    }, {
      sampledAtMs: 2,
      owners: [{
        owner: "route-content",
        phase: "content",
        rect: { top: 152, left: 24, width: 100, height: 200 },
        states: { content: [surface] },
      }],
    }]

    expect(getLoadingGeometryDiagnostics(pairedGeometryTarget, samples)).toMatchObject({
      error: "loading-geometry-owner-drift:route-content:top",
      comparisons: [{
        measurement: "owner-document",
        slot: null,
        delta: { top: 32, left: 0, width: 0, height: 0 },
        exceeded: ["top"],
      }],
    })
  })

  it("allows only the declared shared table to settle shorter with natural-flow pagination", () => {
    expect(getLoadingGeometryDiagnostics(
      intrinsicTableSettlementTarget,
      intrinsicTableSettlementSamples()
    )).toMatchObject({
      error: "",
      comparisons: expect.arrayContaining([
        expect.objectContaining({ measurement: "owner-document", exceeded: [] }),
        expect.objectContaining({ measurement: "content-only", slot: "body", exceeded: [] }),
        expect.objectContaining({ measurement: "content-only", slot: "pagination", exceeded: [] }),
      ]),
    })
  })

  it("rejects reverse or undeclared drift for an intrinsic shared table settlement", () => {
    expect(getLoadingGeometryDiagnostics(
      intrinsicTableSettlementTarget,
      intrinsicTableSettlementSamples({ ownerHeight: 560, surfaceHeight: 560, bodyHeight: 400 })
    ).error).toBe("loading-geometry-owner-drift:table-content:height")

    expect(getLoadingGeometryDiagnostics(
      intrinsicTableSettlementTarget,
      intrinsicTableSettlementSamples({ paginationTop: 460 })
    ).error).toBe("loading-geometry-drift:table-content:pagination:top")

    expect(getLoadingGeometryDiagnostics(
      intrinsicTableSettlementTarget,
      intrinsicTableSettlementSamples({ toolbarHeight: 80 })
    ).error).toBe("loading-geometry-drift:table-content:toolbar:height")
  })

  it("allows only a declared shared table to settle taller with natural-flow pagination", () => {
    expect(getLoadingGeometryDiagnostics(
      contentGrowthTableSettlementTarget,
      intrinsicTableSettlementSamples({
        ownerHeight: 560,
        surfaceHeight: 560,
        bodyHeight: 400,
        paginationTop: 480,
      })
    )).toMatchObject({
      error: "",
      comparisons: expect.arrayContaining([
        expect.objectContaining({ measurement: "owner-document", exceeded: [] }),
        expect.objectContaining({ measurement: "content-only", slot: "body", exceeded: [] }),
        expect.objectContaining({ measurement: "content-only", slot: "pagination", exceeded: [] }),
      ]),
    })
  })

  it("rejects a pagination shift that is not caused by the declared body settlement", () => {
    expect(getLoadingGeometryDiagnostics(
      intrinsicTableSettlementTarget,
      intrinsicTableSettlementSamples({
        ownerHeight: 500,
        surfaceHeight: 500,
        bodyHeight: 340,
        paginationTop: 220,
      })
    ).error).toBe("loading-geometry-intrinsic-table-pagination-not-following-body-settlement:table-content")
  })

  it("rejects a declared growth settlement when pagination does not follow the body", () => {
    expect(getLoadingGeometryDiagnostics(
      contentGrowthTableSettlementTarget,
      intrinsicTableSettlementSamples({
        ownerHeight: 560,
        surfaceHeight: 560,
        bodyHeight: 400,
        paginationTop: 420,
      })
    ).error).toBe("loading-geometry-intrinsic-table-pagination-not-following-body-settlement:table-content")
  })

  it("preserves geometry contract and comparison evidence in geometry failure reports", () => {
    const comparisons = [{
      owner: "route-content",
      slot: "surface",
      mode: "overlap",
      skeleton: { top: 0, left: 0, width: 100, height: 200 },
      content: { top: 0, left: 0, width: 100, height: 400 },
      delta: { top: 0, left: 0, width: 0, height: 200 },
      tolerance: geometryTolerance,
      exceeded: ["height"],
    }]
    const error = "loading-geometry-drift:route-content:surface:height"
    const geometry = getLoadingGeometryReport(pairedGeometryTarget, { error, comparisons })

    expect(geometry).toEqual({
      error,
      disposition: "verify",
      owners: pairedGeometryTarget.geometry.owners,
      comparisons,
    })
    expect(getLoadingGeometryFailureDetails({ error, geometry })).toEqual(geometry)
    expect(getLoadingGeometryFailureDetails({
      error: "loading-owner-not-settled",
      geometry,
    })).toBeNull()
    expect(source).toContain("geometry = getLoadingGeometryReport(")
    expect(source).toContain("getLoadingGeometryFailureDetails(item)")
    expect(source).toContain("...(geometry ? { geometry } : {})")
  })

  it("measures a crossfade surface before Grid stretch can hide intrinsic height drift", async () => {
    const browser = await chromium.launch({ headless: true })

    try {
      const page = await browser.newPage()
      await page.addInitScript(installLoadingStructureObserver)
      await page.goto(`data:text/html,${encodeURIComponent(`
        <style>
          [data-loading-owner="grid-surface-probe"] {
            display: grid;
            width: 600px;
          }

          [data-loading-owner="grid-surface-probe"] > [data-loading-structure-state] {
            grid-area: 1 / 1;
          }
        </style>
        <div data-loading-owner="grid-surface-probe" data-loading-phase="handoff">
          <div id="grid-skeleton" data-loading-structure-state="skeleton" data-loading-slot="surface">
            <div style="height: 200px"></div>
          </div>
          <div data-loading-structure-state="content" data-loading-slot="surface">
            <div style="height: 400px"></div>
          </div>
        </div>
        <script>
          setTimeout(() => {
            const owner = document.querySelector('[data-loading-owner="grid-surface-probe"]')
            owner.setAttribute('data-loading-phase', 'content')
            document.querySelector('#grid-skeleton').remove()
          }, 80)
        </script>
      `)}`)

      await page.waitForFunction(() => {
        const samples = (window as BrowserStructureWindow).__lunafoxLoadingStructureSamples
        return Array.isArray(samples) && samples.some((sample) => (
          sample.owners.some((owner) => (
            owner.owner === "grid-surface-probe" && owner.phase === "content" && !owner.states.skeleton && owner.states.content
          ))
        ))
      })

      const samples = await page.evaluate(() => {
        const structureSamples = (window as BrowserStructureWindow).__lunafoxLoadingStructureSamples ?? []
        return structureSamples.map(({ sampledAtMs, owners }) => ({ sampledAtMs, owners }))
      })
      const geometry = getLoadingGeometryDiagnostics({
        geometry: {
          disposition: "verify",
          owners: [{
            owner: "grid-surface-probe",
            requiredSlots: ["surface"],
            tolerance: geometryTolerance,
          }],
        },
      }, samples)

      expect(geometry.error).toBe("loading-geometry-drift:grid-surface-probe:surface:height")
      expect(geometry.comparisons.find((comparison) => comparison.measurement === "intrinsic")).toMatchObject({
        skeleton: { height: 200 },
        content: { height: 400 },
        delta: { height: 200 },
      })
    } finally {
      await browser.close()
    }
  }, 15_000)

  it("waits for a late expected workspace owner instead of settling on shell content alone", async () => {
    const browser = await chromium.launch({ headless: true })

    try {
      const page = await browser.newPage()
      await page.setContent(`
        <div
          data-loading-owner="auth-layout-route-content"
          data-loading-layer="route"
          data-loading-phase="content"
        ></div>
      `)

      let settled = false
      const settling = waitForLoadingSettle(page, {
        expectedOwners: [{ owner: "late-workspace-content", layer: "workspace" }],
      }).then(() => {
        settled = true
      })

      await page.waitForTimeout(100)
      expect(settled).toBe(false)

      await page.evaluate(() => {
        const owner = document.createElement("div")
        owner.setAttribute("data-loading-owner", "late-workspace-content")
        owner.setAttribute("data-loading-layer", "workspace")
        owner.setAttribute("data-loading-phase", "content")
        document.body.append(owner)
      })

      await settling
      expect(settled).toBe(true)
    } finally {
      await browser.close()
    }
  }, 15_000)

  it("rejects the removed prepared external surface field instead of treating it as a geometry waiver", () => {
    expect(getLoadingGeometryDiagnostics({
      geometry: {
        disposition: "verify",
        owners: [{
          owner: "route-content",
          requiredSlots: ["surface"],
          tolerance: geometryTolerance,
          preparedExternalSurface: {},
        }],
      },
    }, pairedStructureSample())).toEqual({
      error: "loading-geometry-prepared-surface-removed:route-content",
      comparisons: [],
    })
  })

  it("compares the last skeleton and first content-only owner document rect plus inset slots", () => {
    const surface = { slot: "surface", top: 10, left: 10, width: 80, height: 400 }
    const samples = [{
      sampledAtMs: 1,
      owners: [{
        owner: "inset-owner",
        phase: "handoff",
        rect: { top: 240, left: 48, width: 100, height: 420 },
        states: { skeleton: [surface], content: [surface] },
      }],
    }, {
      sampledAtMs: 2,
      owners: [{
        owner: "inset-owner",
        phase: "content",
        rect: { top: 240, left: 48, width: 100, height: 420 },
        states: { content: [surface] },
      }],
    }]

    expect(getLoadingGeometryDiagnostics({
      geometry: {
        disposition: "verify",
        owners: [{ owner: "inset-owner", requiredSlots: ["surface"], tolerance: geometryTolerance }],
      },
    }, samples)).toMatchObject({
      error: "",
      comparisons: expect.arrayContaining([
        expect.objectContaining({
          measurement: "owner-document",
          skeleton: { top: 240, left: 48, width: 100, height: 420 },
          content: { top: 240, left: 48, width: 100, height: 420 },
        }),
        expect.objectContaining({ measurement: "content-only", slot: "surface" }),
      ]),
    })
  })

  it("fails a later content-only slot resize even when the owner outer rect stays stable", () => {
    const samples = [{
      sampledAtMs: 1,
      owners: [{
        owner: "late-slot-drift",
        phase: "handoff",
        rect: { top: 80, left: 20, width: 600, height: 420 },
        states: {
          skeleton: [{ slot: "surface", top: 0, left: 0, width: 600, height: 400 }],
          content: [{ slot: "surface", top: 0, left: 0, width: 600, height: 400 }],
        },
      }],
    }, {
      sampledAtMs: 2,
      owners: [{
        owner: "late-slot-drift",
        phase: "content",
        rect: { top: 80, left: 20, width: 600, height: 420 },
        states: {
          content: [{ slot: "surface", top: 0, left: 0, width: 600, height: 400 }],
        },
      }],
    }, {
      sampledAtMs: 3,
      owners: [{
        owner: "late-slot-drift",
        phase: "content",
        rect: { top: 80, left: 20, width: 600, height: 420 },
        states: {
          content: [{ slot: "surface", top: 0, left: 0, width: 600, height: 460 }],
        },
      }],
    }]

    expect(getLoadingGeometryDiagnostics({
      geometry: {
        disposition: "verify",
        owners: [{ owner: "late-slot-drift", requiredSlots: ["surface"], tolerance: geometryTolerance }],
      },
    }, samples)).toMatchObject({
      error: "loading-geometry-content-only-drift:late-slot-drift:surface:height",
      comparisons: expect.arrayContaining([
        expect.objectContaining({ measurement: "owner-document", exceeded: [] }),
        expect.objectContaining({
          measurement: "content-only",
          sampledAtMs: { skeleton: 1, content: 3 },
          delta: expect.objectContaining({ height: 60 }),
          exceeded: ["height"],
        }),
      ]),
    })
  })

  it("fails a subsequent content-only owner resize after the first content-only frame is stable", () => {
    const surface = { slot: "surface", top: 0, left: 0, width: 600, height: 400 }
    const samples = [{
      sampledAtMs: 1,
      owners: [{
        owner: "late-owner-drift",
        phase: "handoff",
        rect: { top: 80, left: 20, width: 600, height: 420 },
        states: { skeleton: [surface], content: [surface] },
      }],
    }, {
      sampledAtMs: 2,
      owners: [{
        owner: "late-owner-drift",
        phase: "content",
        rect: { top: 80, left: 20, width: 600, height: 420 },
        states: { content: [surface] },
      }],
    }, {
      sampledAtMs: 3,
      owners: [{
        owner: "late-owner-drift",
        phase: "content",
        rect: { top: 80, left: 20, width: 600, height: 460 },
        states: { content: [surface] },
      }],
    }]

    expect(getLoadingGeometryDiagnostics({
      geometry: {
        disposition: "verify",
        owners: [{ owner: "late-owner-drift", requiredSlots: ["surface"], tolerance: geometryTolerance }],
      },
    }, samples)).toMatchObject({
      error: "loading-geometry-owner-drift:late-owner-drift:height",
      comparisons: expect.arrayContaining([
        expect.objectContaining({
          measurement: "owner-document",
          sampledAtMs: { skeleton: 1, content: 3 },
          delta: expect.objectContaining({ height: 40 }),
          exceeded: ["height"],
        }),
      ]),
    })
  })

  it("records CSSOM-only geometry changes through ResizeObserver", async () => {
    const browser = await chromium.launch({ headless: true })

    try {
      const page = await browser.newPage()
      await page.addInitScript(installLoadingStructureObserver)
      await page.goto(`data:text/html,${encodeURIComponent(`
        <style>
          [data-loading-owner="resize-observer-probe"] { width: 120px; }
          [data-loading-structure-state] { height: 40px; }
        </style>
        <div data-loading-owner="resize-observer-probe" data-loading-phase="loading">
          <div data-loading-structure-state="skeleton" data-loading-slot="surface"></div>
        </div>
      `)}`)

      await page.waitForFunction(() => {
        const samples = (window as BrowserStructureWindow).__lunafoxLoadingStructureSamples
        return samples?.some((sample) => sample.owners.some((owner) => (
          owner.owner === "resize-observer-probe" && owner.rect?.width === 120
        )))
      })
      await page.evaluate(() => {
        document.styleSheets[0].insertRule(
          '[data-loading-owner="resize-observer-probe"] { width: 240px; }',
          document.styleSheets[0].cssRules.length
        )
      })
      await page.waitForFunction(() => {
        const samples = (window as BrowserStructureWindow).__lunafoxLoadingStructureSamples
        return samples?.some((sample) => sample.owners.some((owner) => (
          owner.owner === "resize-observer-probe" && owner.rect?.width === 240
        )))
      })
    } finally {
      await browser.close()
    }
  }, 15_000)

  it("does not infer geometry checks from unconfigured local owners", () => {
    expect(getLoadingGeometryDiagnostics(pairedGeometryTarget, pairedStructureSample("local-dialog-loading"))).toEqual({
      error: "loading-geometry-owner-not-observed:route-content",
      comparisons: [],
    })
    expect(getLoadingGeometryDiagnostics({ geometry: { disposition: "not-applicable" } }, pairedStructureSample("local-dialog-loading"))).toEqual({
      error: "",
      comparisons: [],
    })
  })

  it("expects the complete target detail shell on child-route cold loads", () => {
    for (const id of [
      "route_targets_id_screenshots",
      "route_targets_id_settings",
      "route_targets_id_settings_scheduled_scans",
      "route_targets_id_vulnerabilities",
    ]) {
      const contract = getRouteLoadingContract(id)

      expect(contract?.expectedOwners).toContainEqual({
        owner: "target-detail-shell",
        layer: "workspace",
        intent: "data",
      })
      expect(contract?.geometry.owners).toContainEqual(
        expect.objectContaining({
          owner: "target-detail-shell",
          requiredSlots: [
            "surface",
            "target-detail-shell-header",
            "target-detail-shell-primary-tabs",
            "target-detail-shell-content",
          ],
        })
      )
    }
  })

  it("covers query-triggered loading owners that are not visible on initial route load", () => {
    expect(source).toContain("buildExtraTargets")
    expect(source).toContain("search-results-content")
    expect(source).toContain("statusCode%3D%3D%22200%22")
    expect(source).toContain('routePattern: "/search?q=statusCode%3D%3D%22200%22"')
    expect(source).toContain("expectedOwners")
    expect(source).toContain("expected-owner-not-content")
    expect(source).toContain("expected-owner-layer-mismatch")
  })

  it("samples a cold-load timeline by default and fails route contract timeline gaps", () => {
    expect(source).toContain("return [700, 1300, 2200, 4000]")
    expect(source).toContain("getTimelineDiagnosticsError")
    expect(source).toContain("hasExpectedOwnerSettled")
    expect(source).toContain("boot-handoff-overlap-too-long")
    expect(source).toContain("hasBlockingVisibleBoot")
    expect(source).toContain('item.bootExitStarted !== "true"')
    expect(source).toContain("expected-owner-not-observed-in-timeline")
    expect(source).toContain("getTimelineDiagnosticsError(timeline, target, diagnostics, structureSamples) ||")
    expect(source).toContain("getLoadingDiagnosticsError(diagnostics, target) ||")
    expect(source).toContain("geometry.error")
  })

  it("accepts an expected owner observed by the structure observer after coarse timeline samples", () => {
    const target = {
      expectedOwners: [{ owner: "workspace-content", layer: "workspace" }],
    }
    const timeline = [{
      requestedAtMs: 700,
      diagnostics: { loadingOwners: [] },
    }]
    const finalDiagnostics = { loadingOwners: [] }
    const structureSamples = [{
      sampledAtMs: 6_100,
      owners: [{
        owner: "workspace-content",
        layer: "workspace",
        phase: "loading",
        states: { skeleton: [] },
      }],
    }]

    expect(getTimelineDiagnosticsError(timeline, target, finalDiagnostics, structureSamples)).toBe("")
    expect(getTimelineDiagnosticsError(timeline, target, finalDiagnostics, [{
      ...structureSamples[0],
      owners: [{ ...structureSamples[0].owners[0], layer: "route" }],
    }])).toBe("expected-owner-not-observed-in-timeline")
    expect(getTimelineDiagnosticsError(timeline, target, finalDiagnostics, [{
      ...structureSamples[0],
      owners: [{ ...structureSamples[0].owners[0], owner: "other-content" }],
    }])).toBe("expected-owner-not-observed-in-timeline")
  })

  it("retries transient timeline inspection across navigation context swaps", () => {
    expect(source).toContain("function isTransientNavigationInspectionError")
    expect(source).toContain("Execution context was destroyed")
    expect(source).toContain("Cannot find context with specified id")
    expect(source).toContain("function inspectLoadingDomDuringTimeline")
    expect(source).toContain("await page.waitForTimeout(50)")
    expect(source).toContain("inspectLoadingDomDuringTimeline(page")
  })

  it("covers safe interaction-triggered loading owners and local dynamic fallbacks", () => {
    expect(source).toContain("LOADING_SMOKE_INTERACTIONS")
    expect(source).toContain("process.env.LOADING_SMOKE_INTERACTIONS || \"default\"")
    expect(source).toContain("buildInteractionTargets")
    expect(source).toContain("runInteractionTarget")
    expect(source).toContain("interactionTargets")
    expect(source).toContain("interactionName")
    expect(source).toContain("afterAction")
    expect(source).toContain("expectedVisible")
    expect(source).toContain("followupSelector")
    expect(source).toContain("quick-scan-drawer")
    expect(source).toContain('routePattern: "/settings/agents"')
    expect(source).toContain("button:has-text('打开编排')")
    expect(source).toContain("[data-workflow-composition-canvas]")
    expect(source).toContain("interactionMode === \"default\"")
    expect(source).toContain("interactionMode !== \"0\" && interactionMode !== \"false\" && interactionMode !== \"off\"")
    expect(source).toContain("quick-scan-dialog")
    expect(source).toContain("ArchitectureFlowCanvasSkeleton")
    expect(source).toContain("WorkflowConfigPreviewSkeleton")
    expect(source).not.toContain("NucleiRepoContentSkeleton")
    expect(source).not.toContain("/tools/nuclei/[repoId]")
  })

  it("writes a durable loading handoff report", () => {
    expect(source).toContain("loading-handoff-smoke.json")
    expect(source).toContain("diagnostics")
    expect(source).toContain("bootExitStarted")
    expect(source).toContain("ariaHidden")
    expect(source).toContain("visibility")
    expect(source).toContain("timelineScheduleMs")
    expect(source).toContain("captureRouteTimeline")
    expect(source).toContain("requestedAtMs")
    expect(source).toContain("sampledAtMs")
    expect(source).toContain("targetIdFilter")
    expect(source).toContain("stuckOwners")
    expect(source).toContain("visibleDataSlots")
    expect(source).toContain("visibleSkeletonSlashSeparators")
    expect(source).toContain("geometry")
    expect(source).toContain("redirect: item.redirect")
  })
})
