import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { useRef } from "react"
import { afterEach, describe, expect, it, vi } from "vitest"

import {
  formatLogTime,
  formatStructuredLogLine,
  StructuredLogViewer,
  type StructuredLogLineItem,
} from "../structured-log-viewer"

const timestamp = "2026-07-24T12:09:10.375Z"

function makeLine(overrides: Partial<StructuredLogLineItem>): StructuredLogLineItem {
  return {
    id: "line-1",
    ts: timestamp,
    stream: "stdout",
    line: "",
    truncated: false,
    ...overrides,
  }
}

function ViewerHarness({ lines }: { lines: StructuredLogLineItem[] }) {
  const viewportRef = useRef<HTMLDivElement>(null)

  return (
    <div ref={viewportRef} data-testid="log-viewport">
      <StructuredLogViewer
        viewportRef={viewportRef}
        lines={lines}
        empty="empty"
        truncatedLabel="truncated"
      />
    </div>
  )
}

describe("StructuredLogViewer", () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it("renders parsed entries as one continuous copy-friendly text stream", () => {
    const lines = [
      makeLine({
        id: "line-1",
        line: JSON.stringify({
          level: "info",
          msg: "scan started",
          caller: "scan/runner.go:42",
          taskId: "tasks/1",
        }),
      }),
      makeLine({
        id: "line-2",
        stream: "stderr",
        line: "command failed",
        truncated: true,
      }),
    ]
    const { container } = render(<ViewerHarness lines={lines} />)
    const viewer = container.querySelector('[data-slot="structured-log-viewer"]')

    expect(viewer?.textContent).toBe(
      `[${formatLogTime(timestamp)}] [INFO] scan started scan/runner.go:42 taskId=tasks/1\n` +
        `[${formatLogTime(timestamp)}] command failed [truncated]`
    )
    expect(lines.map((line) => formatStructuredLogLine(line)).join("\n")).toBe(viewer?.textContent)
  })

  it("keeps HTML-like structured values as inert text", () => {
    const payload = "<script>unsafe()</script>"
    const { container } = render(
      <StructuredLogViewer
        viewportRef={{ current: document.createElement("div") }}
        lines={[makeLine({ line: JSON.stringify({ level: "error", msg: payload }) })]}
        empty="empty"
      />
    )
    const viewer = container.querySelector('[data-slot="structured-log-viewer"]')

    expect(viewer?.querySelector("script")).toBeNull()
    expect(viewer?.textContent).toContain(`[ERROR] ${payload}`)
  })

  it("keeps a 5000-line window in memory while mounting only the visible range", async () => {
    vi.spyOn(HTMLElement.prototype, "offsetHeight", "get").mockImplementation(function (this: HTMLElement) {
      return this.dataset.testid === "log-viewport" ? 200 : 20
    })
    vi.spyOn(HTMLElement.prototype, "offsetWidth", "get").mockReturnValue(800)
    vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockImplementation(function (this: HTMLElement) {
      const height = this.dataset.testid === "log-viewport" ? 200 : 20
      const top = this.dataset.slot === "structured-log-viewer"
        ? -(this.parentElement?.scrollTop ?? 0)
        : 0
      return {
        x: 0,
        y: top,
        top,
        left: 0,
        right: 800,
        bottom: top + height,
        width: 800,
        height,
        toJSON: () => ({}),
      }
    })
    const lines = Array.from({ length: 5000 }, (_, index) => makeLine({
      id: `line-${index}`,
      line: JSON.stringify({ level: "info", msg: `entry-${index}` }),
    }))

    const { container, rerender } = render(<ViewerHarness lines={lines} />)
    rerender(<ViewerHarness lines={lines} />)

    await waitFor(() => {
      expect(container.querySelectorAll('[data-slot="structured-log-line"]').length).toBeGreaterThan(0)
    })
    expect(container.querySelector('[data-slot="structured-log-viewer"]')).toHaveAttribute("data-total-lines", "5000")
    expect(container.querySelectorAll('[data-slot="structured-log-line"]').length).toBeLessThan(100)
    expect(container.textContent).toContain("entry-0")

    const viewport = screen.getByTestId("log-viewport")
    Object.defineProperty(viewport, "scrollTop", { configurable: true, value: 40_000, writable: true })
    fireEvent.scroll(viewport)

    await waitFor(() => {
      const firstMountedIndex = Number(
        container.querySelector('[data-slot="structured-log-line"]')?.getAttribute("data-index"),
      )
      expect(firstMountedIndex).toBeGreaterThan(0)
    })
    const firstMountedLine = container.querySelector('[data-slot="structured-log-line"]')
    expect(firstMountedLine).toHaveTextContent(`entry-${firstMountedLine?.getAttribute("data-index")}`)
    expect(container.querySelectorAll('[data-slot="structured-log-line"]').length).toBeLessThan(100)
  })

})
