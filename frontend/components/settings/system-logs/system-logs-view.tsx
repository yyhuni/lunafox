"use client"

import { useCallback, useEffect, useMemo, useState, useDeferredValue } from "react"
import { useTranslations } from "next-intl"
import { Download } from "@/components/icons"

import { Separator } from "@/components/ui/separator"
import { Switch } from "@/components/ui/switch"
import { Label } from "@/components/ui/label"
import { Button } from "@/components/ui/button"
import { useSystemLogs, useLogFiles } from "@/hooks/use-system-logs"
import { LogToolbar, type LogLevel } from "./log-toolbar"
import { AnsiLogViewer } from "./ansi-log-viewer"
import { PageHeader } from "@/components/common/page-header"
import { downloadBlob } from "@/lib/download-utils"

const DEFAULT_FILE = "lunafox.log"
const DEFAULT_LINES = 500

export function SystemLogsView() {
  const t = useTranslations("settings.systemLogs")

  // Status management
  const [selectedFile, setSelectedFile] = useState(DEFAULT_FILE)
  const [lines, setLines] = useState(DEFAULT_LINES)
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [searchQuery, setSearchQuery] = useState("")
  const [logLevel, setLogLevel] = useState<LogLevel>("all")
  const deferredSearchQuery = useDeferredValue(searchQuery)

  // Get a list of log files
  const { data: filesData } = useLogFiles()
  const files = useMemo(() => filesData?.files ?? [], [filesData?.files])

  // When the file list is loaded, if the currently selected file is not in the list, switch to the first available file
  useEffect(() => {
    if (files.length > 0 && !files.some((f) => f.filename === selectedFile)) {
      setSelectedFile(files[0].filename)
    }
  }, [files, selectedFile])

  // Get log content
  const { data: logsData } = useSystemLogs({
    file: selectedFile,
    lines,
    autoRefresh,
  })

  // Preserve ANSI color codes, rendered by xterm
  const content = useMemo(() => logsData?.content ?? "", [logsData])

  // Download log
  const handleDownload = useCallback(() => {
    if (!content) return
    
    const blob = new Blob([content], { type: "text/plain;charset=utf-8" })
    downloadBlob(blob, selectedFile)
  }, [content, selectedFile])

  return (
    <div className="flex flex-1 flex-col gap-4 py-4 md:gap-6 md:py-6 min-h-0">
      <PageHeader
        code="LOG-01"
        title={t("title")}
        description={t("description")}
      />

      {/* Compact single-line toolbar */}
      <div className="px-4 lg:px-6">
        <div className="flex items-center gap-4 flex-wrap">
          <LogToolbar
            files={files}
            selectedFile={selectedFile}
            lines={lines}
            searchQuery={searchQuery}
            logLevel={logLevel}
            onFileChange={setSelectedFile}
            onLinesChange={setLines}
            onSearchChange={setSearchQuery}
            onLogLevelChange={setLogLevel}
          />
          {/* download button */}
          <Button
            variant="outline"
            size="sm"
            className="h-9"
            onClick={handleDownload}
            disabled={!content}
          >
            <Download className="h-4 w-4 mr-1.5" />
            {t("toolbar.download")}
          </Button>
        </div>
      </div>

      {/* Log viewer */}
      <div className="px-4 lg:px-6 flex-1 min-h-0">
        <div className="flex-1 flex flex-col rounded-lg overflow-hidden border min-h-0">
          <div className="flex-1 min-h-[400px] bg-[#1e1e1e]">
            {content ? (
              <AnsiLogViewer content={content} searchQuery={deferredSearchQuery} logLevel={logLevel} />
            ) : (
              <div className="flex items-center justify-center h-full text-muted-foreground">
                {t("noContent")}
              </div>
            )}
          </div>

          {/* bottom status bar */}
          <div className="flex items-center justify-between px-4 py-2 bg-muted/50 border-t text-xs text-muted-foreground">
            <div className="flex items-center gap-4">
              <span>{lines} {t("toolbar.linesUnit")}</span>
              <Separator orientation="vertical" className="h-3" />
              <span>{selectedFile}</span>
              <Separator orientation="vertical" className="h-3" />
              <span className="flex items-center gap-1.5">
                {autoRefresh && (
                  <span className="size-1.5 rounded-full bg-green-500 animate-pulse" />
                )}
                {t("description")}
              </span>
            </div>
            {/* Auto refresh switch */}
            <div className="flex items-center gap-2">
              <Switch
                id="auto-refresh"
                checked={autoRefresh}
                onCheckedChange={setAutoRefresh}
                className="scale-75"
              />
              <Label
                htmlFor="auto-refresh"
                className="text-xs cursor-pointer"
              >
                {t("toolbar.autoRefresh")}
              </Label>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
