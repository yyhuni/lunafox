import { act, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { engineCatalogKeys, useEngineCatalogDetail, useInstallEngine } from "@/hooks/use-engine-catalog"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import type { EngineCatalogDetail } from "@/types/engine-catalog.types"

const serviceMocks = vi.hoisted(() => ({
  installEngine: vi.fn(),
  getEngineCatalogDetail: vi.fn(),
}))
const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/services/engine-catalog.service", () => ({
  installEngine: serviceMocks.installEngine,
  getEngineCatalog: vi.fn(),
  getEngineCatalogDetail: serviceMocks.getEngineCatalogDetail,
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

const request = {
  artifactRef: "registry.example/team/engine@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  allowReplacement: false,
}

describe("useInstallEngine", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("deduplicates a pending installation and invalidates the catalog after success", async () => {
    const deferred = createDeferred<EngineCatalogDetail>()
    serviceMocks.installEngine.mockReturnValue(deferred.promise)
    const { result, queryClient } = renderHookWithProviders(() => useInstallEngine())
    const invalidate = vi.spyOn(queryClient, "invalidateQueries")

    let first!: Promise<EngineCatalogDetail>
    let second!: Promise<EngineCatalogDetail>
    act(() => {
      first = result.current.mutateAsync(request)
      second = result.current.mutateAsync({ ...request, allowReplacement: true })
    })

    expect(second).toBe(first)
    await waitFor(() => expect(serviceMocks.installEngine).toHaveBeenCalledTimes(1))
    expect(serviceMocks.installEngine).toHaveBeenCalledWith(request)

    await act(async () => {
      deferred.resolve(engineDetail())
      await first
    })

    expect(invalidate).toHaveBeenCalledWith({ queryKey: engineCatalogKeys.list })
  })

  it("propagates installation failures and permits a later retry", async () => {
    const installationError = new Error("registry unavailable")
    serviceMocks.installEngine.mockRejectedValueOnce(installationError).mockResolvedValueOnce(engineDetail())
    const { result } = renderHookWithProviders(() => useInstallEngine())

    await expect(result.current.mutateAsync(request)).rejects.toBe(installationError)
    await expect(result.current.mutateAsync(request)).resolves.toEqual(engineDetail())

    expect(serviceMocks.installEngine).toHaveBeenCalledTimes(2)
  })

  it("keeps a replacement conflict available to the confirmation UI without a failure toast", async () => {
    const conflict = {
      response: {
        status: 409,
        data: {
          error: {
            details: [
              { field: "engineId", message: "engine.example.scanner" },
              { field: "currentPackageDigest", message: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" },
              { field: "proposedPackageDigest", message: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc" },
            ],
          },
        },
      },
    }
    serviceMocks.installEngine.mockRejectedValueOnce(conflict)
    const { result } = renderHookWithProviders(() => useInstallEngine())

    await expect(result.current.mutateAsync(request)).rejects.toBe(conflict)
    expect(toastMocks.errorFromCode).not.toHaveBeenCalled()
  })
})

describe("useEngineCatalogDetail", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("stays disabled without an engine ID and loads the exact selected engine", async () => {
    serviceMocks.getEngineCatalogDetail.mockResolvedValue(engineDetail())
    const { result, rerender } = renderHookWithProviders(
      ({ engineId, enabled }) => useEngineCatalogDetail(engineId, enabled),
      { initialProps: { engineId: "", enabled: false } }
    )

    expect(result.current.fetchStatus).toBe("idle")
    expect(serviceMocks.getEngineCatalogDetail).not.toHaveBeenCalled()

    rerender({ engineId: "engine.example.scanner", enabled: true })

    await waitFor(() => expect(result.current.data).toEqual(engineDetail()))
    expect(serviceMocks.getEngineCatalogDetail).toHaveBeenCalledWith("engine.example.scanner")
    expect(serviceMocks.getEngineCatalogDetail).toHaveBeenCalledTimes(1)
  })
})

function engineDetail(): EngineCatalogDetail {
  return {
    name: "engines/engine.example.scanner",
    engineId: "engine.example.scanner",
    manifestVersion: "engine.v5",
    publisher: "example",
    packageVersion: "1.0.0",
    artifactRef: request.artifactRef,
    packageDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    execution: {
      engineApiMajor: 2,
      supportedTargetTypes: ["domain"],
      configSections: [],
    },
    localeResources: {},
  }
}

function createDeferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason: unknown) => void
  const promise = new Promise<T>((promiseResolve, promiseReject) => {
    resolve = promiseResolve
    reject = promiseReject
  })
  return { promise, resolve, reject }
}
