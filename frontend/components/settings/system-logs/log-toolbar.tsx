"use client"

import { useMemo } from "react"
import { useTranslations } from "next-intl"
import { FileText, Search } from "@/components/icons"

import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Input } from "@/components/ui/input"
import type { LogFile } from "@/types/system-log.types"

const LINE_OPTIONS = [100, 200, 500, 1000, 5000, 10000] as const

export type LogLevel = "all" | "DEBUG" | "INFO" | "WARN" | "ERROR"
export const LOG_LEVELS: LogLevel[] = ["all", "DEBUG", "INFO", "WARN", "ERROR"]

interface LogToolbarProps {
  files: LogFile[]
  selectedFile: string
  lines: number
  searchQuery: string
  logLevel: LogLevel
  onFileChange: (filename: string) => void
  onLinesChange: (lines: number) => void
  onSearchChange: (query: string) => void
  onLogLevelChange: (level: LogLevel) => void
}

export function LogToolbar({
  files,
  selectedFile,
  lines,
  searchQuery,
  logLevel,
  onFileChange,
  onLinesChange,
  onSearchChange,
  onLogLevelChange,
}: LogToolbarProps) {
  const t = useTranslations("settings.systemLogs")

  // Group files into categories
  const groupedFiles = useMemo(() => {
    const systemLogs = files.filter(
      (f) => f.category === "system" || f.category === "error" || f.category === "performance"
    )
    const containerLogs = files.filter((f) => f.category === "container")
    return { systemLogs, containerLogs }
  }, [files])

  return (
    <div className="flex items-center gap-3 flex-wrap">
      {/* Log file selection */}
      <Select value={selectedFile} onValueChange={onFileChange}>
        <SelectTrigger className="w-[200px]">
          <FileText className="h-4 w-4 text-muted-foreground" />
          <SelectValue placeholder={t("toolbar.selectFile")} />
        </SelectTrigger>
        <SelectContent>
          {groupedFiles.systemLogs.length > 0 && (
            <SelectGroup>
              <SelectLabel>{t("toolbar.systemLogsGroup")}</SelectLabel>
              {groupedFiles.systemLogs.map((file) => (
                <SelectItem key={file.filename} value={file.filename}>
                  {file.filename}
                </SelectItem>
              ))}
            </SelectGroup>
          )}
          {groupedFiles.containerLogs.length > 0 && (
            <SelectGroup>
              <SelectLabel>{t("toolbar.containerLogsGroup")}</SelectLabel>
              {groupedFiles.containerLogs.map((file) => (
                <SelectItem key={file.filename} value={file.filename}>
                  {file.filename}
                </SelectItem>
              ))}
            </SelectGroup>
          )}
        </SelectContent>
      </Select>

      {/* Row number selection */}
      <Select value={String(lines)} onValueChange={(v) => onLinesChange(Number(v))}>
        <SelectTrigger className="w-[110px]">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {LINE_OPTIONS.map((option) => (
            <SelectItem key={option} value={String(option)}>
              {option} {t("toolbar.linesUnit")}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      {/* Log level filter */}
      <Select value={logLevel} onValueChange={(v) => onLogLevelChange(v as LogLevel)}>
        <SelectTrigger className="w-[130px]">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {LOG_LEVELS.map((level) => (
            <SelectItem key={level} value={level}>
              {level === "all" ? t("toolbar.levelAll") : level}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      {/* Search box - centered and expanded */}
      <div className="relative flex-1 min-w-[280px] max-w-[500px]">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          type="search"
          name="logsSearch"
          autoComplete="off"
          placeholder={t("toolbar.searchPlaceholder")}
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
          className="w-full pl-9 h-9"
        />
      </div>
    </div>
  )
}
