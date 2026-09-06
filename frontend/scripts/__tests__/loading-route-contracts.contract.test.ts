import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

import {
  getBoundedGeometryExceptionValidationErrors,
  getLoadingGeometryInventoryEntry,
  getLoadingGeometryInventoryValidationErrors,
  getMissingLoadingGeometrySlotSourceDeclarations,
  getResolvedIntrinsicTableSettlementValidationErrors,
  getRouteLoadingContract,
  routeLoadingContracts,
} from "../loading-route-contracts.mjs"
import { collectRouteInventory } from "../route-inventory.mjs"

const routeInventory = collectRouteInventory(path.resolve(process.cwd(), "app"))

describe("loading route contracts", () => {
  it("covers every active route from the shared route inventory", () => {
    const contractIds = new Set(routeLoadingContracts.map((contract) => contract.id))
    const missing = routeInventory.routes
      .map((route) => route.id)
      .filter((id) => !contractIds.has(id))

    expect(missing).toEqual([])
  })

  it("keeps expected owners on the loading control-plane vocabulary", () => {
    for (const contract of routeLoadingContracts) {
      expect(contract.classification.trim()).toBe(contract.classification)
      expect(contract.classification).not.toBe("")
      expect([
        "authenticated-redirect",
        "direct-client-route",
        "protected-route-content",
        "public-auth-entry",
        "query-driven-workspace",
        "query-triggered-section",
        "sectioned-dashboard",
        "server-redirect",
      ]).toContain(contract.classification)

      if (contract.classification === "server-redirect") {
        expect(contract.expectedFirstScreenLayer).toBeNull()
        expect(contract.allowedLayers).toEqual([])
        expect(contract.expectedOwners).toEqual([])
        expect(contract.redirectTarget).toMatch(/^\//)
      } else {
        expect(contract.allowedLayers).toContain(contract.expectedFirstScreenLayer)
        expect(contract.expectedOwners.length).toBeGreaterThan(0)
      }
      expect(["verify", "bounded", "not-applicable"]).toContain(contract.geometry.disposition)

      if (["verify", "bounded"].includes(contract.geometry.disposition)) {
        expect(contract.geometry.owners.length).toBeGreaterThan(0)
        if (contract.geometry.disposition === "bounded") {
          expect(contract.geometry.boundedExceptions.length).toBeGreaterThan(0)
        }
      } else {
        expect(contract.geometry.owners).toEqual([])
        expect(contract.geometry.reason.trim()).not.toBe("")
      }

      for (const expectedOwner of contract.expectedOwners) {
        expect(expectedOwner.owner.trim()).toBe(expectedOwner.owner)
        expect(expectedOwner.owner).not.toBe("")
        expect([
          "boot",
          "auth-shell",
          "app-shell",
          "route",
          "workspace",
          "section",
          "interaction",
          "compact-pending",
          "domain-status",
        ]).toContain(expectedOwner.layer)
        expect(contract.allowedLayers).toContain(expectedOwner.layer)

        const geometryOwners = contract.geometry.owners.filter(
          (geometryOwner: { owner: string }) => geometryOwner.owner === expectedOwner.owner
        )
        const excludedOwners = contract.geometry.excludedOwners.filter(
          (excludedOwner: { owner: string }) => excludedOwner.owner === expectedOwner.owner
        )
        const boundedExceptions = (contract.geometry.boundedExceptions ?? []).filter(
          (exception: { owner: string }) => exception.owner === expectedOwner.owner
        )

        expect(geometryOwners.length).toBeLessThanOrEqual(1)
        expect(excludedOwners.length).toBeLessThanOrEqual(1)
        expect(boundedExceptions.length).toBeLessThanOrEqual(1)
        expect(geometryOwners.length + excludedOwners.length).toBeGreaterThan(0)
        expect(excludedOwners.length).toBeLessThanOrEqual(1)

        if (geometryOwners.length > 0) {
          expect(["verify", "bounded"]).toContain(contract.geometry.disposition)
          expect(geometryOwners[0]).toMatchObject({
            owner: expectedOwner.owner,
            requiredSlots: expect.arrayContaining(["surface"]),
            tolerance: {
              top: expect.any(Number),
              left: expect.any(Number),
              width: expect.any(Number),
              height: expect.any(Number),
            },
          })
        } else if (excludedOwners.length > 0) {
          expect(excludedOwners[0].reason.trim()).not.toBe("")
        } else {
          expect(contract.geometry.disposition).toBe("bounded")
        }

        if (boundedExceptions.length > 0) {
          expect(geometryOwners).toHaveLength(1)
          expect(contract.geometry.disposition).toBe("bounded")
          expect(boundedExceptions[0]).toMatchObject({
            owner: expectedOwner.owner,
            affectedSlots: expect.any(Array),
            reason: expect.any(String),
            firstFrame: expect.any(Object),
            reviewTrigger: expect.any(String),
            recoveryPath: expect.any(String),
          })
          for (const affectedSlot of boundedExceptions[0].affectedSlots) {
            expect(geometryOwners[0].requiredSlots).not.toContain(affectedSlot)
          }
        }
      }
    }
  })

  it("makes the organizations cold-load workspace owner explicit", () => {
    expect(getRouteLoadingContract("route_organizations")?.expectedOwners).toContainEqual({
      owner: "organization-list-content",
      layer: "workspace",
      intent: "data",
    })
  })

  it("declares intrinsic settlement only for shared paginated table owners", () => {
    const cases = [
      ["route_organizations", "organization-list-content", "organization-list-body", "organization-list-pagination"],
      ["route_scan_history", "scan-history-list-view-content", "scan-history-list-body", "scan-history-list-pagination"],
      ["route_targets", "all-targets-detail-view-content", "all-targets-table-body", "all-targets-pagination"],
      ["route_vulnerabilities", "vulnerabilities-vertical-view-content", "vulnerabilities-rows", "vulnerabilities-pagination"],
      ["route_scan_scheduled", "scheduled-scan-page-route", "scheduled-scan-table", undefined],
      ["route_organizations_id", "organization-detail-view-content", "organization-detail-primary-table", undefined],
      ["route_tools_fingerprints_fingerprinthub", "fingerprinthub-fingerprint-view-content", "fingerprinthub-fingerprint-rows", "fingerprinthub-fingerprint-pagination"],
      ["route_tools_nuclei", "nuclei-poc-catalog-content", "nuclei-poc-catalog-table", undefined],
    ] as const

    for (const [routeId, ownerName, bodySlot, paginationSlot] of cases) {
      const owner = getRouteLoadingContract(routeId)?.geometry.owners.find(
        (item: { owner: string }) => item.owner === ownerName
      )
      const settlement = owner?.resolvedIntrinsicTableSettlement

      expect(settlement).toMatchObject({ bodySlot, ...(paginationSlot ? { paginationSlot } : {}) })
      expect(getResolvedIntrinsicTableSettlementValidationErrors(
        settlement,
        owner?.requiredSlots
      )).toEqual([])
    }
  })

  it("limits content-growth settlement to the scan-history Badge table", () => {
    const scanHistoryOwner = getRouteLoadingContract("route_scan_history")?.geometry.owners.find(
      (owner: { owner: string }) => owner.owner === "scan-history-list-view-content"
    )

    expect(scanHistoryOwner?.resolvedIntrinsicTableSettlement).toMatchObject({
      allowContentGrowth: true,
      bodySlot: "scan-history-list-body",
      paginationSlot: "scan-history-list-pagination",
    })

    for (const contract of routeLoadingContracts) {
      for (const owner of contract.geometry.owners) {
        if (contract.id === "route_scan_history" && owner.owner === "scan-history-list-view-content") continue
        expect(owner.resolvedIntrinsicTableSettlement?.allowContentGrowth).toBeUndefined()
      }
    }
  })

  it("requires every paired owner to match an explicit inventory entry instead of falling back to surface", () => {
    for (const contract of routeLoadingContracts) {
      for (const geometryOwner of contract.geometry.owners) {
        expect(getLoadingGeometryInventoryEntry(contract.id, geometryOwner.owner)).not.toBeNull()
        expect(getLoadingGeometryInventoryValidationErrors(
          contract.id,
          geometryOwner.owner,
          geometryOwner.requiredSlots
        )).toEqual([])
        expect(geometryOwner.requiredSlots).toContain("surface")
        expect(geometryOwner.requiredSlots.length).toBeGreaterThan(1)
      }
    }

    expect(getLoadingGeometryInventoryValidationErrors(
      "route_organizations",
      "organization-list-content",
      ["surface"]
    )).toEqual(["geometry-inventory-slots-mismatch"])
    expect(getLoadingGeometryInventoryValidationErrors(
      "route_organizations",
      "unknown-loading-owner",
      ["surface"]
    )).toEqual(["geometry-inventory-entry-missing"])
  })

  it("rejects geometry slots that production source never declares", () => {
    expect(getMissingLoadingGeometrySlotSourceDeclarations(
      ["surface", "workspace-toolbar", "workspace-body"],
      [
        'getLoadingStructureSlotAttributes("surface")',
        "const toolbarSlot = 'workspace-toolbar'",
      ]
    )).toEqual(["workspace-body"])

    expect(getMissingLoadingGeometrySlotSourceDeclarations(
      ["surface", "workspace-toolbar"],
      ['data-loading-slot="surface"', 'getLoadingStructureSlotAttributes("workspace-toolbar")']
    )).toEqual([])
  })

  it("rejects incomplete bounded exception metadata before route smoke runs", () => {
    expect(getBoundedGeometryExceptionValidationErrors({
      owner: "route-content",
      affectedSlots: ["data-dependent-region"],
      reason: "A browser-owned region cannot share intrinsic geometry.",
    })).toEqual(expect.arrayContaining([
      "bounded-exception-review-trigger-missing",
      "bounded-exception-recovery-path-missing",
      "bounded-exception-first-frame-missing",
    ]))

    expect(getBoundedGeometryExceptionValidationErrors({
      owner: "route-content",
      affectedSlots: ["data-dependent-region"],
      reason: "A browser-owned region cannot share intrinsic geometry.",
      firstFrame: {
        measurement: "cold-browser-geometry",
        stableFrames: 2,
        contentOnlyStable: true,
      },
      reviewTrigger: "When the platform-owned region changes.",
      recoveryPath: "Replace the region with a shared measurable surface.",
    })).toEqual([])

    expect(getBoundedGeometryExceptionValidationErrors({
      owner: "route-content",
      affectedSlots: ["data-dependent-region"],
      reason: "A browser-owned region cannot share intrinsic geometry.",
      firstFrame: {
        measurement: "",
        stableFrames: 1,
        contentOnlyStable: false,
      },
      reviewTrigger: "When the platform-owned region changes.",
      recoveryPath: "Replace the region with a shared measurable surface.",
    })).toEqual(expect.arrayContaining([
      "bounded-exception-first-frame-measurement-missing",
      "bounded-exception-first-frame-stable-frames-invalid",
      "bounded-exception-first-frame-content-only-stability-missing",
    ]))
  })

  it("rejects an intrinsic table settlement outside the owner’s named slots", () => {
    expect(getResolvedIntrinsicTableSettlementValidationErrors({
      bodySlot: "surface",
      paginationSlot: "missing-pagination",
      reason: "Sparse results may shrink the table.",
      reviewTrigger: "When the table layout changes.",
    }, ["surface", "table-body", "pagination"])).toEqual(expect.arrayContaining([
      "resolved-intrinsic-table-settlement-body-slot-invalid",
      "resolved-intrinsic-table-settlement-pagination-slot-invalid",
    ]))

    expect(getResolvedIntrinsicTableSettlementValidationErrors({
      bodySlot: "table-body",
      paginationSlot: "pagination",
      reason: "Sparse results may shrink the table.",
      reviewTrigger: "When the table layout changes.",
    }, ["surface", "table-body", "pagination"])).toEqual([])

    expect(getResolvedIntrinsicTableSettlementValidationErrors({
      bodySlot: "table-body",
      paginationSlot: "pagination",
      allowContentGrowth: false,
      reason: "Multiline content may grow the table.",
      reviewTrigger: "When the table layout changes.",
    }, ["surface", "table-body", "pagination"])).toEqual(expect.arrayContaining([
      "resolved-intrinsic-table-settlement-content-growth-invalid",
    ]))
  })

  it("requires the overview dashboard's structural regions to survive the loading handoff", () => {
    const geometry = getRouteLoadingContract("route_overview")?.geometry.owners.find(
      (owner: { owner: string }) => owner.owner === "overview-sections-loader"
    )

    expect(geometry?.requiredSlots).toEqual([
      "surface",
      "overview-runtime-region",
      "overview-asset-region",
      "overview-operational-region",
    ])
  })

  it("makes the wordlists cold-load workspace owner explicit", () => {
    const contract = getRouteLoadingContract("route_tools_wordlists")

    expect(contract?.expectedOwners).toContainEqual({
      owner: "wordlists-page-content",
      layer: "workspace",
      intent: "data",
    })
    expect(contract?.geometry.owners.find(
      (owner: { owner: string }) => owner.owner === "wordlists-page-content"
    )?.requiredSlots).toEqual([
      "surface",
      "wordlists-controls",
      "wordlists-list",
    ])
  })

  it("makes the backend-owned nuclei POC catalog workspace explicit", () => {
    const contract = getRouteLoadingContract("route_tools_nuclei")
    expect(contract?.expectedFirstScreenLayer).toBe("workspace")
    expect(contract?.expectedOwners).toContainEqual({
      owner: "nuclei-poc-catalog-content",
      layer: "workspace",
      intent: "data",
    })
    expect(contract?.expectedOwners).not.toContainEqual(
      expect.objectContaining({ owner: "nuclei-template-detail-panel" })
    )

    const geometry = contract?.geometry.owners.find(
      (owner: { owner: string }) => owner.owner === "nuclei-poc-catalog-content"
    )
    expect(geometry?.requiredSlots).toEqual([
      "surface",
      "nuclei-poc-catalog-source",
      "nuclei-poc-catalog-table",
    ])
  })

  it("makes hidden-readiness page workspaces explicit below the protected route owner", () => {
    for (const [id, owner] of [
      ["route_scan_scheduled", "scheduled-scan-page-route"],
      ["route_settings_api_keys", "api-keys-page-route"],
      ["route_settings_database_health", "database-health-page-route"],
      ["route_settings_notifications", "notification-settings-page-route"],
      ["route_settings_support", "support-page-route"],
      ["route_settings_system_logs", "system-logs-page-route"],
    ] as const) {
      expect(getRouteLoadingContract(id)?.expectedFirstScreenLayer).toBe("workspace")
      expect(getRouteLoadingContract(id)?.expectedOwners).toContainEqual({
        owner,
        layer: "workspace",
        intent: "data",
      })
    }
  })

  it("measures the complete external-destination notification settings workbench", () => {
    const notificationGeometry = getRouteLoadingContract("route_settings_notifications")?.geometry.owners.find(
      (owner: { owner: string }) => owner.owner === "notification-settings-page-route"
    )

    expect(notificationGeometry?.requiredSlots).toEqual([
      "surface",
      "notification-settings-header",
      "notification-settings-channel-workbench",
    ])
  })

  it("registers every previously unpaired inventory first-screen owner with its exact geometry slots", () => {
    const cases = [
      ["route_organizations_id", "organization-detail-view-content", "workspace"],
      ["route_scan_config_workflows", "scan-workflow-page-content", "workspace"],
      ["route_settings_blacklist", "blacklist-page-content", "workspace"],
      ["route_settings_agents", "agent-list-overview", "section"],
      ["route_settings_agents", "agent-list-toolbar", "section"],
      ["route_settings_agents", "agent-list-results", "section"],
      ["route_targets", "all-targets-detail-view-content", "workspace"],
      ["route_tools_fingerprints_fingerprinthub", "fingerprinthub-fingerprint-view-content", "workspace"],
    ] as const

    for (const [id, owner, layer] of cases) {
      const contract = getRouteLoadingContract(id)

      expect(contract?.expectedFirstScreenLayer).toBe(layer)
      expect(contract?.expectedOwners).toContainEqual({ owner, layer, intent: "data" })
      expect(contract?.geometry).toMatchObject({ disposition: "verify" })
      const geometryOwner = contract?.geometry.owners.find(
        (item: { owner: string }) => item.owner === owner
      )
      expect(geometryOwner).toBeDefined()
      expect(geometryOwner?.requiredSlots).toEqual(getLoadingGeometryInventoryEntry(id, owner)?.requiredSlots)
    }
  })

  it("requires Support to preserve its value-first structural slots", () => {
    const geometry = getRouteLoadingContract("route_settings_support")?.geometry.owners.find(
      (owner: { owner: string }) => owner.owner === "support-page-route"
    )

    expect(geometry?.requiredSlots).toEqual([
      "surface",
      "support-page-header",
      "support-page-value-band",
      "support-page-actions",
    ])
  })

  it("requires every target and scan-history detail route to pair the feature-local shell slots", () => {
    const cases = [
      {
        ids: [
          "route_targets_id_directories",
          "route_targets_id_endpoints",
          "route_targets_id_ip_addresses",
          "route_targets_id_overview",
          "route_targets_id_screenshots",
          "route_targets_id_settings",
          "route_targets_id_settings_scheduled_scans",
          "route_targets_id_subdomains",
          "route_targets_id_vulnerabilities",
          "route_targets_id_websites",
          "route_targets_id_websites_websiteid_section",
        ],
        owner: "target-detail-shell",
        requiredSlots: [
          "surface",
          "target-detail-shell-header",
          "target-detail-shell-primary-tabs",
          "target-detail-shell-content",
        ],
      },
      {
        ids: [
          "route_scan_history_id_directories",
          "route_scan_history_id_endpoints",
          "route_scan_history_id_ip_addresses",
          "route_scan_history_id_overview",
          "route_scan_history_id_screenshots",
          "route_scan_history_id_subdomains",
          "route_scan_history_id_vulnerabilities",
          "route_scan_history_id_websites",
        ],
        owner: "scan-history-detail-shell",
        requiredSlots: [
          "surface",
          "scan-history-detail-shell-header",
          "scan-history-detail-shell-primary-tabs",
          "scan-history-detail-shell-content",
        ],
      },
    ] as const

    for (const { ids, owner, requiredSlots } of cases) {
      for (const id of ids) {
        const contract = getRouteLoadingContract(id)
        expect(contract?.expectedFirstScreenLayer).toBe("workspace")
        expect(contract?.expectedOwners).toContainEqual({ owner, layer: "workspace", intent: "data" })
        expect(contract?.geometry.owners).toContainEqual(
          expect.objectContaining({ owner, requiredSlots })
        )
      }
    }
  })

  it("keeps target website child owners explicit while excluding their deferred cold-entry geometry", () => {
    const cases = [
      [
        "route_targets_id_websites",
        "target-website-evidence-view-content",
        "website-relation-detail-view-content",
      ],
      [
        "route_targets_id_websites_websiteid_section",
        "website-relation-detail-view-content",
        "target-website-evidence-view-content",
      ],
    ] as const

    for (const [id, owner, siblingOwner] of cases) {
      const contract = getRouteLoadingContract(id)

      expect(contract?.expectedOwners).toContainEqual({ owner, layer: "section", intent: "data" })
      expect(contract?.expectedOwners).not.toContainEqual(expect.objectContaining({ owner: siblingOwner }))
      expect(contract?.geometry.owners).not.toContainEqual(expect.objectContaining({ owner }))
      expect(contract?.geometry.excludedOwners).toContainEqual(
        expect.objectContaining({
          owner,
          reason: expect.stringContaining("cold first entry"),
        })
      )
    }
  })

  it("keeps resolved-only route reveal owners out of paired geometry contracts", () => {
    const supportGeometry = getRouteLoadingContract("route_settings_support")?.geometry
    const agentsGeometry = getRouteLoadingContract("route_settings_agents")?.geometry

    expect(supportGeometry).toMatchObject({
      disposition: "verify",
      owners: [
        expect.objectContaining({ owner: "support-page-route" }),
      ],
      excludedOwners: [
        expect.objectContaining({
          owner: "auth-layout-route-content",
          reason: expect.stringContaining("ContentReveal"),
        }),
      ],
    })
    expect(supportGeometry?.owners).not.toContainEqual(
      expect.objectContaining({ owner: "auth-layout-route-content" })
    )

    expect(agentsGeometry).toMatchObject({
      disposition: "verify",
      owners: [
        expect.objectContaining({
          owner: "agent-list-overview",
          requiredSlots: ["surface", "agent-list-overview-region"],
        }),
        expect.objectContaining({
          owner: "agent-list-toolbar",
          requiredSlots: ["surface", "agent-list-toolbar-region"],
        }),
        expect.objectContaining({
          owner: "agent-list-results",
          requiredSlots: ["surface", "agent-list-results-primary-region"],
        }),
      ],
      excludedOwners: [
        expect.objectContaining({
          owner: "auth-layout-route-content",
          reason: expect.stringContaining("ContentReveal"),
        }),
      ],
    })
  })

  it("includes the route inventory additions that previously bypassed loading verification", () => {
    expect(getRouteLoadingContract("route_scan_config_engines")?.expectedOwners).toContainEqual({
      owner: "engine-catalog-content",
      layer: "workspace",
      intent: "data",
    })
    expect(getRouteLoadingContract("route_targets_id_websites_websiteid_section")?.expectedOwners).toContainEqual({
      owner: "target-detail-shell",
      layer: "workspace",
      intent: "data",
    })
  })

  it("keeps the root server redirect out of client loading ownership", () => {
    expect(getRouteLoadingContract("route_root")).toMatchObject({
      classification: "server-redirect",
      expectedFirstScreenLayer: null,
      expectedOwners: [],
      redirectTarget: "/overview/",
      geometry: {
        disposition: "not-applicable",
        owners: [],
      },
    })
  })

  it("keeps the public login route bound to its real resolved-only owner", () => {
    expect(getRouteLoadingContract("route_login")).toMatchObject({
      classification: "public-auth-entry",
      expectedFirstScreenLayer: "route",
      expectedOwners: [
        {
          owner: "login-page-content",
          layer: "route",
          intent: "route",
        },
      ],
      geometry: {
        disposition: "not-applicable",
        owners: [],
        excludedOwners: [
          expect.objectContaining({ owner: "login-page-content" }),
        ],
      },
    })
  })

  it("keeps critical top-level workspace handoffs explicit in source", () => {
    const sourceByFile = new Map([
      [
        "components/organization/organization-list.tsx",
        readFileSync(path.resolve(process.cwd(), "components/organization/organization-list.tsx"), "utf8"),
      ],
      [
        "components/vulnerabilities/vulnerabilities-vertical-view.tsx",
        readFileSync(path.resolve(process.cwd(), "components/vulnerabilities/vulnerabilities-vertical-view.tsx"), "utf8"),
      ],
      [
        "components/tools/wordlists-page.tsx",
        readFileSync(path.resolve(process.cwd(), "components/tools/wordlists-page.tsx"), "utf8"),
      ],
      [
        "components/tools/nuclei-poc-catalog-page.tsx",
        readFileSync(path.resolve(process.cwd(), "components/tools/nuclei-poc-catalog-page.tsx"), "utf8"),
      ],
      [
        "components/organization/organization-detail-view.tsx",
        readFileSync(path.resolve(process.cwd(), "components/organization/organization-detail-view.tsx"), "utf8"),
      ],
      [
        "components/scan/workflow/scan-workflow-page.tsx",
        readFileSync(path.resolve(process.cwd(), "components/scan/workflow/scan-workflow-page.tsx"), "utf8"),
      ],
      [
        "components/settings/blacklist/blacklist-settings-workspace.tsx",
        readFileSync(path.resolve(process.cwd(), "components/settings/blacklist/blacklist-settings-workspace.tsx"), "utf8"),
      ],
      [
        "components/target/all-targets-detail-view.tsx",
        readFileSync(path.resolve(process.cwd(), "components/target/all-targets-detail-view.tsx"), "utf8"),
      ],
      [
        "components/fingerprints/fingerprint-library-workspace.tsx",
        readFileSync(path.resolve(process.cwd(), "components/fingerprints/fingerprint-library-workspace.tsx"), "utf8"),
      ],
    ])

    for (const [file, source] of sourceByFile) {
      if (file === "components/settings/blacklist/blacklist-settings-workspace.tsx") {
        expect(source).toContain('layer={embedded ? "section" : "workspace"}')
        continue
      }
      expect(source).toContain('layer="workspace"')
    }

    const scanHistoryListSource = readFileSync(
      path.resolve(process.cwd(), "components/scan/history/scan-history-list.tsx"),
      "utf8"
    )
    expect(scanHistoryListSource).toContain('layer = "workspace"')
    expect(scanHistoryListSource).toContain("layer={layer}")
  })

  it("binds fingerprint page owners to the shared workspace handoff", () => {
    const cases = [
      ["components/fingerprints/fingerprinthub-fingerprint-view.tsx", "fingerprinthub-fingerprint-view-content"],
    ] as const
    const workspace = readFileSync(
      path.resolve(process.cwd(), "components/fingerprints/fingerprint-library-workspace.tsx"),
      "utf8"
    )

    expect(workspace).toContain("ContentHandoff")
    expect(workspace).toContain("owner={owner}")
    expect(workspace).toContain('layer="workspace"')

    for (const [file, owner] of cases) {
      const source = readFileSync(path.resolve(process.cwd(), file), "utf8")
      expect(source).toContain("FingerprintLibraryWorkspace")
      expect(source).toContain(`owner="${owner}"`)
    }
  })
})
