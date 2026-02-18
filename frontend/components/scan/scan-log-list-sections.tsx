"use client"

import { AnsiLogViewer } from "@/components/settings/system-logs/ansi-log-viewer"

export function ScanLogListLoadingState() {
  return (
    <div className="h-full flex items-center justify-center bg-[#1e1e1e] text-[#808080]">
      加载中...
    </div>
  )
}

export function ScanLogListEmptyState() {
  return (
    <div className="h-full flex items-center justify-center bg-[#1e1e1e] text-[#808080]">
      暂无日志
    </div>
  )
}

export function ScanLogListContent({ content }: { content: string }) {
  return (
    <div className="h-full">
      <AnsiLogViewer content={content} />
    </div>
  )
}
