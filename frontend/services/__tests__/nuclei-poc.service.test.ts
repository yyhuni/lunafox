import { beforeEach, describe, expect, it, vi } from "vitest"
import { AxiosError } from "axios"

const apiClientMock = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
}))

vi.mock("@/lib/api-client", () => ({ default: apiClientMock }))

import {
  getNucleiPoc,
  getNucleiPocActiveSyncTaskName,
  getNucleiPocFilterOptions,
  getNucleiPocSource,
  getNucleiPocSyncTask,
  getNucleiPocErrorReason,
  listNucleiPocs,
  parsePocTemplateId,
  parseTaskId,
  setNucleiPocActivation,
  syncNucleiPocSource,
  updateNucleiPoc,
} from "@/services/nuclei-poc.service"

const taskName = "nucleiPocSyncTasks/00000000-0000-4000-8000-000000000001"
const pocName = "nucleiPocs/http-missing-security-headers"

describe("nuclei-poc.service", () => {
  beforeEach(() => vi.clearAllMocks())

  it("uses the canonical source, task, list, filter-options, detail, and update boundaries", async () => {
    apiClientMock.get
      .mockResolvedValueOnce({ data: { name: "nucleiPocSources/current", sourceType: "git" } })
      .mockResolvedValueOnce({ data: { name: taskName, state: "SUCCEEDED" } })
      .mockResolvedValueOnce({ data: { results: [], totalSize: 0 } })
      .mockResolvedValueOnce({ data: { results: [{ value: "cve", label: "cve", count: 2 }] } })
      .mockResolvedValueOnce({ data: { name: pocName, tags: [], cve: [], cwe: [], references: [], content: "id: demo" } })
    apiClientMock.patch.mockResolvedValueOnce({ data: { name: pocName, isEnabled: false } })

    await getNucleiPocSource()
    await getNucleiPocSyncTask(taskName)
    await listNucleiPocs({ pageSize: 25, filter: 'severity=="high"', orderBy: "templateId asc" })
    await getNucleiPocFilterOptions("tags")
    await getNucleiPoc(pocName)
    await updateNucleiPoc({ name: pocName, isEnabled: false, updateMask: ["isEnabled"] })

    expect(apiClientMock.get).toHaveBeenNthCalledWith(1, "/nucleiPocSources/current")
    expect(apiClientMock.get).toHaveBeenNthCalledWith(2, "/nucleiPocSyncTasks/00000000-0000-4000-8000-000000000001")
    expect(apiClientMock.get).toHaveBeenNthCalledWith(3, "/nucleiPocs", {
      params: { pageSize: 25, filter: 'severity=="high"', orderBy: "templateId asc" },
    })
    expect(apiClientMock.get).toHaveBeenNthCalledWith(4, "/nucleiPocs/filterOptions", { params: { field: "tags" } })
    expect(apiClientMock.get).toHaveBeenNthCalledWith(5, "/nucleiPocs/http-missing-security-headers")
    expect(apiClientMock.patch).toHaveBeenCalledWith("/nucleiPocs/http-missing-security-headers", {
      name: pocName,
      isEnabled: false,
      updateMask: ["isEnabled"],
    })
  })

  it("strictly validates the tags filter-options request and response", async () => {
    await expect(getNucleiPocFilterOptions("severity" as never)).rejects.toThrow("only the tags field")
    expect(apiClientMock.get).not.toHaveBeenCalled()

    apiClientMock.get.mockResolvedValueOnce({ data: { results: [{ value: "CVE", label: "CVE", count: 1 }] } })
    await expect(getNucleiPocFilterOptions("tags")).rejects.toThrow("invalid value")

    apiClientMock.get.mockResolvedValueOnce({ data: { results: [{ value: "cve", label: "cve", count: 0 }] } })
    await expect(getNucleiPocFilterOptions("tags")).rejects.toThrow("invalid count")
  })

  it("rejects nullable detail collections at the service boundary", async () => {
    apiClientMock.get.mockResolvedValueOnce({
      data: { name: pocName, tags: [], cve: null, cwe: [], references: [], content: "id: demo" },
    })

    await expect(getNucleiPoc(pocName)).rejects.toThrow("invalid cve")
  })

  it("sends a caller-owned request id and never falls back to a legacy route", async () => {
    apiClientMock.post.mockResolvedValueOnce({ data: { name: taskName, state: "VALIDATING_SOURCE" } })

    await syncNucleiPocSource({
      requestId: "00000000-0000-4000-8000-000000000002",
      sourceType: "gitee",
      repoUrl: "https://gitee.com/example/nuclei-templates.git",
    })

    expect(apiClientMock.post).toHaveBeenCalledWith("/nucleiPocSources:sync", {
      requestId: "00000000-0000-4000-8000-000000000002",
      sourceType: "gitee",
      repoUrl: "https://gitee.com/example/nuclei-templates.git",
    })
  })

  it("uses the full-catalog activation custom method with only the explicit target", async () => {
    apiClientMock.post.mockResolvedValueOnce({ data: { enabled: false, affectedCount: 2 } })

    await expect(setNucleiPocActivation({ enabled: false })).resolves.toEqual({ enabled: false, affectedCount: 2 })
    expect(apiClientMock.post).toHaveBeenCalledWith("/nucleiPocs:setActivation", { enabled: false })
  })

  it("strips runtime extras and rejects malformed activation responses", async () => {
    await expect(
      setNucleiPocActivation({ enabled: true, pageSize: 50 } as never),
    ).rejects.toThrow()
    expect(apiClientMock.post).not.toHaveBeenCalled()

    apiClientMock.post.mockResolvedValueOnce({ data: { enabled: true, affectedCount: 1, pageToken: "leak" } })
    await expect(setNucleiPocActivation({ enabled: true })).rejects.toThrow(/unsupported field/)
    expect(apiClientMock.post).toHaveBeenLastCalledWith("/nucleiPocs:setActivation", { enabled: true })

    apiClientMock.post.mockResolvedValueOnce({ data: { enabled: false, affectedCount: 1 } })
    await expect(setNucleiPocActivation({ enabled: true })).rejects.toThrow(/does not match/)
  })

  it("rejects an invalid activation target before transport", async () => {
    await expect(setNucleiPocActivation({ enabled: undefined as never })).rejects.toThrow()
    expect(apiClientMock.post).not.toHaveBeenCalled()
  })

  it("rejects non-canonical resource names before making a request", () => {
    expect(() => parseTaskId(` ${taskName}`)).toThrow()
    expect(() => parseTaskId("nucleiPocSyncTasks/00000000-0000-0000-0000-000000000000")).toThrow()
    expect(() => parsePocTemplateId(`${pocName} `)).toThrow()
    expect(() => parsePocTemplateId("nucleiPocs/http%2Fescape")).toThrow()
    expect(apiClientMock.get).not.toHaveBeenCalled()
  })

  it("extracts only the canonical active task from a conflict ErrorInfo", () => {
    const error = Object.assign(new AxiosError("conflict"), {
      response: {
        status: 409,
        data: {
          error: {
            code: 409,
            status: "ABORTED",
            message: "Another Nuclei POC sync is already running.",
            details: [{
              "@type": "type.googleapis.com/google.rpc.ErrorInfo",
              reason: "SYNC_ALREADY_RUNNING",
              domain: "lunafox",
              metadata: { task: taskName },
            }],
          },
        },
      },
    })

    expect(getNucleiPocErrorReason(error)).toBe("SYNC_ALREADY_RUNNING")
    expect(getNucleiPocActiveSyncTaskName(error)).toBe(taskName)
  })

  it("rejects malformed active-task metadata without falling back to the raw message", () => {
    const error = Object.assign(new AxiosError("conflict"), {
      response: {
        status: 409,
        data: {
          error: {
            status: "ABORTED",
            message: `Another task is running: ${taskName}`,
            details: [{ reason: "SYNC_ALREADY_RUNNING", metadata: { task: taskName } }],
          },
        },
      },
    })

    expect(getNucleiPocErrorReason(error)).toBeNull()
    expect(getNucleiPocActiveSyncTaskName(error)).toBeNull()
  })
})
