"use client"

import React from "react"
import { PageHeader } from "@/components/common/page-header"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Link } from "@/i18n/navigation"
import { demoEntries } from "@/components/demo/component-demo-registry"

export default function ComponentDemosIndexPage() {
  const [query, setQuery] = React.useState("")

  const filtered = demoEntries.filter((item) =>
    `${item.title} ${item.group} ${item.kind}`.toLowerCase().includes(query.toLowerCase())
  )

  const grouped = filtered.reduce<Record<string, typeof filtered>>((acc, item) => {
    const groupTitle = item.kind === "ui" ? `UI / ${item.group}` : `业务 / ${item.group}`
    if (!acc[groupTitle]) acc[groupTitle] = []
    acc[groupTitle].push(item)
    return acc
  }, {})

  return (
    <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <PageHeader
        code="DEMO-LAB"
        title="组件 Demo 索引"
        description="UI 与业务组件的独立 Demo 页面集合。"
      />
      <div className="px-4 lg:px-6">
        <Input
          type="search"
          name="componentSearch"
          autoComplete="off"
          placeholder="搜索组件…"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          className="max-w-md"
        />
      </div>
      <div className="grid gap-4 px-4 lg:px-6">
        {Object.entries(grouped).map(([group, items]) => (
          <Card key={group} className="border-border/60">
            <CardHeader>
              <CardTitle className="text-base">{group}</CardTitle>
              <CardDescription className="text-xs">共 {items.length} 个组件</CardDescription>
            </CardHeader>
            <CardContent className="flex flex-wrap gap-2">
              {items.map((item) => (
                <Badge key={item.slug} variant="outline" className="text-xs">
                  <Link href={`/tools/component-demos/${item.slug}/`}>{item.title}</Link>
                </Badge>
              ))}
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
