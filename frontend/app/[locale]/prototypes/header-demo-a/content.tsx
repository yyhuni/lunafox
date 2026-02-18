"use client"

import { AllTargetsDetailView } from "@/components/target/all-targets-detail-view"
import { Button } from "@/components/ui/button"
import { Plus } from "@/components/icons"

/**
 * Demo A: All-in-one card
 * Design concept: Put Header and Table into the same container, only the outer frame has a border
 * Key CSS: Remove the borders and rounded corners that come with tables
 */
export default function DemoPageA() {
  return (
    <div className="flex flex-col h-full">
      {/* Description area */}
      <div className="p-6 border-b bg-muted/30">
        <h1 className="text-xl font-bold">方案 A：一体化卡片</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Header 和 Table 共用一个外边框。表格内部边框被移除。
        </p>
      </div>

      {/* Core Demo area */}
      <div className="flex-1 p-6">
        {/* Unified container - outer unique border */}
        <div className="border-2 border-primary bg-card h-full flex flex-col overflow-hidden">
          
          {/* Header area */}
          <div className="flex flex-col md:flex-row md:items-center justify-between px-2 py-3 border-b border-border bg-muted/10 shrink-0">
            <div className="flex items-center gap-3 mb-3 md:mb-0">
              <div className="bg-primary text-primary-foreground px-2 py-1 text-[10px] font-mono font-bold tracking-wider">
                TGT-01
              </div>
              <div>
                <h2 className="text-lg font-bold tracking-tight">目标管理</h2>
                <p className="text-xs text-muted-foreground">Manage all scan targets</p>
              </div>
            </div>
            <Button size="sm">
              <Plus className="h-3.5 w-3.5 mr-1"/> Add Target
            </Button>
          </div>

          {/* Table area - remove the table's own borders */}
          <div className="flex-1 overflow-auto">
            <AllTargetsDetailView
              className="space-y-3 px-2 pb-3"
              tableClassName="border-0 rounded-none"
            />
          </div>
        </div>
      </div>
    </div>
  )
}
