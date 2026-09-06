import React from "react"
import { render, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import {
  RuntimeHeader,
  ScanRuntimeDetailDrawer,
} from "@/components/scan/history/scan-runtime-detail-drawer"
import { agentService } from "@/services/agent.service"
import type { ScanRecord } from "@/types/scan.types"

const runtimeState = vi.hoisted(() => ({ scan: null as unknown }))

vi.mock("@/hooks/use-scan-workflows", () => ({
  useScanWorkflows: () => ({ data: [] }),
}))

vi.mock("@/hooks/use-scan-executed-engine-display", () => ({
  useScanExecutedEngineDisplay: () => ({
    isLoading: false,
    error: null,
    engineNamesById: new Map(),
    engineDescriptionsById: new Map(),
  }),
}))

vi.mock("@/components/scan/history/scan-overview-state", () => ({
  useScanOverviewState: () => ({
    scan: runtimeState.scan,
    isLoading: false,
    error: null,
    logs: [],
    logsLoading: false,
  }),
}))

const labels: Record<string, string> = {
  "runtimeDrawer.meta.createdAt": "创建时间",
  "runtimeDrawer.meta.assignment": "分配方式",
  "runtimeDrawer.meta.automatic": "自动分配",
  "runtimeDrawer.meta.pinned": "指定节点",
  "runtimeDrawer.meta.agent": "执行节点",
  "runtimeDrawer.meta.online": "在线",
  "runtimeDrawer.meta.offline": "离线",
  "runtimeDrawer.meta.deleted": "已删除",
  "runtimeDrawer.meta.stateUnavailable": "当前状态不可用",
  "runtimeDrawer.meta.healthy": "健康",
  "runtimeDrawer.meta.warning": "预警",
  "runtimeDrawer.meta.healthUnknown": "健康状态未知",
  "inputSource.label": "扫描输入",
  "inputSource.scanSnapshot": "扫描快照",
  "inputSource.targetInventory": "目标当前资产",
  "runtimeDrawer.summary.duration": "运行时长",
}

const translate = ((key: string) => labels[key] ?? key) as React.ComponentProps<typeof RuntimeHeader>["t"]
const statusLabel = ((status: string) => `扫描状态:${status}`) as React.ComponentProps<typeof RuntimeHeader>["statusLabel"]

function scanFixture(overrides: Partial<ScanRecord> = {}): ScanRecord {
  return {
    id: 42,
    targetId: 7,
    target: {
      id: 7,
      name: "targets/7",
      displayName: "example.com",
      type: "domain",
    },
    plannedEngineIds: [],
    triggerType: "manual",
    inputSource: "scanSnapshot",
    createdAt: "2026-08-04T08:00:00Z",
    status: "succeeded",
    progress: 100,
    agentId: 5001,
    agent: "agents/5001",
    agentName: "renamed-agent",
    agentStatus: "online",
    agentHealthState: "healthy",
    assignmentMode: "pinned",
    agentDeleted: false,
    runtimeTasks: [],
    ...overrides,
  }
}

function renderHeader(scan: ScanRecord) {
  return render(
    <RuntimeHeader
      scan={scan}
      tasks={[]}
      now={new Date("2026-08-04T08:05:00Z")}
      locale="zh"
      t={translate}
      statusLabel={statusLabel}
    />
  )
}

describe("RuntimeHeader Agent projection", () => {
  beforeEach(() => {
    runtimeState.scan = null
    vi.restoreAllMocks()
  })

  it("renders the current renamed Agent and pinned runtime states from Scan detail", () => {
    renderHeader(scanFixture())

    expect(screen.getByText("renamed-agent")).toBeInTheDocument()
    expect(screen.queryByText("agents/5001")).not.toBeInTheDocument()
    expect(screen.getByText("指定节点")).toBeInTheDocument()
    expect(screen.getByText("在线")).toBeInTheDocument()
    expect(screen.getByText("健康")).toBeInTheDocument()
  })

  it("renders the persisted input source in runtime details", () => {
    renderHeader(scanFixture({ inputSource: "targetInventory" }))

    expect(screen.getByText("扫描输入")).toBeInTheDocument()
    expect(screen.getByText("目标当前资产")).toBeInTheDocument()
  })

  it("retains the canonical reference and omits current state after Agent deletion", () => {
    renderHeader(scanFixture({
      agentDeleted: true,
      agentName: undefined,
      agentStatus: undefined,
      agentHealthState: undefined,
    }))

    expect(screen.getByText("agents/5001")).toBeInTheDocument()
    expect(screen.getByText("已删除")).toBeInTheDocument()
    expect(screen.queryByText("renamed-agent")).not.toBeInTheDocument()
    expect(screen.queryByText("在线")).not.toBeInTheDocument()
    expect(screen.queryByText("健康")).not.toBeInTheDocument()
  })

  it("does not infer offline when current runtime state is unavailable", () => {
    renderHeader(scanFixture({
      assignmentMode: "automatic",
      agentStatus: undefined,
      agentHealthState: undefined,
    }))

    expect(screen.getByText("自动分配")).toBeInTheDocument()
    expect(screen.getByText("renamed-agent")).toBeInTheDocument()
    expect(screen.getByText("当前状态不可用")).toBeInTheDocument()
    expect(screen.queryByText("离线")).not.toBeInTheDocument()
  })

  it("renders an unassigned Scan without fabricating an execution node", () => {
    renderHeader(scanFixture({
      agentId: undefined,
      agent: undefined,
      agentName: undefined,
      agentStatus: undefined,
      agentHealthState: undefined,
      assignmentMode: "automatic",
      agentDeleted: undefined,
    }))

    expect(screen.getByText("自动分配")).toBeInTheDocument()
    expect(screen.queryByText("执行节点")).not.toBeInTheDocument()
    expect(screen.queryByText("当前状态不可用")).not.toBeInTheDocument()
  })

  it("opens detail for an Agent outside the first page without requesting the Agent collection", async () => {
    const getAgents = vi.spyOn(agentService, "getAgents")
    const scan = scanFixture({ agentName: "page-51-agent" })
    runtimeState.scan = scan

    render(<ScanRuntimeDetailDrawer open onOpenChange={vi.fn()} scan={scan} />)

    await waitFor(() => expect(screen.getByText("page-51-agent")).toBeInTheDocument())
    expect(getAgents).not.toHaveBeenCalled()
  })
})
