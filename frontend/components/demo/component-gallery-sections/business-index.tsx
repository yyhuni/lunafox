"use client"

import React from "react"
import type { ComponentGroup } from "@/components/demo/component-index"
import { DemoSection } from "@/components/demo/component-gallery-sections/shared"
import { Link } from "@/i18n/navigation"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

const demoRoutes: Record<string, string> = {
  root: "/dashboard/",
  common: "/prototypes/header-demo-a/",
  dashboard: "/dashboard/",
  scan: "/scan/",
  target: "/target/",
  organization: "/organization/",
  vulnerabilities: "/vulnerabilities/",
  search: "/search/",
  tools: "/tools/",
  settings: "/settings/workers/",
  fingerprints: "/tools/fingerprints/",
  endpoints: "/target/",
  directories: "/target/",
  websites: "/target/",
  subdomains: "/target/",
  "ip-addresses": "/target/",
  notifications: "/settings/notifications/",
  auth: "/login/",
  providers: "/dashboard/",
  "animate-ui": "/prototypes/dashboard-demo/",
  prototypes: "/prototypes/scan-dialogs/",
  disk: "/settings/database-health/",
  screenshots: "/target/",
}

export function BusinessIndexSection({ componentGroups }: { componentGroups: ComponentGroup[] }) {
  return (
    <DemoSection
      id="business-index"
      title="业务组件索引"
      description="以下为业务模块组件清单与入口。建议在 Mock 模式查看完整数据交互。"
    >
      <div className="grid gap-4 px-4 lg:px-6 md:grid-cols-2">
        {componentGroups.map((group) => {
          const items = group.items
          const preview = items.slice(0, 14)
          const rest = items.length - preview.length
          const route = demoRoutes[group.key]

          return (
            <Card key={group.key} className="border-border/60">
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                  <span>{group.title}</span>
                  {route ? (
                    <Badge variant="outline" className="text-[10px]">
                      {route}
                    </Badge>
                  ) : null}
                </CardTitle>
                {group.description ? (
                  <CardDescription className="text-xs">{group.description}</CardDescription>
                ) : null}
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="flex flex-wrap gap-2">
                  {preview.map((item) => (
                    <Badge key={item} variant="outline" className="text-[10px]">
                      {item}
                    </Badge>
                  ))}
                  {rest > 0 ? (
                    <Badge variant="secondary" className="text-[10px]">
                      +{rest}
                    </Badge>
                  ) : null}
                </div>
                {route ? (
                  <Button asChild variant="outline" size="sm">
                    <Link href={route}>进入模块</Link>
                  </Button>
                ) : null}
              </CardContent>
            </Card>
          )
        })}
      </div>
    </DemoSection>
  )
}
