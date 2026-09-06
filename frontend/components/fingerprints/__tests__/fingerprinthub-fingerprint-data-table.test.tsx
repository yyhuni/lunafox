import type { ColumnDef } from "@tanstack/react-table"
import { fireEvent, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { createBusinessListQuery } from "@/components/shared/data-table/business-list-query"
import { renderWithProviders } from "@/test/utils/render-with-providers"
import type { FingerPrintHubFingerprint } from "@/types/fingerprint.types"

import { FingerPrintHubFingerprintDataTable } from "../fingerprinthub-fingerprint-data-table"

const fingerprints: FingerPrintHubFingerprint[] = [{
  name: "fingerprintLibraries/fingerprinthub/fingerprints/00000000-0000-4000-8000-000000000001",
  fingerprintId: "nginx",
  displayName: "Nginx",
  author: null,
  tags: null,
  severity: "info",
  metadata: null,
  http: [],
  sourceFile: null,
  createdAt: "2024-12-20T10:00:00Z",
}]

const columns: ColumnDef<FingerPrintHubFingerprint>[] = [{
  accessorKey: "displayName",
  header: "Name",
  meta: { title: "Name" },
}]

describe("FingerPrintHubFingerprintDataTable", () => {
  it("为 page-token 列表只暴露相邻页导航", () => {
    const onPaginationChange = vi.fn()

    renderWithProviders(
      <FingerPrintHubFingerprintDataTable
        data={fingerprints}
        columns={columns}
        query={createBusinessListQuery({ pageSize: 10 })}
        totalCount={20}
        pagination={{ pageIndex: 0, pageSize: 10 }}
        cursorPaginationSummary={{ total: 20 }}
        paginationNavigation={{ mode: "cursor", canFirstPage: false, canPreviousPage: false, canNextPage: true }}
        onPaginationChange={onPaginationChange}
      />
    )

    expect(screen.getByText(/total:\{"count":20\}/)).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "previous" })).toBeDisabled()
    expect(screen.getByRole("button", { name: "next" })).toBeEnabled()
    expect(screen.getByRole("button", { name: "first" })).toBeDisabled()
    expect(screen.queryByRole("button", { name: "last" })).not.toBeInTheDocument()
    expect(screen.queryByText(/page:\{/)).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole("button", { name: "next" }))

    expect(onPaginationChange).toHaveBeenCalledWith({ pageIndex: 1, pageSize: 10 })
  })
})
