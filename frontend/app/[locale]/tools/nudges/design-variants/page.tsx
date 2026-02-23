"use client"

import * as React from "react"
import { toast } from "sonner"
import { IconTerminal, IconDashboard, IconCpu, IconFileText } from "@/components/icons"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { PageHeader } from "@/components/common/page-header"

import { NudgeToastCard } from "@/components/nudges/nudge-toast-card"
import { NudgeTerminal } from "@/components/nudges/nudge-minimal"
import { NudgeGlass } from "@/components/nudges/nudge-banner"
import { NudgeReport } from "@/components/nudges/nudge-playful"

const DEMO_DATA = {
  title: "设计风格预览",
  description: "这是一条测试消息，用于展示不同 UI 风格在实际环境中的表现效果。",
  icon: "🎨",
  primaryAction: { label: "查看详情", onClick: () => console.log("primary") },
  secondaryAction: { label: "忽略", onClick: () => console.log("secondary") },
}

const NUDGE_TOAST_ID = "nudge-singleton"

export default function NudgeDesignVariantsPage() {
  const triggerStandard = () => {
    toast.custom(() => (
      <NudgeToastCard
        {...DEMO_DATA}
        icon={<span className="text-2xl">{DEMO_DATA.icon}</span>}
        onDismiss={() => toast.dismiss(NUDGE_TOAST_ID)}
      />
    ), { id: NUDGE_TOAST_ID, duration: 8000, position: "bottom-right" })
  }

  const triggerTerminal = () => {
    toast.custom(() => (
      <NudgeTerminal
        {...DEMO_DATA}
        icon={<span className="text-xl">{DEMO_DATA.icon}</span>}
        onDismiss={() => toast.dismiss(NUDGE_TOAST_ID)}
      />
    ), { id: NUDGE_TOAST_ID, duration: 8000, position: "bottom-right" })
  }

  const triggerGlass = () => {
    toast.custom(() => (
      <NudgeGlass
        {...DEMO_DATA}
        icon={<span className="text-xl">{DEMO_DATA.icon}</span>}
        onDismiss={() => toast.dismiss(NUDGE_TOAST_ID)}
      />
    ), { id: NUDGE_TOAST_ID, duration: 8000, position: "bottom-right" })
  }

  const triggerReport = () => {
    toast.custom(() => (
      <NudgeReport
        {...DEMO_DATA}
        icon={<span className="text-2xl">{DEMO_DATA.icon}</span>}
        onDismiss={() => toast.dismiss(NUDGE_TOAST_ID)}
      />
    ), { id: NUDGE_TOAST_ID, duration: 8000, position: "bottom-right" })
  }

  return (
    <div className="flex h-screen flex-col bg-background">
      <PageHeader
        title="Nudge Design Variants"
        description="探索不同风格的通知弹窗设计"
        breadcrumbItems={[
          { label: "Tools", href: "/tools" },
          { label: "Nudges", href: "/tools/nudges" },
          { label: "Variants", href: "/tools/nudges/design-variants" },
        ]}
      />

      <div className="flex-1 overflow-auto p-6 md:p-8">
        <div className="mx-auto max-w-5xl grid gap-6 md:grid-cols-2">

          {/* Standard */}
          <Card className="flex flex-col">
            <CardHeader>
              <div className="mb-2 flex size-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <IconDashboard className="size-6" />
              </div>
              <CardTitle>Standard / 标准版</CardTitle>
              <CardDescription>
                当前使用的默认风格。采用卡片式设计，带有显眼的图标和操作区，适合大多数场景。
              </CardDescription>
            </CardHeader>
            <CardContent className="mt-auto pt-6">
              <Button onClick={triggerStandard} className="w-full">
                预览效果 (Bottom Right)
              </Button>
            </CardContent>
          </Card>

          {/* Terminal */}
          <Card className="flex flex-col">
            <CardHeader>
              <div className="mb-2 flex size-10 items-center justify-center rounded-lg bg-zinc-950 border border-zinc-800 text-green-500">
                <IconTerminal className="size-6" />
              </div>
              <CardTitle>Terminal / 终端风</CardTitle>
              <CardDescription>
                模拟命令行界面的极客风格。深色背景，Monospace 字体，适合系统级通知、扫描日志或漏洞告警。
              </CardDescription>
            </CardHeader>
            <CardContent className="mt-auto pt-6">
              <Button onClick={triggerTerminal} variant="outline" className="w-full">
                预览效果 (Bottom Right)
              </Button>
            </CardContent>
          </Card>

          {/* Glass */}
          <Card className="flex flex-col">
            <CardHeader>
              <div className="mb-2 flex size-10 items-center justify-center rounded-lg bg-indigo-500/10 text-indigo-500">
                <IconCpu className="size-6" />
              </div>
              <CardTitle>Glass / 赛博玻璃</CardTitle>
              <CardDescription>
                带有发光边框和毛玻璃效果的未来感设计。适合 AI 助手建议、高优先级通知或科技感强的内容。
              </CardDescription>
            </CardHeader>
            <CardContent className="mt-auto pt-6">
              <Button onClick={triggerGlass} variant="outline" className="w-full">
                预览效果 (Bottom Right)
              </Button>
            </CardContent>
          </Card>

          {/* Report */}
          <Card className="flex flex-col">
            <CardHeader>
              <div className="mb-2 flex size-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <IconFileText className="size-6" />
              </div>
              <CardTitle>Report / 报告风</CardTitle>
              <CardDescription>
                结构清晰的专业风格，带有状态指示条和背景水印。适合常规业务通知、报表生成完成等场景。
              </CardDescription>
            </CardHeader>
            <CardContent className="mt-auto pt-6">
              <Button onClick={triggerReport} variant="outline" className="w-full">
                预览效果 (Bottom Right)
              </Button>
            </CardContent>
          </Card>

        </div>

        <div className="mx-auto mt-12 max-w-2xl text-center text-sm text-muted-foreground">
          <p>
            点击按钮可在当前页面预览不同风格的弹窗。
            <br />
            实际应用中，可根据消息类型（如：Security Alert 使用 Banner，Achievement 使用 Playful）动态切换组件。
          </p>
        </div>
      </div>
    </div>
  )
}
