"use client"

import * as React from "react"
import {
  Background,
  getViewportForBounds,
  Handle,
  MarkerType,
  Panel,
  Position,
  ReactFlow,
  useReactFlow,
  useNodesState,
  useStore,
  useViewport,
  type Edge,
  type Node,
  type NodeProps,
} from "@xyflow/react"
import { useTranslations } from "next-intl"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  AlertDialog,
  AlertDialogClose,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  DETAIL_DRAWER_COMPACT_BODY_CLASS,
  DETAIL_DRAWER_COMPACT_FOOTER_CLASS,
} from "@/components/shared/detail-drawer"
import { CopyButton } from "@/components/shared/feedback/copy-button"
import {
  ArrowLeft,
  FileCode,
  FocusCentered,
  GripVerticalIcon,
  IconPlus,
  IconShieldPlus,
  IconTrash,
  Minus,
  PanelLeftCloseIcon,
  PanelLeftIcon,
  PanelRightCloseIcon,
  PanelRightIcon,
  Save,
  semanticIcons,
} from "@/components/icons"
import type { WorkflowEngineLibraryItem } from "@/lib/engine-catalog"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import { narrowFormDialogContentClassName } from "@/lib/ui/overlay-styles"
import type { ScanWorkflowStageView } from "@/types/scan-workflow.types"

import styles from "./workflow-composition-canvas.module.css"

export type WorkflowEngineCatalogStatus = "loading" | "error" | "ready"

type DraftStep = {
  id: string
  engineId: string
  profileDefaultEnabled: boolean
}

type DraftStage = {
  id: string
  label: string
  steps: DraftStep[]
}

type StageNodeData = {
  stage: DraftStage
  selectedStageId: string
  dragOverStageId: string | null
  onSelectStage: (stageId: string) => void
  onDropEngine: (stageId: string, engineId: string) => void
  onDragOverStage: (stageId: string | null) => void
  isLastStage: boolean
  onAddSerialStageAfter: (stageId: string) => void
  enginesById: ReadonlyMap<string, WorkflowEngineLibraryItem>
  engineCatalogStatus: WorkflowEngineCatalogStatus
}

type WorkflowStageNode = Node<StageNodeData, "workflowStage">
type WorkflowTranslator = ReturnType<typeof useTranslations>

const nodeTypes = {
  workflowStage: WorkflowStageNode,
}

const defaultEdgeOptions = {
  type: "bezier",
  focusable: false,
  selectable: false,
  animated: true,
  markerEnd: { type: MarkerType.ArrowClosed },
  labelShowBg: false,
} satisfies Partial<Edge>

const WORKFLOW_CANVAS_MIN_ZOOM = 0.4
const WORKFLOW_CANVAS_MAX_ZOOM = 1.6
const WORKFLOW_STAGE_HORIZONTAL_SPACING = 400
const WORKFLOW_STAGE_VERTICAL_STAGGER = 56
// Desktop auto-layout centers within the area not obscured by the floating engine library.
const WORKFLOW_ENGINE_LIBRARY_OCCLUSION_PX = 304
const WORKFLOW_STAGE_CARD_DEFAULT_WIDTH = 256
const WORKFLOW_STAGE_CARD_DEFAULT_HEIGHT = 100

function resolveNodesBounds(nodes: Node[], getBounds: (nodes: Node[]) => { x: number; y: number; width: number; height: number }) {
  const populatedNodes = nodes.map((node) => ({
    ...node,
    measured: {
      width: node.measured?.width ?? WORKFLOW_STAGE_CARD_DEFAULT_WIDTH,
      height: node.measured?.height ?? WORKFLOW_STAGE_CARD_DEFAULT_HEIGHT,
    },
    width: node.width ?? node.measured?.width ?? WORKFLOW_STAGE_CARD_DEFAULT_WIDTH,
    height: node.height ?? node.measured?.height ?? WORKFLOW_STAGE_CARD_DEFAULT_HEIGHT,
  }))
  return getBounds(populatedNodes)
}

function WorkflowCanvasZoomControls() {
  const t = useTranslations("scan.workflow")
  const { zoomTo } = useReactFlow()
  const { zoom } = useViewport()
  const zoomPercent = Math.round(zoom * 100)

  return (
    <div className={styles.zoomControls} data-workflow-canvas-zoom-controls>
      <Button
        type="button"
        variant="quiet"
        size="icon-sm"
        className="radius-none hover:bg-muted"
        aria-label={t("canvas.zoomOut")}
        disabled={zoom <= WORKFLOW_CANVAS_MIN_ZOOM}
        onClick={() => void zoomTo(Math.max(WORKFLOW_CANVAS_MIN_ZOOM, zoom - 0.1))}
      >
        <Minus className="size-4" aria-hidden="true" />
      </Button>
      <output className={cn("flex h-8 min-w-12 items-center justify-center border-x text-center tabular-nums", textRole.compactCaption)} aria-live="polite">
        {zoomPercent}%
      </output>
      <Button
        type="button"
        variant="quiet"
        size="icon-sm"
        className="radius-none hover:bg-muted"
        aria-label={t("canvas.zoomIn")}
        disabled={zoom >= WORKFLOW_CANVAS_MAX_ZOOM}
        onClick={() => void zoomTo(Math.min(WORKFLOW_CANVAS_MAX_ZOOM, zoom + 0.1))}
      >
        <IconPlus className="size-4" aria-hidden="true" />
      </Button>
    </div>
  )
}

function createStep(engineId: string, enginesById: ReadonlyMap<string, WorkflowEngineLibraryItem>, stepId: string): DraftStep {
  if (!enginesById.has(engineId)) {
    throw new Error(`Engine ${engineId} is unavailable in the installed catalog`)
  }
  return {
    id: stepId,
    engineId,
    profileDefaultEnabled: true,
  }
}

function createInitialDraft(t: WorkflowTranslator, stages?: ScanWorkflowStageView[]): DraftStage[] {
  if (stages?.length) {
    return stages.map((stage) => ({
      id: stage.stageId,
      label: stage.stageId,
      steps: stage.steps.map((step) => ({
        id: step.stepId,
        engineId: step.engineId,
        profileDefaultEnabled: step.profileDefaultEnabled,
      })),
    }))
  }
  return [
    {
      id: "stage-1",
      label: t("canvas.serialStageLabel", { count: 1 }),
      steps: [],
    },
    {
      id: "stage-2",
      label: t("canvas.serialStageLabel", { count: 2 }),
      steps: [],
    },
  ]
}

function WorkflowStageNode({ data }: NodeProps<WorkflowStageNode>) {
  const t = useTranslations("scan.workflow")
  const WorkflowIcon = semanticIcons.concept.workflow
  const isSelected = data.stage.id === data.selectedStageId
  const isDragOver = data.stage.id === data.dragOverStageId

  const handleDragOver = React.useCallback((event: React.DragEvent) => {
    event.preventDefault()
    data.onDragOverStage(data.stage.id)
  }, [data])

  const handleDragLeave = React.useCallback(() => {
    data.onDragOverStage(null)
  }, [data])

  const handleDrop = React.useCallback((event: React.DragEvent) => {
    event.preventDefault()
    const engineId = event.dataTransfer.getData("application/x-lunafox-engine")
    if (engineId) {
      data.onDropEngine(data.stage.id, engineId)
    }
    data.onDragOverStage(null)
  }, [data])

  return (
    <div
      data-stage-drop-target={data.stage.id}
      data-stage-selected={isSelected ? "true" : undefined}
      data-stage-drag-over={isDragOver ? "true" : undefined}
      className={cn(
        "radius-overlay relative w-64 border shadow-2xs transition-[background-color,border-color,box-shadow]",
        styles.stageNode
      )}
      onClick={() => data.onSelectStage(data.stage.id)}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
    >
      <Handle type="target" position={Position.Left} />
      <div className="flex min-w-0 items-center justify-between gap-1.5 border-b px-3 py-2">
        <div className="flex min-w-0 items-center gap-1.5">
          <WorkflowIcon className="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
          <p className={cn("min-w-0 truncate font-medium", textRole.compactSectionTitle)}>{data.stage.label}</p>
        </div>
        <div className="flex shrink-0 items-center gap-1">
          {data.stage.steps.length > 0 ? (
            <Badge variant="secondary" data-stage-execution-mode={data.stage.steps.length === 1 ? "serial" : "parallel"}>
              {t(data.stage.steps.length === 1 ? "canvas.serialExecution" : "canvas.parallelExecution")}
            </Badge>
          ) : null}
        </div>
      </div>

      <div className="flex flex-col gap-1.5 p-2.5">
        {data.stage.steps.length === 0 ? (
          <div
            className="radius-control flex h-9 items-center justify-center gap-1.5 border border-dashed px-2.5 py-1.5 text-center"
            data-empty-stage-placeholder
          >
            <IconPlus className="size-3.5 shrink-0 text-muted-foreground/70" aria-hidden="true" />
            <p className={cn(textRole.compactCaption, "text-muted-foreground font-normal")}>{t("canvas.emptyStageDropHint")}</p>
          </div>
        ) : data.stage.steps.map((step) => (
          <WorkflowEngineStep
            key={step.id}
            step={step}
            engine={data.enginesById.get(step.engineId)}
            engineCatalogStatus={data.engineCatalogStatus}
          />
        ))}

      </div>
      {data.isLastStage ? (
        <>
          <Handle type="source" position={Position.Right} className="opacity-0" />
          <span className="nodrag nopan radius-round absolute top-1/2 right-0 z-10 flex size-7 translate-x-1/2 -translate-y-1/2 items-center justify-center bg-[var(--workflow-add-stage-pad-background)]">
            <Button
              type="button"
              variant="surface"
              size="icon-sm"
              aria-label={t("canvas.addSerialStage")}
              data-add-serial-handle="true"
              className="radius-round size-5 border bg-[var(--workflow-add-stage-background)] p-0 text-[var(--workflow-add-stage-foreground)] shadow-xs hover:bg-[var(--workflow-add-stage-background)] [border-color:var(--workflow-add-stage-border)]"
              onClick={(event) => {
                event.stopPropagation()
                data.onAddSerialStageAfter(data.stage.id)
              }}
            >
              <IconPlus className="size-2.5" />
            </Button>
          </span>
        </>
      ) : (
        <Handle type="source" position={Position.Right} />
      )}
    </div>
  )
}

const ScanEngineIcon = semanticIcons.concept.engine

function WorkflowEngineStep({
  step,
  engine,
  engineCatalogStatus,
}: {
  step: DraftStep
  engine?: WorkflowEngineLibraryItem
  engineCatalogStatus: WorkflowEngineCatalogStatus
}) {
  const t = useTranslations("scan.workflow")
  const description = engine?.description ?? t(
    engineCatalogStatus === "loading" ? "canvas.engineLibraryLoading" : "canvas.engineUnavailable"
  )

  return (
    <div
      className={cn(
        "group/step radius-control flex min-w-0 items-center gap-2 border px-2.5 py-1.5 transition-[background-color,border-color]",
        styles.stepRow
      )}
      data-engine-unavailable={engineCatalogStatus === "ready" && !engine ? "true" : undefined}
    >
      <ScanEngineIcon className="size-4 shrink-0 text-muted-foreground group-hover/step:text-primary transition-colors" aria-hidden="true" />
      <div className="min-w-0 flex-1">
        <p className={cn("truncate font-medium", textRole.compactPrimary)}>{engine?.displayName ?? step.engineId}</p>
        <p className={cn("mt-0.5 line-clamp-1 text-muted-foreground", textRole.compactCaption)}>{description}</p>
      </div>
    </div>
  )
}

function buildGraph({
  stages,
  selectedStageId,
  dragOverStageId,
  onSelectStage,
  onDropEngine,
  onDragOverStage,
  onAddSerialStageAfter,
  enginesById,
  engineCatalogStatus,
}: {
  stages: DraftStage[]
  selectedStageId: string
  dragOverStageId: string | null
  onSelectStage: (stageId: string) => void
  onDropEngine: (stageId: string, engineId: string) => void
  onDragOverStage: (stageId: string | null) => void
  onAddSerialStageAfter: (stageId: string) => void
  enginesById: ReadonlyMap<string, WorkflowEngineLibraryItem>
  engineCatalogStatus: WorkflowEngineCatalogStatus
}) {
  const nodes: WorkflowStageNode[] = stages.map((stage, stageIndex) => ({
    id: stage.id,
    type: "workflowStage",
    width: WORKFLOW_STAGE_CARD_DEFAULT_WIDTH,
    position: {
      x: stageIndex * WORKFLOW_STAGE_HORIZONTAL_SPACING,
      y: stageIndex % 2 === 0 ? 0 : WORKFLOW_STAGE_VERTICAL_STAGGER,
    },
    data: {
      stage,
      selectedStageId,
      dragOverStageId,
      onSelectStage,
      onDropEngine,
      onDragOverStage,
      isLastStage: stageIndex === stages.length - 1,
      onAddSerialStageAfter,
      enginesById,
      engineCatalogStatus,
    },
  }))

  const edges: Edge[] = []
  for (let stageIndex = 0; stageIndex < stages.length - 1; stageIndex += 1) {
    edges.push({
      id: `${stages[stageIndex].id}-${stages[stageIndex + 1].id}`,
      source: stages[stageIndex].id,
      target: stages[stageIndex + 1].id,
      label: "next",
      animated: true,
    })
  }

  return { nodes, edges }
}

function toScanWorkflowStages(stages: DraftStage[]): ScanWorkflowStageView[] {
  return stages.map((stage) => ({
    stageId: stage.id,
    steps: stage.steps.map((step) => ({
      stageId: stage.id,
      stepId: step.id,
      engineId: step.engineId,
      profileDefaultEnabled: step.profileDefaultEnabled,
    })),
  }))
}

function createWorkflowYaml(stages: DraftStage[]) {
  const lines = [
    "scanWorkflowId: default",
    "displayName: Default Scan",
    "description: Run the default built-in scan workflow: discover subdomains, then scan target-scoped hosts for open ports.",
    "stages:",
  ]
  stages.forEach((stage) => {
    lines.push(`  - stageId: ${stage.id}`)
    lines.push(`    displayName: ${stage.label}`)
    if (stage.steps.length === 0) {
      lines.push("    steps: []")
      return
    }

    lines.push("    steps:")
    stage.steps.forEach((step) => {
      lines.push(`      - stepId: ${step.id}`)
      lines.push(`        engineId: ${step.engineId}`)
      lines.push(`        profileDefaultEnabled: ${step.profileDefaultEnabled}`)
    })
  })
  return lines.join("\n")
}

type WorkflowCompositionCanvasProps = {
  engines: WorkflowEngineLibraryItem[]
  engineCatalogStatus: WorkflowEngineCatalogStatus
  className?: string
  backLabel?: string
  saveLabel?: string
  onBack?: () => void
  onSave?: (stages: ScanWorkflowStageView[]) => void
  stages?: ScanWorkflowStageView[]
  readOnly?: boolean
  displayName?: string
  description?: string
  onDisplayNameChange?: (value: string) => void
  onDescriptionChange?: (value: string) => void
  isSaving?: boolean
}

export function WorkflowCompositionCanvas({
  engines,
  engineCatalogStatus,
  className,
  backLabel,
  saveLabel,
  onBack,
  onSave,
  stages,
  readOnly = false,
  displayName = "",
  description = "",
  onDisplayNameChange,
  onDescriptionChange,
  isSaving = false,
}: WorkflowCompositionCanvasProps) {
  const t = useTranslations("scan.workflow")
  const [draftStages, setDraftStages] = React.useState<DraftStage[]>(() => createInitialDraft(t, stages))
  const [selectedStageId, setSelectedStageId] = React.useState("stage-1")
  const [selectedStepId, setSelectedStepId] = React.useState<string | null>(null)
  const [dragOverStageId, setDragOverStageId] = React.useState<string | null>(null)
  const [stageInspectorOpen, setStageInspectorOpen] = React.useState(false)
  const [deleteStageId, setDeleteStageId] = React.useState<string | null>(null)
  const [renameDialogStageId, setRenameDialogStageId] = React.useState<string | null>(null)
  const [renameStageName, setRenameStageName] = React.useState("")
  const [yamlDialogOpen, setYamlDialogOpen] = React.useState(false)
  const [saveMetadataDialogOpen, setSaveMetadataDialogOpen] = React.useState(false)
  const [isLibraryCollapsed, setIsLibraryCollapsed] = React.useState(false)
  const [layoutVersion, setLayoutVersion] = React.useState(0)
  const hasInitializedLayoutRef = React.useRef(false)
  const nextStageSequenceRef = React.useRef(3)
  const nextStepSequenceRef = React.useRef(100)
  const enginesById = React.useMemo(
    () => new Map(engines.map((engine) => [engine.engineId, engine])),
    [engines]
  )

  React.useEffect(() => {
    if (selectedStepId === null || draftStages.some((stage) => stage.steps.some((step) => step.id === selectedStepId))) return

    const firstStepStage = draftStages.find((stage) => stage.steps.length > 0)
    setSelectedStepId(firstStepStage?.steps[0]?.id ?? null)
    setSelectedStageId(firstStepStage?.id ?? draftStages[0]?.id ?? "stage-1")
  }, [draftStages, selectedStepId])

  const selectedStage = draftStages.find((stage) => stage.id === selectedStageId) ?? draftStages[0]
  const yamlPreview = React.useMemo(() => createWorkflowYaml(draftStages), [draftStages])
  const hasUnavailableEngine = draftStages.some((stage) => (
    stage.steps.some((step) => !enginesById.has(step.engineId))
  ))
  const canSave = engineCatalogStatus === "ready" && !hasUnavailableEngine

  const addStepToStage = React.useCallback((stageId: string, engineId: string) => {
    if (readOnly || engineCatalogStatus !== "ready") return
    const nextStep = createStep(engineId, enginesById, `step-local-${nextStepSequenceRef.current}`)
    nextStepSequenceRef.current += 1
    setDraftStages((currentStages) => currentStages.map((stage) => (
      stage.id === stageId ? { ...stage, steps: [...stage.steps, nextStep] } : stage
    )))
    setSelectedStageId(stageId)
    setSelectedStepId(nextStep.id)
  }, [engineCatalogStatus, enginesById, readOnly])

  const addSerialStageAfter = React.useCallback((afterStageId: string) => {
    if (readOnly) return
    // IDs are stable draft identities, not visual positions; deleted IDs are never reused.
    const nextStageNumber = nextStageSequenceRef.current
    nextStageSequenceRef.current += 1
    const nextStageId = `stage-${nextStageNumber}`
    setDraftStages((currentStages) => {
      const insertAfterIndex = currentStages.findIndex((stage) => stage.id === afterStageId)
      const nextStage = {
        id: nextStageId,
        label: t("canvas.serialStageLabel", { count: nextStageNumber }),
        steps: [],
      }

      if (insertAfterIndex === -1) return [...currentStages, nextStage]
      return [
        ...currentStages.slice(0, insertAfterIndex + 1),
        nextStage,
        ...currentStages.slice(insertAfterIndex + 1),
      ]
    })
    setSelectedStageId(nextStageId)
    setSelectedStepId(null)
  }, [readOnly, t])

  const updateStageLabel = React.useCallback((stageId: string, label: string) => {
    setDraftStages((currentStages) => currentStages.map((stage) => (
      stage.id === stageId ? { ...stage, label } : stage
    )))
  }, [])

  const startStageRename = React.useCallback((stage: DraftStage) => {
    setRenameStageName(stage.label)
    setRenameDialogStageId(stage.id)
  }, [])

  const closeRenameStageDialog = React.useCallback(() => {
    setRenameDialogStageId(null)
    setRenameStageName("")
  }, [])

  const confirmStageRename = React.useCallback(() => {
    const nextLabel = renameStageName.trim()
    if (renameDialogStageId === null || nextLabel.length === 0) return

    updateStageLabel(renameDialogStageId, nextLabel)
    closeRenameStageDialog()
  }, [closeRenameStageDialog, renameDialogStageId, renameStageName, updateStageLabel])

  const deleteStep = React.useCallback((stageId: string, stepId: string) => {
    setDraftStages((currentStages) => currentStages.map((stage) => {
      if (stage.id !== stageId) return stage
      return { ...stage, steps: stage.steps.filter((step) => step.id !== stepId) }
    }))
    if (selectedStepId === stepId) setSelectedStepId(null)
  }, [selectedStepId])

  const deleteStage = React.useCallback((stageId: string) => {
    if (draftStages.length <= 1) return

    const deletedIndex = draftStages.findIndex((stage) => stage.id === stageId)
    if (deletedIndex === -1) return

    const nextStages = draftStages.filter((stage) => stage.id !== stageId)
    const nextSelectedStage = nextStages[deletedIndex] ?? nextStages[deletedIndex - 1]
    setDraftStages(nextStages)
    setStageInspectorOpen(false)
    setSelectedStageId(nextSelectedStage.id)
    setSelectedStepId(nextSelectedStage.steps[0]?.id ?? null)
  }, [draftStages])

  const selectStageForInspection = React.useCallback((stageId: string) => {
    setSelectedStageId(stageId)
    setSelectedStepId(null)
    setStageInspectorOpen(true)
  }, [])

  const confirmStageDelete = React.useCallback(() => {
    if (deleteStageId === null) return
    deleteStage(deleteStageId)
    setDeleteStageId(null)
  }, [deleteStage, deleteStageId])

  const isDirty = React.useMemo(() => {
    if (!stages || stages.length === 0) {
      return draftStages.some((stage) => stage.steps.length > 0) || draftStages.length !== 2
    }
    return JSON.stringify(toScanWorkflowStages(draftStages)) !== JSON.stringify(stages)
  }, [draftStages, stages])

  const [discardDialogOpen, setDiscardDialogOpen] = React.useState(false)

  const handleBack = React.useCallback(() => {
    if (!onBack) return
    if (isDirty) {
      setDiscardDialogOpen(true)
      return
    }
    onBack()
  }, [isDirty, onBack])

  const graph = React.useMemo(
    () => buildGraph({
      stages: draftStages,
      selectedStageId,
      dragOverStageId,
      onSelectStage: selectStageForInspection,
      onDropEngine: addStepToStage,
      onDragOverStage: setDragOverStageId,
      onAddSerialStageAfter: addSerialStageAfter,
      enginesById,
      engineCatalogStatus,
    }),
    [addSerialStageAfter, addStepToStage, draftStages, dragOverStageId, engineCatalogStatus, enginesById, selectedStageId, selectStageForInspection]
  )
  const [flowNodes, setFlowNodes, onNodesChange] = useNodesState<WorkflowStageNode>(graph.nodes)

  React.useEffect(() => {
    setFlowNodes((currentNodes) => {
      const currentNodesById = new Map(currentNodes.map((node) => [node.id, node]))

      const hasMatchingGraphData = currentNodes.length === graph.nodes.length && graph.nodes.every((node) => {
        const currentNode = currentNodesById.get(node.id)
        if (!currentNode) return false

        return currentNode.data.stage === node.data.stage
          && currentNode.data.selectedStageId === node.data.selectedStageId
          && currentNode.data.dragOverStageId === node.data.dragOverStageId
          && currentNode.data.isLastStage === node.data.isLastStage
          && currentNode.data.enginesById === node.data.enginesById
          && currentNode.data.engineCatalogStatus === node.data.engineCatalogStatus
      })

      if (hasMatchingGraphData) return currentNodes

      return graph.nodes.map((node) => {
        const currentNode = currentNodesById.get(node.id)
        return currentNode ? { ...node, position: currentNode.position } : node
      })
    })
  }, [graph.nodes, setFlowNodes])

  const autoLayoutStages = React.useCallback(() => {
    setFlowNodes((currentNodes) => currentNodes.map((node, index) => ({
      ...node,
      position: {
        x: index * WORKFLOW_STAGE_HORIZONTAL_SPACING,
        y: index % 2 === 0 ? 0 : WORKFLOW_STAGE_VERTICAL_STAGGER,
      },
    })))
    setLayoutVersion((currentVersion) => currentVersion + 1)
  }, [setFlowNodes])

  React.useEffect(() => {
    if (hasInitializedLayoutRef.current || flowNodes.length === 0) return

    // Arrange once per editor mount; later draft updates must preserve manual node movement.
    hasInitializedLayoutRef.current = true
    autoLayoutStages()
  }, [autoLayoutStages, flowNodes.length])

  const handleSave = React.useCallback(() => {
    if (readOnly || !onSave || !canSave) return
    if (!displayName.trim()) {
      setSaveMetadataDialogOpen(true)
      return
    }

    onSave(toScanWorkflowStages(draftStages))
  }, [canSave, displayName, draftStages, onSave, readOnly])

  return (
    <section
      className={cn("relative flex h-full min-h-0 flex-1 overflow-hidden", styles.workbench, className)}
      data-workflow-composition-canvas="mock-only"
      data-workflow-builder="floating-canvas"
      data-workspace-width="full"
    >
      {!readOnly && !isLibraryCollapsed ? (
        <EngineLibraryPanel
          engines={engines}
          status={engineCatalogStatus}
          onEngineClick={(engineId) => addStepToStage(selectedStage?.id ?? selectedStageId, engineId)}
          onCollapse={() => setIsLibraryCollapsed(true)}
        />
      ) : null}
      <div className={cn("relative flex min-h-0 flex-1 flex-col overflow-hidden", styles.canvasArea)}>
        <div className={cn("absolute inset-0", styles.canvasShell)} data-workflow-builder-canvas="serial-parallel">
          <ReactFlow
            nodes={flowNodes}
            edges={graph.edges}
            nodeTypes={nodeTypes}
            defaultEdgeOptions={defaultEdgeOptions}
            nodesDraggable={!readOnly}
            onNodesChange={onNodesChange}
            nodesConnectable={false}
            elementsSelectable={false}
            panOnDrag
            zoomOnScroll
            zoomOnDoubleClick={false}
            minZoom={WORKFLOW_CANVAS_MIN_ZOOM}
            maxZoom={WORKFLOW_CANVAS_MAX_ZOOM}
            defaultViewport={{ x: 355, y: 265, zoom: 1 }}
            proOptions={{ hideAttribution: true }}
          >
            <Background gap={24} size={1} />
            <WorkflowCanvasActions
              saveLabel={saveLabel ?? t("management.save")}
              layoutVersion={layoutVersion}
              onSave={readOnly || !onSave ? undefined : handleSave}
              saveDisabled={!canSave}
              onAutoLayout={autoLayoutStages}
              onOpenYaml={() => setYamlDialogOpen(true)}
              isSaving={isSaving}
              inspectorOpen={stageInspectorOpen && Boolean(selectedStage)}
              isLibraryCollapsed={isLibraryCollapsed}
            />
          </ReactFlow>
        </div>
        <WorkflowCanvasHeader
          backLabel={backLabel ?? t("management.backToList")}
          onBack={onBack ? handleBack : undefined}
          displayName={displayName}
          inspectorOpen={stageInspectorOpen && Boolean(selectedStage)}
          isLibraryCollapsed={isLibraryCollapsed}
          onExpandLibrary={() => setIsLibraryCollapsed(false)}
          readOnly={readOnly}
          onEditMetadata={onDisplayNameChange && !readOnly ? () => setSaveMetadataDialogOpen(true) : undefined}
          isDirty={isDirty}
        />
      </div>
      <WorkflowStageInspector
        open={stageInspectorOpen}
        stage={selectedStage}
        canDeleteStage={draftStages.length > 1}
        readOnly={readOnly}
        enginesById={enginesById}
        engineCatalogStatus={engineCatalogStatus}
        onOpenChange={setStageInspectorOpen}
        onRename={startStageRename}
        onRemoveEngine={deleteStep}
        onDelete={() => setDeleteStageId(selectedStage?.id ?? null)}
        onAddStageAfter={addSerialStageAfter}
      />
      <WorkflowRenameStageDialog
        open={renameDialogStageId !== null}
        name={renameStageName}
        onNameChange={setRenameStageName}
        onOpenChange={(open) => {
          if (!open) closeRenameStageDialog()
        }}
        onConfirm={confirmStageRename}
        onCancel={closeRenameStageDialog}
      />
      <WorkflowDeleteStageDialog
        open={deleteStageId !== null}
        stageName={draftStages.find((stage) => stage.id === deleteStageId)?.label ?? ""}
        onOpenChange={(open) => {
          if (!open) setDeleteStageId(null)
        }}
        onConfirm={confirmStageDelete}
      />
      <WorkflowDiscardDraftDialog
        open={discardDialogOpen}
        onOpenChange={setDiscardDialogOpen}
        onConfirm={() => {
          setDiscardDialogOpen(false)
          onBack?.()
        }}
      />
      <WorkflowYamlDialog
        open={yamlDialogOpen}
        onOpenChange={setYamlDialogOpen}
        yamlPreview={yamlPreview}
      />
      <WorkflowSaveMetadataDialog
        open={saveMetadataDialogOpen}
        displayName={displayName}
        description={description}
        isSaving={isSaving}
        saveDisabled={!canSave}
        onDisplayNameChange={onDisplayNameChange}
        onDescriptionChange={onDescriptionChange}
        onOpenChange={setSaveMetadataDialogOpen}
        onConfirm={() => {
          handleSave()
          setSaveMetadataDialogOpen(false)
        }}
      />
    </section>
  )
}

function WorkflowCanvasHeader({
  backLabel,
  onBack,
  displayName,
  inspectorOpen,
  isLibraryCollapsed,
  onExpandLibrary,
  readOnly,
  onEditMetadata,
  isDirty,
}: {
  backLabel: string
  onBack?: () => void
  displayName: string
  inspectorOpen?: boolean
  isLibraryCollapsed?: boolean
  onExpandLibrary?: () => void
  readOnly?: boolean
  onEditMetadata?: () => void
  isDirty?: boolean
}) {
  const t = useTranslations("scan.workflow")

  if (!onBack) return null

  return (
    <header
      className={styles.canvasHeader}
      data-library-open={!readOnly && !isLibraryCollapsed ? "true" : undefined}
      data-inspector-open={inspectorOpen ? "true" : undefined}
    >
      <div className="flex items-center gap-1.5">
        <Button type="button" variant="ghost" size="icon-sm" aria-label={backLabel} onClick={onBack}>
          <ArrowLeft className="size-4" />
        </Button>
        {isLibraryCollapsed && !readOnly && onExpandLibrary ? (
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label={t("canvas.expandEngineLibrary")}
            onClick={onExpandLibrary}
            title={t("canvas.expandEngineLibrary")}
          >
            <PanelLeftIcon className="size-4" />
          </Button>
        ) : null}
      </div>
      <div className="flex min-w-0 items-center gap-2">
        <h1 className={cn(textRole.sectionTitle, styles.canvasHeaderTitle)}>{displayName || t("canvas.editorTitle")}</h1>
        {onEditMetadata && !readOnly ? (
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            layout="compactAction"
            aria-label={t("canvas.editWorkflowMetadata")}
            onClick={onEditMetadata}
            title={t("canvas.editWorkflowMetadata")}
          >
            <semanticIcons.action.edit className="size-3.5 text-muted-foreground" />
          </Button>
        ) : null}
        {isDirty ? (
          <Badge variant="outline" className="shrink-0 border-dashed text-[11px] text-muted-foreground">
            {t("canvas.mockBadge")}
          </Badge>
        ) : null}
      </div>
    </header>
  )
}

function WorkflowSaveMetadataDialog({
  open,
  displayName,
  description,
  isSaving,
  saveDisabled,
  onDisplayNameChange,
  onDescriptionChange,
  onOpenChange,
  onConfirm,
}: {
  open: boolean
  displayName: string
  description: string
  isSaving: boolean
  saveDisabled: boolean
  onDisplayNameChange?: (value: string) => void
  onDescriptionChange?: (value: string) => void
  onOpenChange: (open: boolean) => void
  onConfirm: () => void
}) {
  const t = useTranslations("scan.workflow")

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className={narrowFormDialogContentClassName}>
        <form
          onSubmit={(event) => {
            event.preventDefault()
            if (!displayName.trim() || saveDisabled) return
            onConfirm()
          }}
        >
          <DialogHeader>
            <DialogTitle>{t("canvas.saveMetadataTitle")}</DialogTitle>
            <DialogDescription>{t("canvas.saveMetadataDescription")}</DialogDescription>
          </DialogHeader>
          <div className="grid gap-3 py-3">
            <div className="grid gap-2">
              <Label htmlFor="workflow-display-name">{t("canvas.workflowName")}</Label>
              <Input
                id="workflow-display-name"
                value={displayName}
                onChange={(event) => onDisplayNameChange?.(event.target.value)}
                autoFocus
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="workflow-description">{t("canvas.workflowDescription")}</Label>
              <Input
                id="workflow-description"
                value={description}
                onChange={(event) => onDescriptionChange?.(event.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <Button type="button" variant="surface" onClick={() => onOpenChange(false)}>
              {t("canvas.cancelStageRename")}
            </Button>
            <Button type="submit" loading={isSaving} disabled={!displayName.trim() || saveDisabled}>
              {t("canvas.confirmSaveMetadata")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function WorkflowCanvasActions({
  saveLabel,
  layoutVersion,
  onSave,
  onAutoLayout,
  onOpenYaml,
  isSaving,
  saveDisabled,
  inspectorOpen,
  isLibraryCollapsed = false,
}: {
  saveLabel: string
  layoutVersion: number
  onSave?: () => void
  onAutoLayout: () => void
  onOpenYaml: () => void
  isSaving: boolean
  saveDisabled: boolean
  inspectorOpen?: boolean
  isLibraryCollapsed?: boolean
}) {
  const t = useTranslations("scan.workflow")
  const canvasWidth = useStore((state) => state.width)
  const canvasHeight = useStore((state) => state.height)
  const { getNodes, getNodesBounds, setViewport } = useReactFlow()
  const lastHandledLayoutVersionRef = React.useRef(0)

  React.useEffect(() => {
    if (layoutVersion === 0 || canvasWidth === 0 || canvasHeight === 0) return
    if (lastHandledLayoutVersionRef.current === layoutVersion) return

    const isFirstLayout = lastHandledLayoutVersionRef.current === 0
    lastHandledLayoutVersionRef.current = layoutVersion

    const libraryOcclusion = (typeof window === "undefined" || window.matchMedia("(min-width: 768px)").matches) && !isLibraryCollapsed
      ? WORKFLOW_ENGINE_LIBRARY_OCCLUSION_PX
      : 0
    const availableWidth = Math.max(canvasWidth - libraryOcclusion, 100)
    const bounds = resolveNodesBounds(getNodes(), getNodesBounds)
    const viewport = getViewportForBounds(
      bounds,
      availableWidth,
      canvasHeight,
      WORKFLOW_CANVAS_MIN_ZOOM,
      1,
      0.16,
    )
    viewport.x += libraryOcclusion

    if (isFirstLayout) {
      void setViewport(viewport)
    } else {
      void setViewport(viewport, { duration: 200 })
    }
  }, [canvasHeight, canvasWidth, getNodes, getNodesBounds, isLibraryCollapsed, layoutVersion, setViewport])

  return (
    <>
      <Panel
        position="top-right"
        className={cn("nodrag nopan nowheel", styles.canvasPrimaryActions)}
        data-inspector-open={inspectorOpen ? "true" : undefined}
      >
        <Button
          type="button"
          variant="ghost"
          size="sm"
          aria-label={t("canvas.yamlConfig")}
          onClick={onOpenYaml}
          title={t("canvas.yamlConfig")}
        >
          <FileCode className="size-4" />
          <span className={styles.canvasActionLabel}>{t("canvas.yamlConfig")}</span>
        </Button>
        {onSave ? (
          <Button type="button" size="sm" aria-label={saveLabel} onClick={onSave} loading={isSaving} disabled={saveDisabled}>
            <Save className="size-4" />
            <span className={styles.canvasActionLabel}>{saveLabel}</span>
          </Button>
        ) : null}
      </Panel>
      <Panel
        position="bottom-right"
        className={cn("nodrag nopan nowheel", styles.canvasActions)}
        data-inspector-open={inspectorOpen ? "true" : undefined}
      >
        <Button
          type="button"
          variant="surface"
          size="icon-sm"
          aria-label={t("canvas.autoLayout")}
          onClick={onAutoLayout}
        >
          <FocusCentered className="size-4" />
        </Button>
        <WorkflowCanvasZoomControls />
      </Panel>
    </>
  )
}

function WorkflowRenameStageDialog({
  open,
  name,
  onNameChange,
  onOpenChange,
  onConfirm,
  onCancel,
}: {
  open: boolean
  name: string
  onNameChange: (value: string) => void
  onOpenChange: (open: boolean) => void
  onConfirm: () => void
  onCancel: () => void
}) {
  const t = useTranslations("scan.workflow")

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className={narrowFormDialogContentClassName}>
        <form
          onSubmit={(event) => {
            event.preventDefault()
            onConfirm()
          }}
        >
          <DialogHeader>
            <DialogTitle>{t("canvas.renameStageTitle")}</DialogTitle>
            <DialogDescription>{t("canvas.renameStageDescription")}</DialogDescription>
          </DialogHeader>
          <div className="py-3">
            <Label htmlFor="workflow-stage-display-name">{t("canvas.stageDisplayName")}</Label>
            <Input
              id="workflow-stage-display-name"
              className="mt-2"
              value={name}
              autoFocus
              onChange={(event) => onNameChange(event.target.value)}
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="surface" onClick={onCancel}>
              {t("canvas.cancelStageRename")}
            </Button>
            <Button type="submit" disabled={name.trim().length === 0}>
              {t("canvas.confirmStageRename")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function WorkflowStageInspector({
  open,
  stage,
  canDeleteStage,
  readOnly,
  enginesById,
  engineCatalogStatus,
  onOpenChange,
  onRename,
  onRemoveEngine,
  onDelete,
  onAddStageAfter,
}: {
  open: boolean
  stage?: DraftStage
  canDeleteStage: boolean
  readOnly: boolean
  enginesById: ReadonlyMap<string, WorkflowEngineLibraryItem>
  engineCatalogStatus: WorkflowEngineCatalogStatus
  onOpenChange: (open: boolean) => void
  onRename: (stage: DraftStage) => void
  onRemoveEngine: (stageId: string, stepId: string) => void
  onDelete: () => void
  onAddStageAfter?: (stageId: string) => void
}) {
  const t = useTranslations("scan.workflow")
  const tActions = useTranslations("common.actions")
  const WorkflowIcon = semanticIcons.concept.workflow

  if (!open || !stage) return null

  const isParallel = stage.steps.length > 1

  return (
    <aside
      className={cn("flex min-h-0 flex-col overflow-hidden animate-in fade-in-0 slide-in-from-right-4 duration-200", styles.dockedInspectorPanel)}
      data-stage-inspector="workflow-builder"
    >
      <div className="flex h-12 shrink-0 items-center justify-between border-b px-3">
        <div className="flex min-w-0 items-center gap-2">
          <WorkflowIcon className="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
          <h2 className={cn("truncate font-medium", textRole.compactPrimary)}>
            {stage.label}
          </h2>
          <Badge variant="secondary" className="shrink-0">
            {t("canvas.stepCount", { count: stage.steps.length })}
          </Badge>
        </div>
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          layout="compactAction"
          aria-label={tActions("close")}
          onClick={() => onOpenChange(false)}
        >
          <PanelRightCloseIcon className="size-4" />
        </Button>
      </div>

      <div className="flex min-h-0 flex-1 flex-col">
        <div className={DETAIL_DRAWER_COMPACT_BODY_CLASS}>
          <div className="space-y-4">
            <section className="space-y-2 rounded-lg border bg-muted/20 p-2.5">
              <div className="flex items-center justify-between gap-2">
                <span className="text-xs font-medium text-muted-foreground">
                  {t("canvas.executionStrategy")}
                </span>
                <Badge variant="secondary" data-stage-execution-mode={isParallel ? "parallel" : "serial"}>
                  {t(isParallel ? "canvas.parallelExecution" : "canvas.serialExecution")}
                </Badge>
              </div>
              <p className="text-xs text-muted-foreground leading-relaxed">
                {stage.steps.length === 1
                  ? t("canvas.executionModeSerialDesc")
                  : stage.steps.length > 1
                  ? t("canvas.executionModeParallelDesc")
                  : t("canvas.emptyStageHelp")}
              </p>
            </section>

            <section className="space-y-2">
              <div className="flex items-center justify-between">
                <h3 className={textRole.sectionTitle}>{t("canvas.stageEngines")}</h3>
                <span className="text-xs text-muted-foreground tabular-nums">{stage.steps.length}</span>
              </div>
              {stage.steps.length ? (
                <div className="space-y-1.5">
                  {stage.steps.map((step) => {
                    const engine = enginesById.get(step.engineId)
                    return (
                        <div
                          key={step.id}
                          className={cn(
                            "group radius-control flex min-w-0 items-center gap-2.5 border p-2 transition-colors",
                            styles.stepRow
                          )}
                          data-engine-unavailable={engineCatalogStatus === "ready" && !engine ? "true" : undefined}
                        >
                          <ScanEngineIcon className="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
                          <div className="min-w-0 flex-1">
                          <p className={cn("truncate font-medium", textRole.compactPrimary)}>
                            {engine?.displayName ?? step.engineId}
                          </p>
                          <p className={cn("line-clamp-1 text-muted-foreground", textRole.compactCaption)}>
                            {engine ? engine.description : t(engineCatalogStatus === "loading" ? "canvas.engineLibraryLoading" : "canvas.engineUnavailable")}
                          </p>
                        </div>
                        <Button
                          type="button"
                          variant="quiet"
                          size="icon-sm"
                          layout="rowAction"
                          aria-label={t("canvas.removeEngine")}
                          disabled={readOnly}
                          onClick={() => onRemoveEngine(stage.id, step.id)}
                        >
                          <IconTrash className="size-4" />
                        </Button>
                      </div>
                    )
                  })}
                </div>
              ) : (
                <p className={cn("radius-control flex h-12 items-center justify-center border border-dashed px-3 py-2 text-center", textRole.caption)}>
                  {t("canvas.emptyStagePlaceholder")}
                </p>
              )}
            </section>
          </div>
        </div>
        <div className={cn("space-y-2", DETAIL_DRAWER_COMPACT_FOOTER_CLASS)}>
          {stage ? (
            <>
              {onAddStageAfter ? (
                <Button type="button" variant="surface" size="sm" layout="fullWidth" disabled={readOnly} onClick={() => onAddStageAfter(stage.id)}>
                  <IconPlus className="size-3.5" />
                  {t("canvas.addStageAfter")}
                </Button>
              ) : null}
              <Button type="button" variant="surface" size="sm" layout="fullWidth" disabled={readOnly} onClick={() => onRename(stage)}>
                <semanticIcons.action.edit className="size-3.5" />
                {t("canvas.renameStage")}
              </Button>
            </>
          ) : null}
          <Button
            type="button"
            variant="outline"
            size="sm"
            layout="fullWidth"
            className="border-destructive/30 text-destructive hover:border-destructive/60 hover:bg-destructive/10 hover:text-destructive dark:border-destructive/40 dark:hover:bg-destructive/20"
            disabled={readOnly || !canDeleteStage}
            onClick={onDelete}
          >
            <IconTrash className="size-3.5" />
            {t("canvas.deleteStage")}
          </Button>
        </div>
      </div>
    </aside>
  )
}

function WorkflowDeleteStageDialog({
  open,
  stageName,
  onOpenChange,
  onConfirm,
}: {
  open: boolean
  stageName: string
  onOpenChange: (open: boolean) => void
  onConfirm: () => void
}) {
  const t = useTranslations("scan.workflow")

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("canvas.deleteStageTitle")}</AlertDialogTitle>
          <AlertDialogDescription>{t("canvas.deleteStageDescription", { name: stageName })}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogClose variant="outline">{t("canvas.cancelStageDelete")}</AlertDialogClose>
          <AlertDialogClose variant="destructive" onClick={onConfirm}>
            {t("canvas.confirmStageDelete")}
          </AlertDialogClose>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

function WorkflowDiscardDraftDialog({
  open,
  onOpenChange,
  onConfirm,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: () => void
}) {
  const t = useTranslations("scan.workflow")

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("canvas.discardDraftTitle")}</AlertDialogTitle>
          <AlertDialogDescription>{t("canvas.discardDraftDescription")}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogClose variant="outline">{t("canvas.cancelDiscard")}</AlertDialogClose>
          <AlertDialogClose variant="destructive" onClick={onConfirm}>
            {t("canvas.confirmDiscard")}
          </AlertDialogClose>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

function EngineLibraryPanel({
  engines,
  status,
  onEngineClick,
  onCollapse,
}: {
  engines: WorkflowEngineLibraryItem[]
  status: WorkflowEngineCatalogStatus
  onEngineClick: (engineId: string) => void
  onCollapse?: () => void
}) {
  const t = useTranslations("scan.workflow")

  return (
    <aside className={cn("flex min-h-0 flex-col overflow-hidden animate-in fade-in-0 slide-in-from-left-4 duration-200", styles.dockedEnginePanel)} data-engine-library="workflow-builder">
      <div className="flex h-12 shrink-0 items-center justify-between border-b px-3">
        <div className="flex items-center gap-2">
          <h2 className={cn("font-medium", textRole.compactPrimary)}>{t("canvas.engineLibrary")}</h2>
          <span className="rounded-full bg-muted px-1.5 py-0.5 text-[10px] font-semibold text-muted-foreground">
            {engines.length}
          </span>
        </div>
        {onCollapse ? (
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            layout="compactAction"
            aria-label={t("canvas.collapseEngineLibrary")}
            onClick={onCollapse}
          >
            <PanelLeftCloseIcon className="size-4" />
          </Button>
        ) : null}
      </div>

      <div className="min-h-0 flex-1 space-y-1.5 overflow-y-auto p-3">
        {status === "loading" ? (
          <p aria-live="polite" className={textRole.helperText}>{t("canvas.engineLibraryLoading")}</p>
        ) : status === "error" ? (
          <p role="alert" className={textRole.helperText}>{t("canvas.engineLibraryError")}</p>
        ) : engines.length === 0 ? (
          <p className={textRole.helperText}>{t("canvas.engineLibraryEmpty")}</p>
        ) : (
          engines.map((engine) => {
            return (
              <Button
                key={engine.engineId}
                type="button"
                variant="surface"
                size="sm"
                draggable
                className={cn(
                  "group h-auto w-full min-w-0 items-center gap-2.5 px-2.5 py-2 text-left whitespace-normal",
                  styles.engineButton
                )}
                onClick={() => onEngineClick(engine.engineId)}
                onDragStart={(event) => {
                  event.dataTransfer.setData("application/x-lunafox-engine", engine.engineId)
                  event.dataTransfer.effectAllowed = "copy"
                }}
              >
                <ScanEngineIcon className="size-4 shrink-0 text-muted-foreground group-hover:text-primary transition-colors" aria-hidden="true" />
                <span className="min-w-0 flex-1">
                  <span className="block min-w-0">
                    <span className={cn("truncate block font-medium", textRole.compactPrimary)}>
                      {engine.displayName}
                    </span>
                  </span>
                  <span className={cn("mt-0.5 block line-clamp-1", textRole.compactCaption)}>
                    {engine.description}
                  </span>
                </span>
                <GripVerticalIcon className="size-4 shrink-0 self-center text-muted-foreground" aria-hidden="true" />
              </Button>
            )
          })
        )}
      </div>

      <div className={cn("flex items-center justify-center gap-2 border-t px-3 py-2.5 text-center", textRole.helperText)}>
        <IconShieldPlus className="size-3.5 shrink-0 text-highlight" aria-hidden="true" />
        <span className="min-w-0 text-[11px]">{t("canvas.engineLibraryHint")}</span>
      </div>
    </aside>
  )
}

function WorkflowYamlDialog({
  open,
  onOpenChange,
  yamlPreview,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  yamlPreview: string
}) {
  const t = useTranslations("scan.workflow")
  const tActions = useTranslations("common.actions")
  const tToast = useTranslations("toast")

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="gap-0 overflow-hidden p-0 sm:max-w-3xl">
        <DialogHeader className="border-b px-4 py-3 pr-14">
          <DialogTitle>{t("canvas.yamlConfig")}</DialogTitle>
          <DialogDescription>{t("canvas.yamlHint")}</DialogDescription>
        </DialogHeader>
        <div className="relative px-4 py-3" data-yaml-inspector="workflow-builder">
          <div className="absolute top-5 right-6 z-10">
            <CopyButton
              value={yamlPreview}
              copyLabel={tActions("copy")}
              copiedLabel={tToast("copied")}
              copyFailedLabel={tToast("copyFailed")}
              toastId="workflow-yaml-preview"
            />
          </div>
          <pre className={cn("max-h-96 overflow-auto rounded-md border bg-muted/30 p-4 pr-12 whitespace-pre-wrap break-words", textRole.code)}>{yamlPreview}</pre>
        </div>
      </DialogContent>
    </Dialog>
  )
}
