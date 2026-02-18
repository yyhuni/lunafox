"use client"

import React from "react"
import dynamic from "next/dynamic"
import type { ComponentGroup } from "@/components/demo/component-index"
import { PageHeader } from "@/components/common/page-header"
import { Toaster } from "@/components/ui/sonner"

type ComponentGalleryProps = {
  componentGroups: ComponentGroup[]
}

const SectionSkeleton = ({
  title,
  description,
  cards = 6,
}: {
  title: string
  description?: string
  cards?: number
}) => (
  <section className="space-y-4">
    <div className="px-4 lg:px-6">
      <div className="h-5 w-40 rounded bg-muted/50" />
      {description ? (
        <div className="mt-2 h-4 w-72 rounded bg-muted/40" />
      ) : null}
      <div className="sr-only">{title}</div>
    </div>
    <div className="grid gap-4 px-4 lg:px-6 md:grid-cols-2 xl:grid-cols-3">
      {Array.from({ length: cards }).map((_, index) => (
        <div key={index} className="h-32 rounded-md border border-border/60 bg-muted/30" />
      ))}
    </div>
  </section>
)

const UiBasicsSection = dynamic(
  () => import("./component-gallery-sections/ui-basics").then((mod) => mod.UiBasicsSection),
  {
    ssr: false,
    loading: () => (
      <SectionSkeleton
        title="UI 基础组件"
        description="按钮、表单、选择器与基础元素。"
      />
    ),
  }
)

const UiOverlaysSection = dynamic(
  () => import("./component-gallery-sections/ui-overlays").then((mod) => mod.UiOverlaysSection),
  {
    ssr: false,
    loading: () => (
      <SectionSkeleton
        title="反馈与浮层"
        description="弹窗、提示、反馈与状态组件。"
      />
    ),
  }
)

const UiLayoutSection = dynamic(
  () => import("./component-gallery-sections/ui-layout").then((mod) => mod.UiLayoutSection),
  {
    ssr: false,
    loading: () => (
      <SectionSkeleton
        title="布局与数据展示"
        description="容器、表格、命令面板与侧边栏。"
      />
    ),
  }
)

const UiVisualSection = dynamic(
  () => import("./component-gallery-sections/ui-visual").then((mod) => mod.UiVisualSection),
  {
    ssr: false,
    loading: () => (
      <SectionSkeleton
        title="可视化与高级组件"
        description="图表、终端、编辑器等高级组件。"
      />
    ),
  }
)

const UiBrandingSection = dynamic(
  () => import("./component-gallery-sections/ui-branding").then((mod) => mod.UiBrandingSection),
  {
    ssr: false,
    loading: () => (
      <SectionSkeleton
        title="品牌与状态组件"
        description="Banner 与状态标识。"
        cards={2}
      />
    ),
  }
)

const BusinessIndexSection = dynamic(
  () => import("./component-gallery-sections/business-index").then((mod) => mod.BusinessIndexSection),
  {
    ssr: false,
    loading: () => (
      <SectionSkeleton
        title="业务组件索引"
        description="以下为业务模块组件清单与入口。建议在 Mock 模式查看完整数据交互。"
        cards={4}
      />
    ),
  }
)

export function ComponentGallery({ componentGroups }: ComponentGalleryProps) {
  return (
    <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <Toaster />
      <PageHeader
        code="COMP-LAB"
        title="组件总览"
        description="覆盖 UI 基础组件与业务模块组件的统一演示入口。业务模块默认建议在 Mock 模式查看完整数据展示。"
      />

      <UiBasicsSection />
      <UiOverlaysSection />
      <UiLayoutSection />
      <UiVisualSection />
      <UiBrandingSection />
      <BusinessIndexSection componentGroups={componentGroups} />
    </div>
  )
}
