"use client"

import * as React from "react"
import {
  Background,
  getViewportForBounds,
  Handle,
  MarkerType,
  MiniMap,
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
import { DetailDrawer } from "@/components/shared/detail-drawer"
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
  Save,
  semanticIcons,
} from "@/components/icons"
import type { WorkflowEngineLibraryItem } from "@/lib/engine-catalog"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
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

// Desktop auto-layout centers within the area not obscured by the floating engine library.
const WORKFLOW_ENGINE_LIBRARY_OCCLUSION_PX = 304

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
        "radius-overlay relative w-72 border shadow-2xs transition-[background-color,border-color,box-shadow]",
        styles.stageNode
      )}
      onClick={() => data.onSelectStage(data.stage.id)}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
    >
      <Handle type="target" position={Position.Left} />
      <div className="flex min-w-0 items-center justify-between gap-2 border-b px-3 py-3">
        <div className="flex min-w-0 items-center gap-2">
          <WorkflowIcon className="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
          <p className={cn("min-w-0 truncate", textRole.compactSectionTitle)}>{data.stage.label}</p>
        </div>
        {data.stage.steps.length > 0 ? (
          <Badge variant="secondary" data-stage-execution-mode={data.stage.steps.length === 1 ? "serial" : "parallel"}>
            {t(data.stage.steps.length === 1 ? "canvas.serialExecution" : "canvas.parallelExecution")}
          </Badge>
        ) : null}
      </div>

      <div className="flex flex-col gap-2 p-3">
        {data.stage.steps.length === 0 ? (
          <div
            className="radius-control flex h-12 items-center justify-center gap-2 border border-dashed px-3 py-2 text-center"
            data-empty-stage-placeholder
          >
            <IconPlus className="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
            <p className={textRole.compactPrimary}>{t("canvas.emptyStageDropHint")}</p>
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
      className={cn("radius-control min-w-0 border px-3 py-2", styles.stepRow)}
      data-engine-unavailable={engineCatalogStatus === "ready" && !engine ? "true" : undefined}
    >
      <p className={cn("truncate", textRole.compactPrimary)}>{engine?.displayName ?? step.engineId}</p>
      <p className={cn("mt-1 line-clamp-1", textRole.compactCaption)}>{description}</p>
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
    >
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
          fitView
          fitViewOptions={{ padding: 0.16, maxZoom: 1 }}
          proOptions={{ hideAttribution: true }}
        >
          <Background gap={24} size={1.5} />
          <MiniMap
            position="bottom-left"
            style={{ height: 120 }}
            pannable
            zoomable
            ariaLabel={t("canvas.miniMap")}
            className={styles.canvasMiniMap}
          />
          <WorkflowCanvasActions
            saveLabel={saveLabel ?? t("management.save")}
            layoutVersion={layoutVersion}
            onSave={readOnly || !onSave ? undefined : handleSave}
            saveDisabled={!canSave}
            onAutoLayout={autoLayoutStages}
            onOpenYaml={() => setYamlDialogOpen(true)}
            isSaving={isSaving}
          />
        </ReactFlow>
      </div>
      <WorkflowCanvasHeader
        backLabel={backLabel ?? t("management.backToList")}
        onBack={onBack}
        displayName={displayName}
      />
      {!readOnly ? (
        <EngineLibraryPanel
          engines={engines}
          status={engineCatalogStatus}
          onEngineClick={(engineId) => addStepToStage(selectedStage?.id ?? selectedStageId, engineId)}
        />
      ) : null}
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
}: {
  backLabel: string
  onBack?: () => void
  displayName: string
}) {
  const t = useTranslations("scan.workflow")

  if (!onBack) return null

  return (
    <header className={styles.canvasHeader}>
      <Button type="button" variant="surface" size="icon-sm" aria-label={backLabel} onClick={onBack}>
        <ArrowLeft className="size-4" />
      </Button>
      <h1 className={cn(textRole.sectionTitle, styles.canvasHeaderTitle)}>{displayName || t("canvas.editorTitle")}</h1>
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
      <DialogContent className="sm:max-w-md">
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
          <div className="grid gap-4 py-5">
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
}: {
  saveLabel: string
  layoutVersion: number
  onSave?: () => void
  onAutoLayout: () => void
  onOpenYaml: () => void
  isSaving: boolean
  saveDisabled: boolean
}) {
  const t = useTranslations("scan.workflow")
  const canvasWidth = useStore((state) => state.width)
  const canvasHeight = useStore((state) => state.height)
  const { getNodes, getNodesBounds, setViewport } = useReactFlow()

  React.useEffect(() => {
    if (layoutVersion === 0 || canvasWidth === 0 || canvasHeight === 0) return

    const libraryOcclusion = window.matchMedia("(min-width: 768px)").matches
      ? WORKFLOW_ENGINE_LIBRARY_OCCLUSION_PX
      : 0
    const viewport = getViewportForBounds(
      getNodesBounds(getNodes()),
      canvasWidth - libraryOcclusion,
      canvasHeight,
      WORKFLOW_CANVAS_MIN_ZOOM,
      1,
      0.16,
    )

    // The usable area starts after the desktop engine library, so calculate and apply one final viewport.
    viewport.x += libraryOcclusion
    void setViewport(viewport, { duration: 200 })
  }, [canvasHeight, canvasWidth, getNodes, getNodesBounds, layoutVersion, setViewport])

  return (
    <>
      <Panel position="top-right" className={cn("nodrag nopan nowheel", styles.canvasPrimaryActions)}>
        <Button type="button" variant="surface" size="sm" aria-label={t("canvas.yamlConfig")} onClick={onOpenYaml}>
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
      <Panel position="bottom-right" className={cn("nodrag nopan nowheel", styles.canvasActions)}>
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
      <DialogContent className="sm:max-w-md">
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
          <div className="py-5">
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
}) {
  const t = useTranslations("scan.workflow")

  return (
    <DetailDrawer
      open={open}
      onOpenChange={onOpenChange}
      title={stage?.label ?? ""}
      description={t("canvas.stageInspectorDescription")}
      titleMeta={<Badge variant="secondary">{t("canvas.stepCount", { count: stage?.steps.length ?? 0 })}</Badge>}
      className="sm:max-w-md"
    >
      <div className="flex min-h-0 flex-1 flex-col" data-stage-inspector="workflow-builder">
        <div className="min-h-0 flex-1 overflow-y-auto px-6 py-5">
          <section className="space-y-3">
            <h2 className={textRole.sectionTitle}>{t("canvas.stageEngines")}</h2>
            {stage?.steps.length ? (
              <div className="space-y-2">
                {stage.steps.map((step) => {
                  const engine = enginesById.get(step.engineId)
                  return (
                    <div
                      key={step.id}
                      className="radius-control flex min-w-0 items-center gap-3 border px-3 py-2"
                      data-engine-unavailable={engineCatalogStatus === "ready" && !engine ? "true" : undefined}
                    >
                      <div className="min-w-0 flex-1">
                        <p className={cn("truncate", textRole.tableCellPrimary)}>{engine?.displayName ?? step.engineId}</p>
                        <p className={cn("mt-1 truncate", textRole.caption)}>
                          {engine ? step.engineId : t(engineCatalogStatus === "loading" ? "canvas.engineLibraryLoading" : "canvas.engineUnavailable")}
                        </p>
                      </div>
                      <Button
                        type="button"
                        variant="quiet"
                        size="icon-sm"
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
        <div className="space-y-2 border-t px-6 py-4">
          {stage ? (
            <Button type="button" variant="surface" layout="fullWidth" disabled={readOnly} onClick={() => onRename(stage)}>
              <semanticIcons.action.edit className="size-4" />
              {t("canvas.renameStage")}
            </Button>
          ) : null}
          <Button type="button" variant="destructive" layout="fullWidth" disabled={readOnly || !canDeleteStage} onClick={onDelete}>
            <IconTrash className="size-4" />
            {t("canvas.deleteStage")}
          </Button>
        </div>
      </div>
    </DetailDrawer>
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

function EngineLibraryPanel({
  engines,
  status,
  onEngineClick,
}: {
  engines: WorkflowEngineLibraryItem[]
  status: WorkflowEngineCatalogStatus
  onEngineClick: (engineId: string) => void
}) {
  const t = useTranslations("scan.workflow")

  return (
    <aside className={cn("flex min-h-0 flex-col overflow-hidden", styles.floatingEnginePanel)} data-engine-library="workflow-builder">
      <div className="border-b p-4">
        <div className="flex items-center justify-between gap-3">
          <h2 className={textRole.sectionTitle}>{t("canvas.engineLibrary")}</h2>
        </div>
      </div>

      <div className="min-h-0 flex-1 space-y-2 overflow-y-auto p-4">
        {status === "loading" ? (
          <p aria-live="polite" className={textRole.helperText}>{t("canvas.engineLibraryLoading")}</p>
        ) : status === "error" ? (
          <p role="alert" className={textRole.helperText}>{t("canvas.engineLibraryError")}</p>
        ) : engines.length === 0 ? (
          <p className={textRole.helperText}>{t("canvas.engineLibraryEmpty")}</p>
        ) : engines.map((engine) => {
          return (
            <Button
              key={engine.engineId}
              type="button"
              variant="surface"
              size="sm"
              draggable
              className={cn("h-auto w-full min-w-0 items-start gap-2 px-3 py-2 text-left whitespace-normal", styles.engineButton)}
              onClick={() => onEngineClick(engine.engineId)}
              onDragStart={(event) => {
                event.dataTransfer.setData("application/x-lunafox-engine", engine.engineId)
                event.dataTransfer.effectAllowed = "copy"
              }}
            >
              <span className="min-w-0 flex-1">
                <span className="block min-w-0">
                  <span className={cn("truncate", textRole.compactPrimary)}>{engine.displayName}</span>
                </span>
                <span className={cn("mt-1 block line-clamp-1", textRole.compactCaption)}>{engine.description}</span>
              </span>
              <GripVerticalIcon className="size-4 shrink-0 self-center text-muted-foreground" aria-hidden="true" />
            </Button>
          )
        })}
      </div>

      <div className={cn("hidden items-center justify-center gap-2 border-t px-4 py-3 text-center md:flex", textRole.helperText)}>
        <IconShieldPlus className="size-4 shrink-0 text-highlight" aria-hidden="true" />
        <span className="min-w-0">{t("canvas.engineLibraryHint")}</span>
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
        <DialogHeader className="border-b px-6 py-4 pr-12">
          <DialogTitle>{t("canvas.yamlConfig")}</DialogTitle>
          <DialogDescription>{t("canvas.yamlHint")}</DialogDescription>
        </DialogHeader>
        <div className="relative p-6" data-yaml-inspector="workflow-builder">
          <div className="absolute top-8 right-8 z-10">
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
