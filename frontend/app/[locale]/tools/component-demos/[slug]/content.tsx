"use client"

import { useParams } from "next/navigation"
import { PageHeader } from "@/components/common/page-header"
import { demoMap } from "@/components/demo/component-demo-registry"
import { DynamicModuleDemo } from "@/components/demo/dynamic-module-demo"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"

export default function ComponentDemoPage() {
  const params = useParams()
  const slug = typeof params?.slug === "string" ? params.slug : Array.isArray(params?.slug) ? params?.slug[0] : ""
  const demo = demoMap[slug]

  if (!demo) {
    return (
      <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
        <PageHeader code="DEMO" title="组件 Demo" description="未找到对应组件" />
        <div className="px-4 lg:px-6">
          <Alert>
            <AlertTitle>无效的组件 Demo</AlertTitle>
            <AlertDescription className="text-xs text-muted-foreground">
              请返回组件 Demo 索引选择有效组件。
            </AlertDescription>
          </Alert>
        </div>
      </div>
    )
  }

  const showHeader = demo.kind === "business"

  return (
    <DynamicModuleDemo
      loader={demo.loader}
      props={demo.props}
      title={showHeader ? demo.title : undefined}
      description={showHeader ? demo.description : undefined}
      fallbackRoute={showHeader ? demo.fallbackRoute : undefined}
    />
  )
}
