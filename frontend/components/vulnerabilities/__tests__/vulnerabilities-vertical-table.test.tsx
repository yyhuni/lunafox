import * as React from "react"
import { render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { VulnerabilitiesVerticalTable } from "@/components/vulnerabilities/vulnerabilities-vertical-table"
import { MOCK_VULNS } from "@/lib/mock-vulnerabilities"

describe("VulnerabilitiesVerticalTable", () => {
  it("uses the shared table shell tokens instead of local header and row overrides", () => {
    const { container } = render(
      <VulnerabilitiesVerticalTable
        items={MOCK_VULNS.slice(0, 2)}
        selectedId={MOCK_VULNS[0].id}
        selectedRows={[MOCK_VULNS[0]]}
        onSelect={vi.fn()}
        onSelectionChange={vi.fn()}
      />
    )

    const tableShell = container.querySelector("[role='grid']")
    const header = container.querySelector("thead")
    const firstDataRow = container.querySelector("tbody tr")

    expect(tableShell).not.toBeNull()
    expect(header).not.toBeNull()
    expect(firstDataRow).not.toBeNull()

    expect(tableShell?.className).toContain("bg-card")
    expect(tableShell?.className).not.toContain("bg-background")
    expect(header?.className).toContain("sticky")
    expect(header?.className).not.toContain("backdrop-blur-sm")
    expect(header?.className).not.toContain("bg-muted/80")
    expect(container.innerHTML).not.toContain("--vuln-row-h")

    expect(firstDataRow?.className).toContain("hover:bg-secondary")
    expect(firstDataRow?.className).toContain("data-[state=selected]:bg-secondary/70")
    expect(firstDataRow?.className).toContain("h-12")
    expect(firstDataRow?.className).not.toContain("h-[var(--vuln-row-h)]")
    expect(firstDataRow?.className).not.toContain("hover:bg-muted/40")
    expect(firstDataRow?.className).not.toContain("bg-primary/5")
  })

  it("uses shared neutral content styling while preserving semantic cells", () => {
    const { container } = render(
      <VulnerabilitiesVerticalTable
        items={MOCK_VULNS.slice(0, 2)}
        selectedId={MOCK_VULNS[0].id}
        selectedRows={[MOCK_VULNS[0]]}
        onSelect={vi.fn()}
        onSelectionChange={vi.fn()}
      />
    )

    const headers = Array.from(container.querySelectorAll("th"))
    const statusButtons = screen.getAllByRole("button", { name: "markAsPending" })
    const sourceBadge = screen.getByText(MOCK_VULNS[0].source)
    const urlText = screen.getByTitle(MOCK_VULNS[0].url)
    const urlCell = urlText.closest("td")
    const typeCell = screen.getByTitle(MOCK_VULNS[0].vulnType).closest("td")
    const createdAtCell = screen.getAllByText(new Date(MOCK_VULNS[0].createdAt).toLocaleDateString())[0]?.closest("td")

    expect(statusButtons[0]?.className).toContain("size-8")
    expect(statusButtons[0]?.className).not.toContain("md:h-8")
    expect(statusButtons[0]?.className).not.toContain("h-10")
    expect(statusButtons[0]?.className).not.toContain("hover:bg-muted/60")

    for (const header of headers) {
      expect(header.className).not.toContain("h-[var(--vuln-table-head-h)]")
    }

    expect(typeCell?.className).toContain("px-2")
    expect(typeCell?.className).toContain("py-1")
    expect(typeCell?.className).not.toContain("h-9")

    expect(sourceBadge.className).not.toContain("bg-background/50")
    expect(sourceBadge.className).toContain("text-muted-foreground")

    expect(urlCell?.className).toContain("text-muted-foreground")
    expect(urlCell?.className).toContain("text-sm")
    expect(urlCell?.className).not.toContain("opacity-80")

    expect(createdAtCell?.className).toContain("text-muted-foreground")
    expect(createdAtCell?.className).toContain("text-sm")
    expect(createdAtCell?.className).not.toContain("opacity-80")
  })

  it("renders localized full severity labels instead of abbreviated raw values", () => {
    const { container } = render(
      <VulnerabilitiesVerticalTable
        items={MOCK_VULNS.slice(0, 2)}
        selectedId={MOCK_VULNS[0].id}
        selectedRows={[MOCK_VULNS[0]]}
        onSelect={vi.fn()}
        onSelectionChange={vi.fn()}
      />
    )

    const criticalBadge = screen.getByText("critical")
    const highBadge = screen.getByText("high")

    expect(criticalBadge).toBeInTheDocument()
    expect(highBadge).toBeInTheDocument()
    expect(screen.queryByText("cri")).not.toBeInTheDocument()
    expect(screen.queryByText("hig")).not.toBeInTheDocument()
    expect(container.querySelector(".sr-only")).toBeNull()
    expect(criticalBadge.className).toContain("tracking-normal")
    expect(criticalBadge.className).toContain("normal-case")
    expect(criticalBadge.className).not.toContain("w-[4ch]")
  })
})
