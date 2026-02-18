"use client"

import React from "react"
import type { ColumnDef } from "@tanstack/react-table"
import { DemoCard, DemoSection } from "@/components/demo/component-gallery-sections/shared"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Separator } from "@/components/ui/separator"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
  CommandShortcut,
} from "@/components/ui/command"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { UnifiedDataTable } from "@/components/ui/data-table/unified-data-table"
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
} from "@/components/ui/sidebar"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { ChevronRight, Search, Settings, Wrench } from "@/components/icons"

const sampleTableData = [
  { id: "1", name: "Alpha Node", status: "online", owner: "Core" },
  { id: "2", name: "Beta Node", status: "degraded", owner: "Edge" },
  { id: "3", name: "Gamma Node", status: "maintenance", owner: "Ops" },
]

const sampleTableColumns: ColumnDef<(typeof sampleTableData)[number]>[] = [
  { accessorKey: "name", header: "Name" },
  { accessorKey: "status", header: "Status" },
  { accessorKey: "owner", header: "Owner" },
]

export function UiLayoutSection() {
  return (
    <DemoSection
      id="ui-layout"
      title="布局与数据展示"
      description="容器、表格、命令面板与侧边栏。"
    >
      <div className="grid gap-4 px-4 lg:px-6 md:grid-cols-2 xl:grid-cols-3">
        <DemoCard title="Card" description="内容容器">
          <Card className="border-dashed">
            <CardHeader>
              <CardTitle className="text-sm">资产概览</CardTitle>
              <CardDescription>示例描述文本</CardDescription>
            </CardHeader>
            <CardContent className="text-sm text-muted-foreground">
              这里是卡片内容区域。
            </CardContent>
          </Card>
        </DemoCard>

        <DemoCard title="Tabs" description="标签页">
          <Tabs defaultValue="overview">
            <TabsList>
              <TabsTrigger value="overview">Overview</TabsTrigger>
              <TabsTrigger value="assets">Assets</TabsTrigger>
            </TabsList>
            <TabsContent value="overview" className="text-sm text-muted-foreground">
              Overview 内容示例。
            </TabsContent>
            <TabsContent value="assets" className="text-sm text-muted-foreground">
              Assets 内容示例。
            </TabsContent>
          </Tabs>
        </DemoCard>

        <DemoCard title="ScrollArea" description="滚动容器">
          <ScrollArea className="h-28 rounded-md border p-2">
            <div className="space-y-2 text-sm">
              {Array.from({ length: 6 }).map((_, index) => (
                <div key={index} className="flex items-center justify-between">
                  <span>任务 #{index + 1}</span>
                  <Badge variant="outline">Queued</Badge>
                </div>
              ))}
            </div>
          </ScrollArea>
        </DemoCard>

        <DemoCard title="Separator" description="分割线">
          <div className="space-y-2">
            <div className="text-sm">上方内容</div>
            <Separator />
            <div className="text-sm text-muted-foreground">下方内容</div>
          </div>
        </DemoCard>

        <DemoCard title="Collapsible" description="折叠内容">
          <Collapsible>
            <CollapsibleTrigger asChild>
              <Button variant="outline">
                展开更多
                <ChevronRight className="ml-2 size-4" />
              </Button>
            </CollapsibleTrigger>
            <CollapsibleContent className="mt-2 text-sm text-muted-foreground">
              这里是折叠内容区域。
            </CollapsibleContent>
          </Collapsible>
        </DemoCard>

        <DemoCard title="Command" description="命令面板">
          <Command>
            <CommandInput placeholder="搜索指令…" />
            <CommandList>
              <CommandEmpty>无结果</CommandEmpty>
              <CommandGroup heading="导航">
                <CommandItem>
                  <Search className="mr-2 size-4" />
                  搜索资产
                  <CommandShortcut>⌘K</CommandShortcut>
                </CommandItem>
                <CommandItem>
                  <Settings className="mr-2 size-4" />
                  系统设置
                </CommandItem>
              </CommandGroup>
              <CommandSeparator />
              <CommandGroup heading="工具">
                <CommandItem>
                  <Wrench className="mr-2 size-4" />
                  快速扫描
                </CommandItem>
              </CommandGroup>
            </CommandList>
          </Command>
        </DemoCard>

        <DemoCard title="Table" description="基础表格">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>任务</TableHead>
                <TableHead>状态</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow>
                <TableCell>Recon</TableCell>
                <TableCell>Running</TableCell>
              </TableRow>
              <TableRow>
                <TableCell>Scan</TableCell>
                <TableCell>Queued</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </DemoCard>

        <DemoCard title="UnifiedDataTable" description="统一数据表格">
          <UnifiedDataTable
            data={sampleTableData}
            columns={sampleTableColumns}
            ui={{ hideToolbar: true, hidePagination: true }}
            behavior={{ enableRowSelection: false }}
          />
        </DemoCard>

        <DemoCard title="Sidebar" description="侧边栏骨架">
          <SidebarProvider>
            <div className="flex h-48 border rounded-md overflow-hidden">
              <Sidebar className="w-40">
                <SidebarContent>
                  <SidebarGroup>
                    <SidebarGroupLabel>导航</SidebarGroupLabel>
                    <SidebarGroupContent>
                      <SidebarMenu>
                        <SidebarMenuItem>
                          <SidebarMenuButton>Dashboard</SidebarMenuButton>
                        </SidebarMenuItem>
                        <SidebarMenuItem>
                          <SidebarMenuButton>Scan</SidebarMenuButton>
                        </SidebarMenuItem>
                      </SidebarMenu>
                    </SidebarGroupContent>
                  </SidebarGroup>
                </SidebarContent>
              </Sidebar>
              <div className="flex-1 p-3 text-xs text-muted-foreground">
                Sidebar 预览区域
              </div>
            </div>
          </SidebarProvider>
        </DemoCard>
      </div>
    </DemoSection>
  )
}
