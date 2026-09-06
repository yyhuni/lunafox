import type { EngineExecutionDiagnostics, RuntimeTask, ScanRecord, StageStatus } from "@/types/scan.types"
import type { ScanLog } from "@/types/scan.types"

export interface RuntimeTaskItem {
  id: number
  title: string
  status: StageStatus
  skipReason?: string
  detail: string
  failureKind?: string
  failureSummary?: string
  failureDetail?: string
  startedAt?: string
  duration?: number
  completedAt?: string
  order: number
  diagnostics?: EngineExecutionDiagnostics
}

export type RuntimeTaskPresentationStatus = StageStatus | "interrupted" | "not_started"

export function getRuntimeTaskPresentationStatus(
  task: Pick<RuntimeTaskItem, "status" | "startedAt">
): RuntimeTaskPresentationStatus {
  if (task.status !== "cancelled") return task.status
  return task.startedAt ? "interrupted" : "not_started"
}

function getDurationBetween(startedAt: string | undefined, endedAt: Date | string | undefined) {
  if (!startedAt || !endedAt) return undefined

  const started = new Date(startedAt).getTime()
  const ended = endedAt instanceof Date ? endedAt.getTime() : new Date(endedAt).getTime()

  if (!Number.isFinite(started) || !Number.isFinite(ended) || ended < started) return undefined

  return (ended - started) / 1000
}

export function getRuntimeTaskDurationSeconds(task: RuntimeTaskItem, now: Date = new Date()) {
  if (task.duration !== undefined && task.duration !== null && !Number.isNaN(task.duration)) {
    return task.duration
  }

  if (task.status === "running") {
    return getDurationBetween(task.startedAt, now)
  }

  return getDurationBetween(task.startedAt, task.completedAt)
}

function getRuntimeTaskFailureSummary(task: RuntimeTask, scan: ScanRecord | null, isCanonicalFailure: boolean) {
  return task.error || (isCanonicalFailure ? scan?.failure?.message : undefined)
}

export function buildTaskProgressLogsByTaskId(logs: ScanLog[]) {
  const taskProgressLogsByTaskId = new Map<number, ScanLog[]>()

  for (const log of logs) {
    if (typeof log.taskId !== "number") continue
    const progressLogs = taskProgressLogsByTaskId.get(log.taskId) ?? []
    progressLogs.push(log)
    taskProgressLogsByTaskId.set(log.taskId, progressLogs)
  }

  for (const [taskId, progressLogs] of taskProgressLogsByTaskId.entries()) {
    taskProgressLogsByTaskId.set(
      taskId,
      [...progressLogs].sort((a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime() || a.id - b.id)
    )
  }

  return taskProgressLogsByTaskId
}

export function toRuntimeTaskItems(
  tasks: RuntimeTask[] = [],
  scan: ScanRecord | null,
  localizedEngineNamesByID: ReadonlyMap<string, string>,
  localizedEngineDescriptionsByID: ReadonlyMap<string, string>,
): RuntimeTaskItem[] {
  const firstFailedTaskId = tasks.find((task) => task.status === "failed")?.id

  return tasks
    .map((task, index) => {
      const isCanonicalFailure = task.id === firstFailedTaskId
      const title = localizedEngineNamesByID.get(task.engineId)!

      return {
        id: task.id,
        title,
        status: task.status,
        skipReason: task.skipReason,
        detail: localizedEngineDescriptionsByID.get(task.engineId)!,
        failureKind: task.failureKind || (isCanonicalFailure ? scan?.failure?.kind : undefined),
        failureSummary: getRuntimeTaskFailureSummary(task, scan, isCanonicalFailure),
        failureDetail: task.failureDetail,
        startedAt: task.startedAt,
        completedAt: task.completedAt,
        duration: task.duration,
        order: task.order ?? index,
        diagnostics: task.diagnostics,
      }
    })
    .sort((a, b) => a.order - b.order)
}
