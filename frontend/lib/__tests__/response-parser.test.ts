import {
  getErrorCode,
  getErrorMessage,
  getErrorResponseData,
  getPaginationMeta,
  isErrorResponse,
  isLegacyResponse,
  isSuccessResponse,
  parseResponse,
} from "@/lib/response-parser"
import { describe, expect, it } from "vitest"

describe("response-parser", () => {
  it("识别标准错误响应", () => {
    const response = { error: { code: "NOT_FOUND", message: "missing" } }
    expect(isErrorResponse(response)).toBe(true)
    expect(getErrorCode(response)).toBe("NOT_FOUND")
    expect(getErrorMessage(response)).toBe("missing")
  })

  it("识别旧版响应并解析 data", () => {
    const response = {
      code: "200",
      state: "success",
      message: "ok",
      data: { id: 1, name: "demo" },
    }
    expect(isLegacyResponse(response)).toBe(true)
    expect(parseResponse<{ id: number; name: string }>(response)).toEqual({ id: 1, name: "demo" })
  })

  it("新格式响应直接返回数据", () => {
    const response = { total: 2, results: [{ id: 1 }, { id: 2 }] }
    expect(isSuccessResponse(response)).toBe(true)
    expect(parseResponse<typeof response>(response)).toEqual(response)
  })

  it("错误对象可提取 response.data", () => {
    const error = { response: { data: { error: { code: "SERVER_ERROR" } } } }
    expect(getErrorResponseData(error)).toEqual({ error: { code: "SERVER_ERROR" } })
  })

  it("可提取分页元信息并兼容 snake_case", () => {
    const camel = { total: 100, page: 2, pageSize: 20, totalPages: 5 }
    const snake = { total: 100, page: 2, page_size: 20, total_pages: 5 }

    expect(getPaginationMeta(camel)).toEqual({
      total: 100,
      page: 2,
      pageSize: 20,
      totalPages: 5,
    })
    expect(getPaginationMeta(snake)).toEqual({
      total: 100,
      page: 2,
      pageSize: 20,
      totalPages: 5,
    })
  })
})
