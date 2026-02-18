"use client"

import React from "react"
import { DemoCard, DemoSection } from "@/components/demo/component-gallery-sections/shared"
import { Banner, BannerAction, BannerIcon, BannerTitle } from "@/components/ui/shadcn-io/banner"
import { Status, StatusIndicator, StatusLabel } from "@/components/ui/shadcn-io/status"
import { Info } from "@/components/icons"

export function UiBrandingSection() {
  return (
    <DemoSection
      id="ui-branding"
      title="品牌与状态组件"
      description="Banner 与状态标识。"
    >
      <div className="grid gap-4 px-4 lg:px-6 md:grid-cols-2 xl:grid-cols-3">
        <DemoCard title="Banner" description="顶部提示横条">
          <Banner>
            <BannerIcon icon={Info} />
            <div className="flex flex-1 flex-col gap-0.5">
              <BannerTitle>系统维护</BannerTitle>
              <p className="text-xs text-primary-foreground/80">预计 5 分钟后恢复。</p>
            </div>
            <BannerAction>查看详情</BannerAction>
          </Banner>
        </DemoCard>

        <DemoCard title="Status" description="状态指示">
          <div className="flex flex-col gap-2">
            <Status status="online">
              <StatusIndicator />
              <StatusLabel />
            </Status>
            <Status status="degraded">
              <StatusIndicator />
              <StatusLabel />
            </Status>
          </div>
        </DemoCard>
      </div>
    </DemoSection>
  )
}
