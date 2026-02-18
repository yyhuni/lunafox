import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import * as fingerprintHooks from "@/hooks/use-fingerprints"
import { fingerprintKeys } from "@/hooks/use-fingerprints"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

type MutationHook = () => { mutateAsync: (variables?: unknown) => Promise<unknown> }

const getMutationHook = (name: string) =>
  (fingerprintHooks as unknown as Record<string, MutationHook>)[name]

const serviceMocks = vi.hoisted(() => {
  const store: Record<string, ReturnType<typeof vi.fn>> = {}
  return new Proxy(store, {
    get(target, prop) {
      if (typeof prop !== "string") return undefined
      if (!target[prop]) {
        target[prop] = vi.fn()
      }
      return target[prop]
    },
  }) as Record<string, ReturnType<typeof vi.fn>>
})

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/services/fingerprint.service", () => ({
  FingerprintService: serviceMocks,
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

const eholePayload = {
  cms: "nginx",
  method: "keyword",
  location: "body",
  keyword: ["nginx"],
  isImportant: false,
  type: "middleware",
}

const gobyPayload = {
  name: "goby",
  logic: "and",
  rule: [{ label: "title", feature: "nginx", is_equal: true }],
}

const wappalyzerPayload = {
  name: "wapp",
  cats: [1],
  cookies: {},
  headers: {},
  scriptSrc: [],
  js: [],
  implies: [],
  meta: {},
  html: [],
  description: "desc",
  website: "",
  cpe: "",
}

const fingersPayload = {
  name: "fingers",
  link: "",
  rule: [{ regexps: [{ regexp: "nginx" }] }],
  tag: [],
  focus: false,
  defaultPort: [80],
}

const fingerPrintHubPayload = {
  fpId: "fp-1",
  name: "fingerprinthub",
  author: "ops",
  tags: "web",
  severity: "info",
  metadata: {},
  http: [],
  sourceFile: "fp.yaml",
}

const arlPayload = {
  name: "arl",
  rule: "title=\"nginx\"",
}

const makeCreatedRecord = (payload: Record<string, unknown>) => ({
  id: 1,
  createdAt: "2026-02-12T00:00:00Z",
  ...payload,
})

const fingerprintConfigs = [
  {
    name: "ehole",
    keys: fingerprintKeys.ehole,
    hooks: {
      create: "useCreateEholeFingerprint",
      update: "useUpdateEholeFingerprint",
      remove: "useDeleteEholeFingerprint",
      import: "useImportEholeFingerprints",
      bulkDelete: "useBulkDeleteEholeFingerprints",
      deleteAll: "useDeleteAllEholeFingerprints",
    },
    service: {
      create: "createEholeFingerprint",
      update: "updateEholeFingerprint",
      remove: "deleteEholeFingerprint",
      import: "importEholeFingerprints",
      bulkDelete: "bulkDeleteEholeFingerprints",
      deleteAll: "deleteAllEholeFingerprints",
    },
    payload: eholePayload,
    updatePayload: { cms: "apache" },
  },
  {
    name: "goby",
    keys: fingerprintKeys.goby,
    hooks: {
      create: "useCreateGobyFingerprint",
      update: "useUpdateGobyFingerprint",
      remove: "useDeleteGobyFingerprint",
      import: "useImportGobyFingerprints",
      bulkDelete: "useBulkDeleteGobyFingerprints",
      deleteAll: "useDeleteAllGobyFingerprints",
    },
    service: {
      create: "createGobyFingerprint",
      update: "updateGobyFingerprint",
      remove: "deleteGobyFingerprint",
      import: "importGobyFingerprints",
      bulkDelete: "bulkDeleteGobyFingerprints",
      deleteAll: "deleteAllGobyFingerprints",
    },
    payload: gobyPayload,
    updatePayload: { logic: "or" },
  },
  {
    name: "wappalyzer",
    keys: fingerprintKeys.wappalyzer,
    hooks: {
      create: "useCreateWappalyzerFingerprint",
      update: "useUpdateWappalyzerFingerprint",
      remove: "useDeleteWappalyzerFingerprint",
      import: "useImportWappalyzerFingerprints",
      bulkDelete: "useBulkDeleteWappalyzerFingerprints",
      deleteAll: "useDeleteAllWappalyzerFingerprints",
    },
    service: {
      create: "createWappalyzerFingerprint",
      update: "updateWappalyzerFingerprint",
      remove: "deleteWappalyzerFingerprint",
      import: "importWappalyzerFingerprints",
      bulkDelete: "bulkDeleteWappalyzerFingerprints",
      deleteAll: "deleteAllWappalyzerFingerprints",
    },
    payload: wappalyzerPayload,
    updatePayload: { description: "patched" },
  },
  {
    name: "fingers",
    keys: fingerprintKeys.fingers,
    hooks: {
      create: "useCreateFingersFingerprint",
      update: "useUpdateFingersFingerprint",
      remove: "useDeleteFingersFingerprint",
      import: "useImportFingersFingerprints",
      bulkDelete: "useBulkDeleteFingersFingerprints",
      deleteAll: "useDeleteAllFingersFingerprints",
    },
    service: {
      create: "createFingersFingerprint",
      update: "updateFingersFingerprint",
      remove: "deleteFingersFingerprint",
      import: "importFingersFingerprints",
      bulkDelete: "bulkDeleteFingersFingerprints",
      deleteAll: "deleteAllFingersFingerprints",
    },
    payload: fingersPayload,
    updatePayload: { focus: true },
  },
  {
    name: "fingerprinthub",
    keys: fingerprintKeys.fingerprinthub,
    hooks: {
      create: "useCreateFingerPrintHubFingerprint",
      update: "useUpdateFingerPrintHubFingerprint",
      remove: "useDeleteFingerPrintHubFingerprint",
      import: "useImportFingerPrintHubFingerprints",
      bulkDelete: "useBulkDeleteFingerPrintHubFingerprints",
      deleteAll: "useDeleteAllFingerPrintHubFingerprints",
    },
    service: {
      create: "createFingerPrintHubFingerprint",
      update: "updateFingerPrintHubFingerprint",
      remove: "deleteFingerPrintHubFingerprint",
      import: "importFingerPrintHubFingerprints",
      bulkDelete: "bulkDeleteFingerPrintHubFingerprints",
      deleteAll: "deleteAllFingerPrintHubFingerprints",
    },
    payload: fingerPrintHubPayload,
    updatePayload: { tags: "web,updated" },
  },
  {
    name: "arl",
    keys: fingerprintKeys.arl,
    hooks: {
      create: "useCreateARLFingerprint",
      update: "useUpdateARLFingerprint",
      remove: "useDeleteARLFingerprint",
      import: "useImportARLFingerprints",
      bulkDelete: "useBulkDeleteARLFingerprints",
      deleteAll: "useDeleteAllARLFingerprints",
    },
    service: {
      create: "createARLFingerprint",
      update: "updateARLFingerprint",
      remove: "deleteARLFingerprint",
      import: "importARLFingerprints",
      bulkDelete: "bulkDeleteARLFingerprints",
      deleteAll: "deleteAllARLFingerprints",
    },
    payload: arlPayload,
    updatePayload: { rule: "title=\"apache\"" },
  },
]

describe("use-fingerprints mutation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it.each(fingerprintConfigs)("create $name 成功后失效列表与 stats", async (config) => {
    serviceMocks[config.service.create].mockResolvedValue(makeCreatedRecord(config.payload))

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const hook = getMutationHook(config.hooks.create)
    const { result } = renderHookWithProviders(() => hook(), { queryClient })

    await act(async () => {
      await result.current.mutateAsync(config.payload)
    })

    expect(serviceMocks[config.service.create]).toHaveBeenCalled()
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: config.keys.all() })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: fingerprintKeys.stats() })
  })

  it.each(fingerprintConfigs)("update $name 成功后失效列表与 detail", async (config) => {
    serviceMocks[config.service.update].mockResolvedValue({
      ...makeCreatedRecord(config.payload),
      ...config.updatePayload,
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const hook = getMutationHook(config.hooks.update)
    const { result } = renderHookWithProviders(() => hook(), { queryClient })

    await act(async () => {
      await result.current.mutateAsync({
        id: 42,
        data: config.updatePayload,
      })
    })

    expect(serviceMocks[config.service.update]).toHaveBeenCalledWith(42, config.updatePayload)
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: config.keys.all() })
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: config.keys.detail(42),
    })
  })

  it.each(fingerprintConfigs)("delete $name 成功后失效列表与 stats", async (config) => {
    serviceMocks[config.service.remove].mockResolvedValue(undefined)

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const hook = getMutationHook(config.hooks.remove)
    const { result } = renderHookWithProviders(() => hook(), { queryClient })

    await act(async () => {
      await result.current.mutateAsync(9)
    })

    expect(serviceMocks[config.service.remove]).toHaveBeenCalledWith(9)
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: config.keys.all() })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: fingerprintKeys.stats() })
  })

  it.each(fingerprintConfigs)("import $name 成功后失效列表与 stats", async (config) => {
    serviceMocks[config.service.import].mockResolvedValue({
      created: 2,
      failed: 0,
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const hook = getMutationHook(config.hooks.import)
    const { result } = renderHookWithProviders(() => hook(), { queryClient })

    await act(async () => {
      await result.current.mutateAsync(
        new File(["fp"], "fp.json", { type: "application/json" })
      )
    })

    expect(serviceMocks[config.service.import]).toHaveBeenCalled()
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: config.keys.all() })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: fingerprintKeys.stats() })
  })

  it.each(fingerprintConfigs)("bulk delete $name 成功后失效列表与 stats", async (config) => {
    serviceMocks[config.service.bulkDelete].mockResolvedValue({
      deleted: 2,
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const hook = getMutationHook(config.hooks.bulkDelete)
    const { result } = renderHookWithProviders(() => hook(), { queryClient })

    await act(async () => {
      await result.current.mutateAsync([1, 2])
    })

    expect(serviceMocks[config.service.bulkDelete]).toHaveBeenCalledWith([1, 2])
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: config.keys.all() })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: fingerprintKeys.stats() })
  })

  it.each(fingerprintConfigs)("deleteAll $name 成功后失效列表与 stats", async (config) => {
    serviceMocks[config.service.deleteAll].mockResolvedValue({
      deleted: 5,
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const hook = getMutationHook(config.hooks.deleteAll)
    const { result } = renderHookWithProviders(() => hook(), { queryClient })

    await act(async () => {
      await result.current.mutateAsync()
    })

    expect(serviceMocks[config.service.deleteAll]).toHaveBeenCalled()
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: config.keys.all() })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: fingerprintKeys.stats() })
  })

  it("batch create Ehole 成功后失效列表与 stats", async () => {
    serviceMocks.batchCreateEholeFingerprints.mockResolvedValue({
      created: 2,
      failed: 0,
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(
      () => fingerprintHooks.useBatchCreateEholeFingerprints(),
      { queryClient }
    )

    await act(async () => {
      await result.current.mutateAsync([
        eholePayload,
        { ...eholePayload, cms: "apache" },
      ])
    })

    expect(serviceMocks.batchCreateEholeFingerprints).toHaveBeenCalled()
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: fingerprintKeys.ehole.all() })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: fingerprintKeys.stats() })
  })

  it("batch create Ehole 失败时默认错误 toast 被静默", async () => {
    serviceMocks.batchCreateEholeFingerprints.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "SERVER_ERROR",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(
      () => fingerprintHooks.useBatchCreateEholeFingerprints()
    )

    await act(async () => {
      await expect(result.current.mutateAsync([eholePayload])).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).not.toHaveBeenCalled()
  })

  it("deleteAll 出错时默认错误 toast 被静默", async () => {
    serviceMocks.deleteAllEholeFingerprints.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "SERVER_ERROR",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(
      () => fingerprintHooks.useDeleteAllEholeFingerprints()
    )

    await act(async () => {
      await expect(result.current.mutateAsync()).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).not.toHaveBeenCalled()
  })
})
