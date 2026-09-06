import React from "react"
import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { RuntimeTaskList } from "@/components/scan/history/scan-runtime-detail-drawer"
import type { RuntimeTaskItem } from "@/components/scan/history/scan-runtime-detail-utils"

const labels: Record<string, string> = {
  "runtimeDrawer.tasks.title": "Runtime tasks",
  "runtimeDrawer.tasks.expand": "Expand task details",
  "runtimeDrawer.tasks.duration": "Duration",
  "runtimeDrawer.tasks.startedAt": "Started",
  "runtimeDrawer.tasks.noProgress": "No task progress",
  "runtimeDrawer.tasks.failureTitle": "Failure",
  "runtimeDrawer.tasks.diagnostics.title": "Result Delivery Diagnostics",
  "runtimeDrawer.tasks.diagnostics.availability": "Diagnostic availability",
  "runtimeDrawer.tasks.diagnostics.available": "Available",
  "runtimeDrawer.tasks.diagnostics.unavailable": "Unavailable",
  "runtimeDrawer.tasks.diagnostics.unavailableDescription": "The Engine did not provide a verified terminal diagnostic.",
  "runtimeDrawer.tasks.diagnostics.resultState": "Result delivery",
  "runtimeDrawer.tasks.diagnostics.failedStage": "Failure boundary",
  "runtimeDrawer.tasks.diagnostics.errorType": "Error classification",
  "runtimeDrawer.tasks.diagnostics.watermarks": "Delivery watermarks by result type",
  "runtimeDrawer.tasks.diagnostics.zeroResultComplete": "No typed result batches were submitted.",
  "runtimeDrawer.tasks.diagnostics.noWatermarks": "No per-result-type delivery watermarks were reported.",
  "runtimeDrawer.tasks.diagnostics.receivedItems": "Received",
  "runtimeDrawer.tasks.diagnostics.encodedItems": "Encoded",
  "runtimeDrawer.tasks.diagnostics.submittedItems": "Submitted",
  "runtimeDrawer.tasks.diagnostics.acknowledgedItems": "Acknowledged",
  "runtimeDrawer.tasks.diagnostics.submittedBatches": "Submitted batches",
  "runtimeDrawer.tasks.diagnostics.acknowledgedBatches": "Acknowledged batches",
  "runtimeDrawer.tasks.diagnostics.resultStates.complete": "Delivery complete",
  "runtimeDrawer.tasks.diagnostics.resultStates.partial": "Partially confirmed delivery",
  "runtimeDrawer.tasks.diagnostics.resultStates.unknown": "Delivery state unknown",
  "runtimeDrawer.tasks.diagnostics.failedStages.result_submit": "Result submission",
  "runtimeDrawer.tasks.diagnostics.errorTypes.result_submit_failed": "Result submission failed",
}

const translate = (key: string) => labels[key] ?? key

function taskFixture(overrides: Partial<RuntimeTaskItem> = {}): RuntimeTaskItem {
  return {
    id: 1,
    title: "Port scan",
    status: "succeeded",
    detail: "Inspect reachable ports.",
    order: 0,
    ...overrides,
  }
}

function renderTasks(tasks: RuntimeTaskItem[]) {
  return render(
    <RuntimeTaskList
      tasks={tasks}
      taskProgressLogsByTaskId={new Map()}
      now={new Date("2026-08-17T00:00:00Z")}
      t={translate as React.ComponentProps<typeof RuntimeTaskList>["t"]}
      statusLabel={(status) => status}
    />
  )
}

function renderTask(task: RuntimeTaskItem) {
  return renderTasks([task])
}

describe("RuntimeTaskList terminal diagnostics", () => {
  it("renders a legal zero-result complete snapshot without claiming no assets were found", () => {
    renderTask(taskFixture({
      diagnostics: {
        compatibilityRevision: "engine-execution-diagnostics-r1",
        availability: "available",
        resultState: "complete",
        resultTypeWatermarks: [],
      },
    }))

    expect(screen.queryByText("Result Delivery Diagnostics")).not.toBeInTheDocument()
    fireEvent.click(screen.getByLabelText("Expand task details"))

    expect(screen.getByText("Result Delivery Diagnostics")).toBeInTheDocument()
    expect(screen.getByText("Available")).toBeInTheDocument()
    expect(screen.getByText("Delivery complete")).toBeInTheDocument()
    expect(screen.getByText("No typed result batches were submitted.")).toBeInTheDocument()
    expect(screen.queryByText(/no assets were found/i)).not.toBeInTheDocument()
  })

  it("renders partial confirmed delivery from the Server watermarks", () => {
    renderTask(taskFixture({
      status: "failed",
      diagnostics: {
        compatibilityRevision: "engine-execution-diagnostics-r1",
        availability: "available",
        resultState: "partial",
        failedStage: "result_submit",
        errorType: "result_submit_failed",
        resultTypeWatermarks: [{
          resultType: "network.port",
          receivedItems: 4,
          encodedItems: 4,
          submittedItems: 4,
          acknowledgedItems: 2,
          submittedBatches: 2,
          acknowledgedBatches: 1,
        }],
      },
    }))

    expect(screen.getByText("Partially confirmed delivery")).toBeInTheDocument()
    expect(screen.getByText("Result submission")).toBeInTheDocument()
    expect(screen.getByText("Result submission failed")).toBeInTheDocument()
    expect(screen.getByText("network.port")).toBeInTheDocument()
    expect(screen.getByText("Acknowledged")).toBeInTheDocument()
    expect(screen.getAllByText("2")).toHaveLength(2)
  })

  it("shows unavailable evidence without inventing counters or a root cause", () => {
    renderTask(taskFixture({
      status: "failed",
      diagnostics: {
        compatibilityRevision: "engine-execution-diagnostics-r1",
        availability: "unavailable",
        resultState: "unknown",
      },
    }))

    expect(screen.getByText("Unavailable")).toBeInTheDocument()
    expect(screen.getByText("Delivery state unknown")).toBeInTheDocument()
    expect(screen.getByText("The Engine did not provide a verified terminal diagnostic.")).toBeInTheDocument()
    expect(screen.queryByText("Delivery watermarks by result type")).not.toBeInTheDocument()
  })

  it("default-opens only the first failed task and remains user-collapsible", () => {
    renderTasks([
      taskFixture({ id: 1, title: "Pending task", status: "pending" }),
      taskFixture({ id: 2, title: "First failure", status: "failed", failureSummary: "First failure summary" }),
      taskFixture({ id: 3, title: "Second failure", status: "failed", failureSummary: "Second failure summary" }),
    ])

    expect(screen.getByText("First failure summary")).toBeInTheDocument()
    expect(screen.queryByText("Second failure summary")).not.toBeInTheDocument()

    fireEvent.click(screen.getAllByLabelText("Expand task details")[1]!)
    expect(screen.queryByText("First failure summary")).not.toBeInTheDocument()
  })
})
