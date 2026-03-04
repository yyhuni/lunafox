import type { ReactNode } from "react"
import {
  AlertTriangle,
  CheckCircle2,
  ChevronRight,
  Play,
  Settings,
  Zap,
} from "@/components/icons"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { cn } from "@/lib/utils"

const yamlSample = `subdomain_discovery:
  depth: 2
  dns: true
port_scan:
  ports: top-1000
  rate: 80
site_scan:
  crawler_depth: 2
vuln_scan:
  enable: true`

const presets = [
  {
    id: "full",
    title: "全量扫描",
    desc: "覆盖资产发现到漏洞检测的全流程，适合首次评估",
    engines: 3,
    caps: 6,
    tag: "推荐",
    active: true,
  },
  {
    id: "recon",
    title: "信息收集",
    desc: "快速摸清资产结构，生成扫描基线",
    engines: 2,
    caps: 4,
  },
  {
    id: "vuln",
    title: "漏洞扫描",
    desc: "对已知资产进行漏洞核验",
    engines: 1,
    caps: 2,
  },
  {
    id: "custom",
    title: "自定义",
    desc: "手动选择工作流并编辑配置",
    engines: 0,
    caps: 0,
  },
]

const engineChips = ["子域名发现", "端口扫描", "站点探测", "漏洞扫描"]

function DialogShell({ children }: { children: ReactNode }) {
  return (
    <div className="rounded-xl border bg-background shadow-sm overflow-hidden">
      {children}
    </div>
  )
}

function Header() {
  return (
    <div className="flex flex-col gap-4 border-b px-6 py-5 sm:flex-row sm:items-start sm:justify-between">
      <div className="flex items-start gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
          <Play className="h-5 w-5" />
        </div>
        <div>
          <h2 className="text-lg font-semibold">发起扫描</h2>
          <p className="text-sm text-muted-foreground">为目标 <span className="text-foreground font-medium">acme.com</span> 发起扫描</p>
        </div>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant="secondary">推荐：全量扫描</Badge>
        <Button variant="outline" size="sm">取消</Button>
        <Button size="sm">快速启动</Button>
      </div>
    </div>
  )
}

function PresetGrid() {
  return (
    <div className="grid gap-3 sm:grid-cols-2">
      {presets.map((preset) => (
        <div
          key={preset.id}
          className={cn(
            "rounded-lg border p-4 transition",
            preset.active
              ? "border-primary/60 bg-primary/5"
              : "border-border bg-muted/20 hover:border-primary/40"
          )}
        >
          <div className="flex items-center justify-between">
            <p className="font-medium">{preset.title}</p>
            {preset.tag && <Badge variant="secondary">{preset.tag}</Badge>}
          </div>
          <p className="text-xs text-muted-foreground mt-2 leading-relaxed">{preset.desc}</p>
          <p className="text-xs text-muted-foreground mt-3">{preset.engines} 个工作流 · {preset.caps} 项能力</p>
        </div>
      ))}
    </div>
  )
}

export default function ScanDialogPrototypesPage() {
  return (
    <div className="space-y-10 p-6">
      <div className="space-y-2">
        <h1 className="text-xl font-semibold">发起扫描弹窗 · 原型对齐当前接口</h1>
        <p className="text-sm text-muted-foreground">以下三套方案基于现有“发起扫描”接口字段（目标/工作流/配置/快速启动）与现有 UI 规范。</p>
      </div>

      {/* Variant A */}
      <section className="space-y-3">
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Badge variant="secondary">方案 A</Badge>
          <span>双栏摘要，决策效率优先</span>
        </div>
        <DialogShell>
          <Header />
          <div className="grid gap-6 p-6 lg:grid-cols-[1.2fr_0.8fr]">
            <div className="space-y-5">
              <div className="space-y-2">
                <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">扫描方案</p>
                <PresetGrid />
              </div>
              <div className="space-y-2">
                <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">使用工作流</p>
                <div className="flex flex-wrap gap-2">
                  {engineChips.map((chip) => (
                    <Badge key={chip} variant="outline">{chip}</Badge>
                  ))}
                </div>
              </div>
              <div className="rounded-lg border bg-muted/20">
                <div className="flex items-center justify-between px-4 py-3">
                  <div className="flex items-center gap-2 text-sm">
                    <Settings className="h-4 w-4 text-muted-foreground" />
                    高级配置（可选）
                  </div>
                  <Badge variant="outline">已编辑</Badge>
                </div>
                <Separator />
                <pre className="px-4 py-3 text-xs leading-relaxed text-muted-foreground whitespace-pre-wrap">{yamlSample}</pre>
              </div>
            </div>
            <div className="space-y-4">
              <Card>
                <CardHeader>
                  <CardTitle>执行摘要</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3 text-sm">
                  <div className="flex items-center justify-between">
                    <span className="text-muted-foreground">预计耗时</span>
                    <span>约 18 分钟</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-muted-foreground">工作流数量</span>
                    <span>3 个</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-muted-foreground">能力覆盖</span>
                    <span>6 项</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-muted-foreground">输出类型</span>
                    <span>子域名 / 站点 / 漏洞</span>
                  </div>
                </CardContent>
              </Card>
              <div className="flex items-start gap-2 rounded-lg border border-amber-500/30 bg-amber-500/10 p-3 text-xs text-amber-700 dark:text-amber-300">
                <AlertTriangle className="h-4 w-4 mt-0.5" />
                切换扫描方案会覆盖当前 YAML 配置，建议先保存。
              </div>
            </div>
          </div>
          <div className="flex flex-col gap-3 border-t px-6 py-4 sm:flex-row sm:items-center sm:justify-between">
            <span className="text-sm text-muted-foreground">已选择 3 个工作流 · YAML 校验通过</span>
            <div className="flex gap-2">
              <Button variant="outline">保存为计划任务</Button>
              <Button>开始扫描</Button>
            </div>
          </div>
        </DialogShell>
      </section>

      {/* Variant B */}
      <section className="space-y-3">
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Badge variant="secondary">方案 B</Badge>
          <span>向导式流程，强调校验与步骤</span>
        </div>
        <DialogShell>
          <Header />
          <div className="space-y-4 p-6">
            <div className="flex flex-wrap items-center gap-3 text-sm">
              <span className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-primary/40 bg-primary/10 text-primary">1</span>
              <span>选择方案</span>
              <ChevronRight className="h-4 w-4 text-muted-foreground" />
              <span className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-primary/40 bg-primary/10 text-primary">2</span>
              <span className="text-primary">校验配置</span>
              <ChevronRight className="h-4 w-4 text-muted-foreground" />
              <span className="inline-flex h-7 w-7 items-center justify-center rounded-full border">3</span>
              <span>启动扫描</span>
            </div>
            <div className="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
              <div className="space-y-3">
                <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">配置预览</p>
                <div className="rounded-lg border bg-muted/20">
                  <div className="flex items-center justify-between px-4 py-3 text-sm">
                    <span>YAML 配置</span>
                    <Badge variant="outline">未发现语法错误</Badge>
                  </div>
                  <Separator />
                  <pre className="px-4 py-3 text-xs leading-relaxed text-muted-foreground whitespace-pre-wrap">{yamlSample}</pre>
                </div>
                <div className="flex flex-wrap gap-2">
                  {engineChips.map((chip) => (
                    <Badge key={chip} variant="outline">{chip}</Badge>
                  ))}
                </div>
              </div>
              <div className="space-y-3">
                <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">校验结果</p>
                <div className="rounded-lg border bg-muted/20 p-4 space-y-3 text-sm">
                  <div className="flex items-start gap-2">
                    <CheckCircle2 className="h-4 w-4 text-emerald-500 mt-0.5" />
                    <div>
                      <p className="font-medium">基础校验通过</p>
                      <p className="text-xs text-muted-foreground">YAML 语法、重复 key、必填字段均有效</p>
                    </div>
                  </div>
                  <div className="flex items-start gap-2">
                    <CheckCircle2 className="h-4 w-4 text-emerald-500 mt-0.5" />
                    <div>
                      <p className="font-medium">已选择 3 个工作流</p>
                      <p className="text-xs text-muted-foreground">全量扫描覆盖 6 项能力</p>
                    </div>
                  </div>
                  <div className="flex items-start gap-2">
                    <AlertTriangle className="h-4 w-4 text-amber-500 mt-0.5" />
                    <div>
                      <p className="font-medium">发现 1 条建议</p>
                      <p className="text-xs text-muted-foreground">端口扫描速率偏高，建议降低以避免目标异常</p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div className="flex flex-col gap-3 border-t px-6 py-4 sm:flex-row sm:items-center sm:justify-between">
            <span className="text-sm text-muted-foreground">步骤 2/3 · 配置已校验</span>
            <div className="flex gap-2">
              <Button variant="outline">上一步</Button>
              <Button>下一步</Button>
            </div>
          </div>
        </DialogShell>
      </section>

      {/* Variant C */}
      <section className="space-y-3">
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Badge variant="secondary">方案 C</Badge>
          <span>紧凑快启，侧重快速启动与预设切换</span>
        </div>
        <DialogShell>
          <div className="flex flex-col gap-4 border-b px-6 py-5 sm:flex-row sm:items-start sm:justify-between">
            <div className="flex items-start gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <Zap className="h-5 w-5" />
              </div>
              <div>
                <h2 className="text-lg font-semibold">发起扫描</h2>
                <p className="text-sm text-muted-foreground">目标 <span className="text-foreground font-medium">api.redfox.io</span> · 优先级 P1</p>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <Button variant="outline" size="sm">取消</Button>
              <Button size="sm">快速启动</Button>
            </div>
          </div>
          <div className="grid gap-6 p-6 lg:grid-cols-[1.1fr_0.9fr]">
            <div className="space-y-5">
              <div className="space-y-2">
                <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">方案切换</p>
                <div className="flex flex-wrap gap-2">
                  {presets.map((preset) => (
                    <Button
                      key={preset.id}
                      variant={preset.active ? "default" : "outline"}
                      size="sm"
                    >
                      {preset.title}
                    </Button>
                  ))}
                </div>
              </div>
              <div className="rounded-lg border bg-muted/20 p-4 space-y-3">
                <div className="flex items-center justify-between text-sm">
                  <span>快速启动说明</span>
                  <Badge variant="outline">仅预设可用</Badge>
                </div>
                <p className="text-xs text-muted-foreground">快速启动会直接使用预设配置启动，不再进行 YAML 校验，适合紧急扫描。</p>
              </div>
              <div className="rounded-lg border bg-muted/20">
                <div className="flex items-center justify-between px-4 py-3 text-sm">
                  <span>配置预览</span>
                  <Badge variant="outline">未编辑</Badge>
                </div>
                <Separator />
                <pre className="px-4 py-3 text-xs leading-relaxed text-muted-foreground whitespace-pre-wrap">{yamlSample}</pre>
              </div>
            </div>
            <div className="space-y-4">
              <Card>
                <CardHeader>
                  <CardTitle>执行策略</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3 text-sm">
                  <div className="flex items-center justify-between">
                    <span className="text-muted-foreground">工作流数量</span>
                    <span>2 个</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-muted-foreground">节点选择</span>
                    <span>默认扫描节点</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-muted-foreground">速率策略</span>
                    <span>平衡模式</span>
                  </div>
                </CardContent>
              </Card>
              <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 p-3 text-xs text-amber-700 dark:text-amber-300 flex items-start gap-2">
                <AlertTriangle className="h-4 w-4 mt-0.5" />
                快速启动默认跳过 YAML 校验，建议在非紧急任务使用标准启动。
              </div>
            </div>
          </div>
          <div className="flex flex-col gap-3 border-t px-6 py-4 sm:flex-row sm:items-center sm:justify-between">
            <span className="text-sm text-muted-foreground">默认方案：全量扫描 · 已选 2 工作流</span>
            <div className="flex gap-2">
              <Button variant="outline">保存为默认方案</Button>
              <Button>开始扫描</Button>
            </div>
          </div>
        </DialogShell>
      </section>
    </div>
  )
}
