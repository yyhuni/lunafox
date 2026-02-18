"use client"

import { AllTargetsDetailView } from "@/components/target/all-targets-detail-view"
import { Button } from "@/components/ui/button"

/**
 * Demo B: invisible title
 * Design concept: Header is completely borderless, relying only on font size and spacing to establish hierarchy; tables retain their own borders
 * Key CSS: Header without any border decoration
 */
export default function DemoPageB() {
  return (
    <div className="flex flex-col h-full">
      {/* Description area */}
      <div className="p-6 border-b bg-muted/30">
        <h1 className="text-xl font-bold">方案 B：隐形标题</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Header 无边框悬浮，表格独立拥有边框。最透气的设计。
        </p>
      </div>

      {/* Core Demo area */}
      <div className="flex-1 p-6 flex flex-col min-h-0">
        
        {/* Header area - plain text, without any borders */}
        <div className="flex flex-col md:flex-row md:items-end justify-between gap-4 pb-4 shrink-0 px-2">
          <div className="space-y-1">
            <div className="flex items-center gap-3">
              <h1 className="text-3xl font-bold tracking-tight text-foreground">目标管理</h1>
              <span className="font-mono text-xs text-muted-foreground bg-muted px-2 py-1">
                /TGT-01
              </span>
            </div>
            <p className="text-muted-foreground text-sm">
              Manage and monitor all your scan targets here.
            </p>
          </div>
          <div className="flex gap-2">
             <Button variant="secondary" size="sm">Export</Button>
             <Button size="sm">New Scan</Button>
          </div>
        </div>

        {/* Table area - keep the table's own borders, but bold the top border for emphasis */}
        <div className="flex-1 overflow-auto">
          <AllTargetsDetailView
            className="space-y-3"
            tableClassName="border-2 border-primary rounded-none"
          />
        </div>
      </div>
    </div>
  )
}
