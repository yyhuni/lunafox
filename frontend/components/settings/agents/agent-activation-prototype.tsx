"use client"

import { useState } from "react"

import { semanticIcons } from "@/components/icons"
import { SearchInput } from "@/components/shared/search-input"
import { SegmentedMetricProgress } from "@/components/shared/metrics/segmented-metric-progress"
import { COMPACT_CONTENT_GUTTER_CLASS, COMPACT_PAGE_SHELL_CLASS, COMPACT_SECTION_STACK_CLASS } from "@/components/shared/layout/page-shell-density"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { getStatusToneTextClass } from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import { narrowFormDialogContentClassName } from "@/lib/ui/overlay-styles"
import { cn } from "@/lib/utils"
import { AgentOverviewSection } from "./agent-overview-section"
import { AGENT_CARD_GRID_CLASS } from "./agent-layout-contract"
import { AgentActivationPrototypeDialog } from "./agent-activation-prototype-dialog"
import {
  INITIAL_PREVIEW_NODES,
  PROTOTYPE_TODAY,
  previewClusterSummary,
  sampleLicense,
  type ActivationPreviewLicense,
  type ActivationPreviewNode,
} from "./agent-activation-prototype-state"

type PreviewScenario = "basic" | "active" | "expired"

export function AgentActivationPrototype() {
  const [scenario, setScenario] = useState<PreviewScenario>("basic")
  const [license, setLicense] = useState<ActivationPreviewLicense | null>(null)
  const [nodes, setNodes] = useState<ActivationPreviewNode[]>(INITIAL_PREVIEW_NODES)
  const [activationOpen, setActivationOpen] = useState(false)
  const [addOpen, setAddOpen] = useState(false)
  const [nodeName, setNodeName] = useState("")
  const [search, setSearch] = useState("")
  const [lastRefreshedAt, setLastRefreshedAt] = useState<Date | null>(null)
  const expired = scenario === "expired"
  const agentLimit = license && !expired ? 10 : 3
  const atLimit = nodes.length >= agentLimit
  const summary = previewClusterSummary(nodes.length, agentLimit)
  const visibleNodes = nodes.filter((node) => `${node.name} ${node.address}`.toLowerCase().includes(search.toLowerCase()))

  function selectScenario(next: PreviewScenario) {
    setScenario(next)
    setLicense(next === "basic" ? null : sampleLicense("LF-DEMO-3M"))
    setSearch("")
    setNodes(next === "expired"
      ? [...INITIAL_PREVIEW_NODES, { id: 4, name: "海外扫描节点", address: "10.0.0.13", cpu: 16, memory: 35 }]
      : INITIAL_PREVIEW_NODES)
  }

  function activate(next: ActivationPreviewLicense) {
    setLicense(next)
    setScenario("active")
  }

  function addNode(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (atLimit || !nodeName.trim()) return
    const id = Math.max(0, ...nodes.map((node) => node.id)) + 1
    setNodes((current) => [...current, { id, name: nodeName.trim(), address: `10.0.0.${10 + id}`, cpu: 8, memory: 21 }])
    setNodeName("")
    setAddOpen(false)
  }

  return (
    <div className={COMPACT_PAGE_SHELL_CLASS}>
      <div className={cn(COMPACT_CONTENT_GUTTER_CLASS, COMPACT_SECTION_STACK_CLASS)}>
        <div className="flex flex-wrap items-center justify-between gap-3 border-b pb-3">
          <div className="flex flex-wrap items-center gap-2">
            <Badge size="compact" variant="outline">交互原型</Badge>
            <span className={textRole.helperText}>仅当前页面生效 · 刷新重置</span>
          </div>
          <div className="flex flex-wrap items-center gap-1" role="group" aria-label="预览授权状态">
            <span className={cn("mr-1", textRole.helperText)}>预览状态</span>
            {([ ["basic", "未激活"], ["active", "已激活"], ["expired", "已到期"] ] as const).map(([value, label]) => (
              <Button key={value} variant={scenario === value ? "secondary" : "ghost"} size="sm" aria-pressed={scenario === value} onClick={() => selectScenario(value)}>{label}</Button>
            ))}
          </div>
        </div>

        <AgentOverviewSection
          summary={summary}
          error={null}
          isInitialError={false}
          isRefetchStale={false}
          lastSuccessfulAt={null}
          lastRefreshedAt={lastRefreshedAt}
          isRefreshing={false}
          onRefresh={() => setLastRefreshedAt(new Date())}
          onRetry={() => undefined}
        />

        <section aria-label="Agent 扩容授权" className="flex flex-col gap-3 border-y py-4 @2xl/main:flex-row @2xl/main:items-center @2xl/main:justify-between">
          <div className="flex min-w-0 items-start gap-3">
            <semanticIcons.concept.apiKey aria-hidden="true" className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
            <div className="min-w-0 space-y-1.5">
              <div className="flex flex-wrap items-center gap-2">
                <h2 className={textRole.sectionTitle}>{expired ? "扩容授权已到期" : license ? "扩容授权已生效" : "需要更多扫描节点？"}</h2>
                <Badge size="compact" variant={expired ? "warning" : license ? "success" : "outline"}>{expired ? "已到期" : license ? "扩容版" : "基础版"}</Badge>
              </div>
              <p className={textRole.bodySubtle}>
                {expired
                  ? `当前恢复为 3 个节点额度，已添加的 ${nodes.length} 个节点予以保留。更新授权后可继续扩容。`
                  : license
                    ? `最多可添加 10 个 Agent · 有效期至 ${license.expiresAt} · ${license.months} 个月授权`
                    : "基础额度为 3 个 Agent。赞助、提交 Bug、参与引擎优化等贡献，均可联系作者获取 Key，激活更多节点额度。"}
              </p>
            </div>
          </div>
          <Button variant={license && !expired ? "outline" : "default"} size="sm" onClick={() => setActivationOpen(true)} className="self-start @2xl/main:self-auto">
            <semanticIcons.concept.apiKey aria-hidden="true" />{expired ? "更新授权" : license ? "管理授权" : "激活扩容"}
          </Button>
        </section>

        <div className="flex flex-wrap items-center justify-between gap-3 pt-1">
          <div className="w-full @xl/main:max-w-xs">
            <SearchInput aria-label="搜索 Agent" placeholder="搜索名称或 IP 地址" value={search} onChange={(event) => setSearch(event.target.value)} />
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <span id="prototype-quota-hint" className={cn(textRole.helperText, atLimit && getStatusToneTextClass("warning"))}>
              {atLimit ? `${nodes.length}/${agentLimit} 个节点 · ${expired ? "更新授权后可继续添加" : "激活扩容后可继续添加"}` : `${nodes.length}/${agentLimit} 个节点 · 还可添加 ${agentLimit - nodes.length} 个`}
            </span>
            <Button size="sm" variant="surface" disabled={atLimit} aria-describedby="prototype-quota-hint" onClick={() => setAddOpen(true)}>
              <semanticIcons.action.add aria-hidden="true" />添加 Agent
            </Button>
          </div>
        </div>

        <div className={AGENT_CARD_GRID_CLASS}>
          {visibleNodes.map((node) => (
            <Card key={node.id} variant="compact">
              <CardHeader>
                <div className="flex items-center justify-between gap-2">
                  <div className="flex min-w-0 items-center gap-2">
                    <semanticIcons.concept.agent aria-hidden="true" className="size-4 shrink-0 text-muted-foreground" />
                    <CardTitle className="truncate">{node.name}</CardTitle>
                  </div>
                  <Badge size="compact" variant="success">健康</Badge>
                </div>
                <p className={textRole.code}>{node.address}</p>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <SegmentedMetricProgress label="CPU" value={node.cpu} threshold={80} barTone="neutral" valueTone="neutral" />
                  <SegmentedMetricProgress label="内存" value={node.memory} threshold={80} barTone="neutral" valueTone="neutral" />
                  <SegmentedMetricProgress label="磁盘" value={24} threshold={80} barTone="neutral" valueTone="neutral" />
                </div>
                <div className="flex items-center justify-between gap-3 border-t pt-3">
                  <p className={textRole.helperText}>任务槽位 <span className={textRole.metadataValue}>0 / 5</span></p>
                  <Button variant="ghost" size="icon-sm" aria-label={`删除 ${node.name}`} onClick={() => setNodes((current) => current.filter((item) => item.id !== node.id))}>
                    <semanticIcons.action.delete aria-hidden="true" />
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
          {!search && (
            <Button variant="outline" size="action-card" layout="actionTile" disabled={atLimit} aria-describedby="prototype-quota-hint" onClick={() => setAddOpen(true)} className="border-dashed">
              <semanticIcons.action.add aria-hidden="true" className="size-5" />
              <span>添加新的 Agent</span>
              <span className={textRole.helperText}>{atLimit ? "激活或更新授权后开放" : "扩展你的扫描覆盖范围"}</span>
            </Button>
          )}
        </div>
        {search && visibleNodes.length === 0 && <p className={cn("py-10 text-center", textRole.bodySubtle)}>没有匹配的 Agent，请尝试其他名称或 IP。</p>}

        <p className={cn("pt-3", textRole.helperText)}>演示日期 {PROTOTYPE_TODAY} · 示例 Key：LF-DEMO-1M / LF-DEMO-2M / LF-DEMO-3M · 节点和授权均为模拟数据</p>
      </div>

      <AgentActivationPrototypeDialog open={activationOpen} onOpenChange={setActivationOpen} currentLimit={agentLimit} renewing={Boolean(license)} onActivate={activate} />

      <Dialog open={addOpen} onOpenChange={setAddOpen}>
        <DialogContent className={narrowFormDialogContentClassName}>
          <DialogHeader className="pr-8 text-left">
            <DialogTitle>添加演示 Agent</DialogTitle>
            <DialogDescription>当前还可添加 {Math.max(0, agentLimit - nodes.length)} 个节点。这里可以体验扩容后的添加流程。</DialogDescription>
          </DialogHeader>
          <form onSubmit={addNode} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="preview-node-name">节点名称</Label>
              {/* Field names must not shadow DOM properties read by dialog focus management. */}
              <Input id="preview-node-name" name="displayName" required maxLength={80} placeholder="例如：海外扫描节点" value={nodeName} onChange={(event) => setNodeName(event.target.value)} />
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setAddOpen(false)}>取消</Button>
              <Button type="submit" disabled={atLimit || !nodeName.trim()}>添加节点</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
