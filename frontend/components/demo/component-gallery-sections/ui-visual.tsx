"use client"

import React from "react"
import dynamic from "next/dynamic"
import { toast } from "sonner"
import { DemoCard, DemoSection } from "@/components/demo/component-gallery-sections/shared"
import type { ChartConfig } from "@/components/ui/chart"
import { cn } from "@/lib/utils"

const DemoPlaceholder = ({ className }: { className?: string }) => (
  <div className={cn("rounded-md border border-border/60 bg-muted/30", className)} />
)

const ChartPreview = dynamic(
  () => import("@/components/demo/component-gallery-sections/chart-preview").then((mod) => mod.ChartPreview),
  {
    ssr: false,
    loading: () => <DemoPlaceholder className="h-40" />,
  }
)

const MermaidDiagram = dynamic(
  () => import("@/components/ui/mermaid-diagram").then((mod) => mod.MermaidDiagram),
  {
    ssr: false,
    loading: () => <DemoPlaceholder className="h-40" />,
  }
)

const TerminalPreview = dynamic(
  () => import("@/components/demo/component-gallery-sections/terminal-preview").then((mod) => mod.TerminalPreview),
  {
    ssr: false,
    loading: () => <DemoPlaceholder className="h-40" />,
  }
)

const TerminalLogin = dynamic(
  () => import("@/components/ui/terminal-login").then((mod) => mod.TerminalLogin),
  {
    ssr: false,
    loading: () => <DemoPlaceholder className="h-48" />,
  }
)

const YamlEditor = dynamic(
  () => import("@/components/ui/yaml-editor").then((mod) => mod.YamlEditor),
  {
    ssr: false,
    loading: () => <DemoPlaceholder className="h-48" />,
  }
)

const chartData = [
  { name: "Mon", value: 40 },
  { name: "Tue", value: 62 },
  { name: "Wed", value: 51 },
  { name: "Thu", value: 78 },
  { name: "Fri", value: 55 },
  { name: "Sat", value: 92 },
  { name: "Sun", value: 68 },
]

const chartConfig: ChartConfig = {
  value: {
    label: "Risk",
    color: "var(--color-primary)",
  },
}

const mermaidChart = `flowchart LR
  A[Recon] --> B{Scan Workflow}
  B -->|Fast| C[Quick Scan]
  B -->|Deep| D[Full Scan]
  C --> E[Assets]
  D --> E[Assets]
  E --> F[Risk Report]
`

const terminalTranslations = {
  title: "LunaFox Access Terminal",
  subtitle: "Secure access handshake",
  usernamePrompt: "USERNAME",
  passwordPrompt: "PASSWORD",
  authenticating: "AUTHENTICATING",
  processing: "PROCESSING",
  accessGranted: "ACCESS GRANTED",
  welcomeMessage: "Welcome back, operator.",
  authFailed: "AUTH FAILED",
  invalidCredentials: "Invalid credentials",
  shortcuts: "Shortcuts",
  submit: "Enter",
  cancel: "Ctrl+C",
  clear: "Ctrl+U",
  startEnd: "Ctrl+A",
}

export function UiVisualSection() {
  const [yamlValue, setYamlValue] = React.useState("targets:\\n  - example.com\\nscan:\\n  mode: quick\\n")

  return (
    <DemoSection
      id="ui-visual"
      title="可视化与高级组件"
      description="图表、终端、编辑器等高级组件。"
    >
      <div className="grid gap-4 px-4 lg:px-6 md:grid-cols-2 xl:grid-cols-3">
        <DemoCard title="Chart" description="Recharts 组合">
          <ChartPreview data={chartData} config={chartConfig} />
        </DemoCard>

        <DemoCard title="MermaidDiagram" description="流程图渲染">
          <MermaidDiagram chart={mermaidChart} />
        </DemoCard>

        <DemoCard title="Terminal" description="终端动效">
          <TerminalPreview />
        </DemoCard>

        <DemoCard title="TerminalLogin" description="登录终端">
          <TerminalLogin
            translations={terminalTranslations}
            onLogin={async () => {
              toast.success("认证完成")
            }}
          />
        </DemoCard>

        <DemoCard title="YamlEditor" description="YAML 编辑器">
          <div className="h-48">
            <YamlEditor value={yamlValue} onChange={setYamlValue} />
          </div>
        </DemoCard>
      </div>
    </DemoSection>
  )
}
