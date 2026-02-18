import { describe, expect, it } from "vitest"
import { buildPaginationInfo, normalizePagination } from "@/hooks/_shared/pagination"

describe("normalizePagination", () => {
  it("prefers camelCase values when present", () => {
    const result = normalizePagination(
      { total: 12, page: 2, pageSize: 25, totalPages: 4 },
      1,
      10
    )

    expect(result).toEqual({
      total: 12,
      page: 2,
      pageSize: 25,
      totalPages: 4,
    })
  })

  it("uses snake_case values when camelCase is missing", () => {
    const result = normalizePagination(
      { total: 8, page: 3, page_size: 15, total_pages: 2 },
      1,
      10
    )

    expect(result).toEqual({
      total: 8,
      page: 3,
      pageSize: 15,
      totalPages: 2,
    })
  })

  it("falls back to defaults when values are missing", () => {
    const result = normalizePagination({}, 5, 20)

    expect(result).toEqual({
      total: 0,
      page: 5,
      pageSize: 20,
      totalPages: 0,
    })
  })
})

describe("buildPaginationInfo", () => {
  it("calculates totalPages when not provided", () => {
    const result = buildPaginationInfo({ total: 21, page: 2, pageSize: 10 })

    expect(result).toEqual({
      total: 21,
      page: 2,
      pageSize: 10,
      totalPages: 3,
    })
  })

  it("respects provided totalPages and minimum", () => {
    const result = buildPaginationInfo({
      total: 0,
      page: 1,
      pageSize: 10,
      totalPages: 0,
      minTotalPages: 1,
    })

    expect(result).toEqual({
      total: 0,
      page: 1,
      pageSize: 10,
      totalPages: 1,
    })
  })
})
