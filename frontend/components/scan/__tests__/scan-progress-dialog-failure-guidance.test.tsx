import { render, screen } from "@testing-library/react"
import React from "react"
import { describe, expect, it, vi } from "vitest"

import { ScanProgressSummary } from "@/components/scan/scan-progress-dialog-sections"
import {
  NUCLEI_POC_FAILURE_KIND,
  NUCLEI_POC_MANAGEMENT_HREF,
} from "@/components/scan/scan-progress-dialog-utils"
import type { ScanProgressData } from "@/components/scan/scan-progress-dialog-types"

vi.mock("next/link", () => ({
  default: ({ href, children, ...rest }: { href: string; children?: React.ReactNode }) => (
    <a href={href} {...rest}>{children}</a>
  ),
}))

const messages: Record<string, string> = {
  "failureGuidance.pocTitle": "漏洞扫描 POC 模板尚未就绪",
  "failureGuidance.pocDescription": "漏洞扫描任务在准备 Nuclei POC 模板时失败，常见原因是 POC 尚未同步或尚未启用。",
  "failureGuidance.relaunchHint": "如果已启用 POC，请检查资源后重试；准备好 POC 后，请重新发起扫描。",
  "failureGuidance.openPocCatalog": "前往 Nuclei POC",
  "failureGuidance.rawDetail": "原始错误详情",
  errorReason: "错误原因",
  target: "目标",
  workflow: "工作流",
  startTime: "开始时间",
  status: "状态",
}

const t = (key: string) => messages[key] ?? key

function progressData(overrides: Partial<ScanProgressData> = {}): ScanProgressData {
  return {
    id: 21,
    scanWorkflow: "scanWorkflows/default",
    plannedEngineIds: [],
    status: "failed",
    progress: 100,
    stages: [],
    ...overrides,
  }
}

describe("ScanProgressSummary failure guidance", () => {
  it("renders actionable POC guidance and the catalog link for no_enabled_templates", () => {
    render(
      <ScanProgressSummary
        data={progressData({
          failureKind: NUCLEI_POC_FAILURE_KIND,
          failureGuidance: { kind: NUCLEI_POC_FAILURE_KIND, targetHref: NUCLEI_POC_MANAGEMENT_HREF },
          errorMessage: "The runtime artifact required by this task is unavailable.",
        })}
        engineNames={[]}
        locale="zh"
        t={t}
      />
    )

    expect(screen.getByText("漏洞扫描 POC 模板尚未就绪")).toBeDefined()
    expect(screen.getByText(/常见原因是 POC 尚未同步或尚未启用/)).toBeDefined()
    expect(screen.getByText(/请重新发起扫描/)).toBeDefined()

    const link = screen.getByRole("link", { name: "前往 Nuclei POC" })
    expect(link.getAttribute("href")).toBe(NUCLEI_POC_MANAGEMENT_HREF)

    expect(screen.getByText("原始错误详情")).toBeDefined()
    expect(screen.getByText("The runtime artifact required by this task is unavailable.")).toBeDefined()
  })

  it("does not offer retry or automatic preparation affordances", () => {
    render(
      <ScanProgressSummary
        data={progressData({
          failureKind: NUCLEI_POC_FAILURE_KIND,
          failureGuidance: { kind: NUCLEI_POC_FAILURE_KIND, targetHref: NUCLEI_POC_MANAGEMENT_HREF },
          errorMessage: "The runtime artifact required by this task is unavailable.",
        })}
        engineNames={[]}
        locale="zh"
        t={t}
      />
    )

    expect(screen.queryByRole("button")).toBeNull()
    expect(screen.queryByText(/自动重试/)).toBeNull()
  })

  it("keeps the plain error rendering for other failure kinds", () => {
    render(
      <ScanProgressSummary
        data={progressData({
          failureKind: "engine_exit_failed",
          failureGuidance: null,
          errorMessage: "engine exited with code 1",
        })}
        engineNames={[]}
        locale="zh"
        t={t}
      />
    )

    expect(screen.getByText("错误原因")).toBeDefined()
    expect(screen.getByText("engine exited with code 1")).toBeDefined()
    expect(screen.queryByRole("link")).toBeNull()
    expect(screen.queryByText("漏洞扫描 POC 模板尚未就绪")).toBeNull()
  })
})
