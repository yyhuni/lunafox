import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/workflow/workflow-composition-canvas.tsx"), "utf8")
const styles = readFileSync(path.resolve(process.cwd(), "components/scan/workflow/workflow-composition-canvas.module.css"), "utf8")
const globals = readFileSync(path.resolve(process.cwd(), "app/globals.css"), "utf8")

describe("workflow-composition-canvas contract", () => {
  it("renders the mock composition graph with React Flow", () => {
    expect(source).toContain("@xyflow/react")
    expect(source).toContain("export function WorkflowCompositionCanvas")
    expect(source).toContain('data-workflow-composition-canvas="mock-only"')
    expect(source).toContain('data-workflow-builder="floating-canvas"')
    expect(source).toContain("buildGraph")
  })

  it("auto-layouts once on entry while retaining the manual layout control", () => {
    expect(source).toContain("hasInitializedLayoutRef")
    expect(source).toContain("if (hasInitializedLayoutRef.current || flowNodes.length === 0) return")
    expect(source).toContain("hasInitializedLayoutRef.current = true")
    expect(source).toContain("autoLayoutStages()")
    expect(source).toContain('aria-label={t("canvas.autoLayout")}')
  })

  it("exposes the builder workbench and YAML dialog", () => {
    expect(source).toContain("EngineLibraryPanel")
    expect(source).toContain("WorkflowYamlDialog")
    expect(source).toContain('data-engine-library="workflow-builder"')
    expect(source).toContain('data-yaml-inspector="workflow-builder"')
    expect(source).toContain('data-workflow-builder-canvas="serial-parallel"')
  })

  it("opens YAML configuration in a compact shared dialog instead of a fixed right pane", () => {
    expect(source).toContain('from "@/components/ui/dialog"')
    expect(source).toContain("const [yamlDialogOpen, setYamlDialogOpen]")
    expect(source).toContain("<DialogContent")
    expect(source).toContain('className="gap-0 overflow-hidden p-0 sm:max-w-3xl"')
    expect(source).toContain("setYamlDialogOpen(true)")
    expect(source).toContain("CopyButton")
    expect(source).not.toContain('className="flex w-80 shrink-0 flex-col overflow-hidden bg-card"')
    expect(source).not.toContain("WorkflowYamlPanel")
  })

  it("keeps advanced YAML separate from step settings tabs", () => {
    expect(source).not.toContain("TabsTrigger")
    expect(source).not.toContain('value="settings"')
    expect(source).not.toContain('t("canvas.stepSettings")')
    expect(source).not.toContain("onUpdateStep")
    expect(source).toContain('t("canvas.yamlConfig")')
    expect(source).toContain("yamlPreview")
  })

  it("keeps the prototype disconnected from backend workflow data access", () => {
    expect(source).not.toContain("useWorkflows")
    expect(source).not.toContain("useWorkflowProfiles")
    expect(source).not.toContain("@/services/")
    expect(source).not.toContain("@/hooks/")
    expect(source).not.toContain("fetch(")
  })

  it("keeps the back affordance separate from floating canvas actions", () => {
    const toolbarStart = source.indexOf("function WorkflowCanvasHeader")
    const toolbarEnd = source.indexOf("function WorkflowCanvasActions")
    const toolbarBlock = source.slice(toolbarStart, toolbarEnd)

    expect(source).toContain("function WorkflowCanvasHeader")
    expect(source).toContain("function WorkflowCanvasActions")
    expect(source).toContain("styles.canvasHeader")
    expect(source).toContain("styles.canvasActions")
    expect(source).toContain("setYamlDialogOpen(true)")
    expect(toolbarBlock).toContain("<ArrowLeft")
    expect(toolbarBlock).not.toContain('t("canvas.yamlConfig")')
    expect(toolbarBlock).not.toContain("saveLabel")
    expect(source).not.toContain("IconRefresh")
    expect(source).not.toContain("resetDraft")
    expect(source).not.toContain("addParallelStep")
    expect(source).toContain("nodesConnectable={false}")
    expect(source).not.toContain("onConnect")
    expect(source).not.toContain("addEdge")
  })

  it("keeps long workflow titles from overlapping the upper-right actions", () => {
    expect(source).toContain("styles.canvasHeaderTitle")
    expect(source).toContain("styles.canvasActionLabel")
    expect(source).toContain('aria-label={t("canvas.yamlConfig")}')
    expect(source).toContain("aria-label={saveLabel}")
    expect(styles).toContain("right: 12rem;")
    expect(styles).toContain("right: 7rem;")
    expect(styles).toContain("text-overflow: ellipsis;")
    expect(styles).toContain("white-space: nowrap;")
    expect(styles).toContain(".canvasActionLabel")
    expect(styles).toContain("display: none;")
  })

  it("collects new workflow metadata in a shared dialog instead of the canvas header", () => {
    const headerStart = source.indexOf("function WorkflowCanvasHeader")
    const headerEnd = source.indexOf("function WorkflowSaveMetadataDialog")
    const headerBlock = source.slice(headerStart, headerEnd)

    expect(source).toContain("function WorkflowSaveMetadataDialog")
    expect(source).toContain("const [saveMetadataDialogOpen, setSaveMetadataDialogOpen]")
    expect(source).toContain("setSaveMetadataDialogOpen(true)")
    expect(source).toContain('t("canvas.saveMetadataTitle")')
    expect(source).toContain('disabled={!displayName.trim() || saveDisabled}')
    expect(headerBlock).not.toContain("<Input")
    expect(headerBlock).toContain('t("canvas.editorTitle")')
  })

  it("adds a serial stage from the final node connection handle", () => {
    expect(source).toContain("isLastStage: stageIndex === stages.length - 1")
    expect(source).toContain("onAddSerialStageAfter")
    expect(source).toContain('data-add-serial-handle="true"')
    expect(source).toContain("data.onAddSerialStageAfter(data.stage.id)")
    expect(source).toContain("addSerialStageAfter")
    expect(source).toContain("<Handle type=\"source\" position={Position.Right} className=\"opacity-0\" />")
  })

  it("keeps per-stage nodes free of inline add-parallel buttons", () => {
    const nodeStart = source.indexOf("function WorkflowStageNode")
    const nodeEnd = source.indexOf("function buildGraph")
    const nodeBlock = source.slice(nodeStart, nodeEnd)

    expect(nodeBlock).not.toContain('size="action-card"')
    expect(nodeBlock).not.toContain("onAddParallelStep")
    expect(nodeBlock).not.toContain('t("canvas.addParallelStep")')
  })

  it("allows engine drag/drop only as local step insertion", () => {
    expect(source).toContain("application/x-lunafox-engine")
    expect(source).toContain("onDropEngine")
    expect(source).toContain("addStepToStage")
    expect(source).not.toContain("fetch(")
  })

  it("renders directed serial edges with a next label", () => {
    const graphStart = source.indexOf("function buildGraph")
    const graphEnd = source.indexOf("function createWorkflowYaml")
    const graphBlock = source.slice(graphStart, graphEnd)

    expect(source).toContain("MarkerType")
    expect(source).toContain('type: "bezier"')
    expect(source).toContain("markerEnd: { type: MarkerType.ArrowClosed }")
    expect(source).toContain("labelShowBg: false")
    expect(source).not.toContain("workflowEdgeAnimatedArrow")
    expect(source).not.toContain("animateMotion")
    expect(graphBlock).toContain('label: "next"')
    expect(graphBlock).toContain("animated: true")
    expect(source).toContain("WORKFLOW_STAGE_HORIZONTAL_SPACING = 400")
    expect(source).toContain("WORKFLOW_STAGE_VERTICAL_STAGGER = 56")
    expect(globals).toContain("animation: workflow-edge-flow 1.6s linear infinite;")
    expect(globals).toContain("[data-workflow-composition-canvas] .react-flow__edge.animated .react-flow__edge-path")
    expect(styles).not.toContain("animation:")
    expect(styles).toContain(".canvasShell :global(.react-flow__edge-text)")
  })

  it("starts and extends the local draft with empty stages", () => {
    const initialDraftStart = source.indexOf("function createInitialDraft")
    const initialDraftEnd = source.indexOf("function WorkflowStageNode")
    const initialDraftBlock = source.slice(initialDraftStart, initialDraftEnd)
    const addSerialStageStart = source.indexOf("const addSerialStageAfter")
    const addSerialStageEnd = source.indexOf("const deleteStep")
    const addSerialStageBlock = source.slice(addSerialStageStart, addSerialStageEnd)

    expect(source).toContain("scanWorkflowId: default")
    expect(source).toContain("stageId: ${stage.id}")
    expect(source).toContain("displayName: ${stage.label}")
    expect(source).toContain("engineId: step.engineId")
    expect(initialDraftBlock).toContain('id: "stage-1"')
    expect(initialDraftBlock).toContain('id: "stage-2"')
    expect(initialDraftBlock).toContain('t("canvas.serialStageLabel"')
    expect(initialDraftBlock).toContain("steps: []")
    expect(initialDraftBlock).not.toContain("createStep")
    expect(addSerialStageBlock).toContain("steps: []")
    expect(addSerialStageBlock).not.toContain("const step = createStep")
    expect(source).not.toContain("http-fingerprint")
    expect(source).not.toContain("directory-scan")
    expect(source).not.toContain("path-bruteforce")
    expect(source).not.toContain("vulnerability-verify")
  })

  it("uses catalog identities without a handwritten engine registry or fallback", () => {
    expect(source).toContain('import type { WorkflowEngineLibraryItem } from "@/lib/engine-catalog"')
    expect(source).toContain("engines: WorkflowEngineLibraryItem[]")
    expect(source).toContain("new Map(engines.map((engine) => [engine.engineId, engine]))")
    expect(source).toContain("createStep(engineId, enginesById")
    expect(source).toContain("engine?.displayName ?? step.engineId")
    expect(source).toContain('data-engine-unavailable={engineCatalogStatus === "ready" && !engine ? "true" : undefined}')
    expect(source).not.toContain("const ENGINE_LIBRARY")
    expect(source).not.toContain("ENGINE_LIBRARY[0]")
    expect(source).not.toContain("labelKey")
    expect(source).not.toContain("engineRef")
  })

  it("allows the last step in a stage to be removed", () => {
    const deleteStepStart = source.indexOf("const deleteStep")
    const deleteStepEnd = source.indexOf("const graph")
    const deleteStepBlock = source.slice(deleteStepStart, deleteStepEnd)

    expect(deleteStepBlock).toContain("steps: stage.steps.filter")
    expect(deleteStepBlock).not.toContain("stage.steps.length <= 1")
  })

  it("opens a stage inspector with destructive controls while retaining one stage", () => {
    expect(source).toContain('from "@/components/ui/alert-dialog"')
    expect(source).toContain("function WorkflowStageInspector")
    expect(source).toContain('data-stage-inspector="workflow-builder"')
    expect(source).toContain("<WorkflowDeleteStageDialog")
    expect(source).toContain('t("canvas.deleteStage")')
    expect(source).toContain('variant="destructive"')
    expect(source).toContain('variant="surface" layout="fullWidth"')
    expect(source).toContain('variant="destructive" layout="fullWidth"')
    expect(source).not.toContain('variant="surface" className="w-full"')
    expect(source).not.toContain('variant="destructive" className="w-full"')
    expect(source).toContain("canDeleteStage={draftStages.length > 1}")
    expect(source).toContain("const deleteStage")
    expect(source).toContain("const [stageInspectorOpen, setStageInspectorOpen]")
    expect(source).toContain("selectStageForInspection")
    expect(source).not.toContain("DropdownMenuTrigger")
  })

  it("renders an empty-stage placeholder without creating a draft step", () => {
    expect(source).toContain('data-empty-stage-placeholder')
    expect(source).toContain('t("canvas.emptyStageDropHint")')
    expect(source).toContain("data.stage.steps.length === 0")
    expect(source).toContain("flex h-12 items-center justify-center gap-2 border border-dashed px-3 py-2")
    expect(source).toContain('lines.push("    steps: []")')
  })

  it("derives the displayed stage execution mode from its engine count", () => {
    const nodeStart = source.indexOf("function WorkflowStageNode")
    const nodeEnd = source.indexOf("function buildGraph")
    const nodeBlock = source.slice(nodeStart, nodeEnd)

    expect(nodeBlock).toContain('data-stage-execution-mode={data.stage.steps.length === 1 ? "serial" : "parallel"}')
    expect(nodeBlock).toContain('t(data.stage.steps.length === 1 ? "canvas.serialExecution" : "canvas.parallelExecution")')
    expect(nodeBlock).toContain("data.stage.steps.length > 0")
  })

  it("keeps stage identities automatic and stage names editable", () => {
    expect(source).toContain('id: "stage-1"')
    expect(source).toContain('id: "stage-2"')
    expect(source).toContain("const nextStageSequenceRef = React.useRef(3)")
    expect(source).toContain("const updateStageLabel")
    expect(source).toContain("const startStageRename")
    expect(source).toContain("const confirmStageRename")
    expect(source).toContain("<WorkflowRenameStageDialog")
    expect(source).toContain("<WorkflowStageInspector")
    expect(source).toContain('t("canvas.renameStage")')
    expect(source).toContain('t("canvas.stageDisplayName")')
    expect(source).toContain("stage.id === stageId ? { ...stage, label }")
  })

  it("keeps engine library rows content-sized instead of full-height action cards", () => {
    const libraryStart = source.indexOf("function EngineLibraryPanel")
    const libraryEnd = source.indexOf("function WorkflowYamlDialog")
    const libraryBlock = source.slice(libraryStart, libraryEnd)

    expect(libraryBlock).not.toContain('size="action-card"')
    expect(libraryBlock).toContain('size="sm"')
    expect(libraryBlock).toContain("h-auto w-full")
    expect(libraryBlock).toContain("line-clamp-1")
    expect(libraryBlock).not.toContain("estimateKey")
  })

  it("keeps the engine library free of category filter tags", () => {
    const libraryStart = source.indexOf("function EngineLibraryPanel")
    const libraryEnd = source.indexOf("function WorkflowYamlDialog")
    const libraryBlock = source.slice(libraryStart, libraryEnd)

    expect(source).not.toContain("CATEGORY_FILTERS")
    expect(source).not.toContain("setCategory")
    expect(libraryBlock).not.toContain("onCategoryChange")
    expect(libraryBlock).not.toContain("matchesCategory")
    expect(libraryBlock).not.toContain('variant={category === item ? "default" : "surface"}')
  })

  it("keeps the engine library free of local search controls", () => {
    const libraryStart = source.indexOf("function EngineLibraryPanel")
    const libraryEnd = source.indexOf("function WorkflowYamlDialog")
    const libraryBlock = source.slice(libraryStart, libraryEnd)

    expect(source).not.toContain("engineSearch")
    expect(source).not.toContain("filteredEngines")
    expect(source).not.toContain("IconSearch")
    expect(libraryBlock).not.toContain("<Input")
    expect(libraryBlock).not.toContain("onEngineSearchChange")
    expect(source).toContain("engines={engines}")
  })

  it("keeps the engine library as a floating tool panel with internal scrolling", () => {
    const libraryStart = source.indexOf("function EngineLibraryPanel")
    const libraryEnd = source.indexOf("function WorkflowYamlDialog")
    const libraryBlock = source.slice(libraryStart, libraryEnd)

    expect(libraryBlock).toContain("flex min-h-0 flex-col overflow-hidden")
    expect(libraryBlock).toContain("styles.floatingEnginePanel")
    expect(libraryBlock).toContain("min-h-0 flex-1 space-y-2 overflow-y-auto")
    expect(styles).toContain(".floatingEnginePanel")
    expect(styles).toContain("position: absolute;")
    expect(styles).toContain("height: min(50rem, calc(100dvh - 6.5rem));")
    expect(styles).toContain("top: 50%;")
    expect(styles).toContain("transform: translateY(-50%);")
  })

  it("keeps the engine library permanently visible", () => {
    const libraryStart = source.indexOf("function EngineLibraryPanel")
    const libraryEnd = source.indexOf("function WorkflowYamlDialog")
    const libraryBlock = source.slice(libraryStart, libraryEnd)

    expect(libraryBlock).toContain('data-engine-library="workflow-builder"')
    expect(source).not.toContain("engineLibraryCollapsed")
    expect(source).not.toContain("collapseEngineLibrary")
    expect(source).not.toContain("expandEngineLibrary")
    expect(styles).not.toContain(".engineLibraryToggle")
  })

  it("keeps the whole builder as a full-height floating canvas", () => {
    expect(source).toContain('className={cn("relative flex h-full min-h-0 flex-1 overflow-hidden", styles.workbench, className)}')
    expect(source).toContain('className={cn("absolute inset-0", styles.canvasShell)}')
    expect(source).toContain("<WorkflowCanvasHeader")
    expect(source).not.toContain("styles.floatingFooter")
    expect(source).not.toContain('h-[clamp(18rem,42dvh,30rem)]')
    expect(source).not.toContain("styles.canvasPanel")
  })

  it("keeps the canvas navigation map compact", () => {
    expect(source).toContain("<MiniMap")
    expect(source).toContain("style={{ height: 120 }}")
  })

  it("allows panning the React Flow canvas", () => {
    expect(source).toContain("panOnDrag")
    expect(styles).toContain(".canvasShell :global(.react-flow__pane)")
    expect(styles).toContain("cursor: grab;")
    expect(styles).toContain(".canvasShell :global(.react-flow__pane:active)")
    expect(styles).toContain("cursor: grabbing;")
    expect(styles).toContain("pointer-events: all;")
    expect(styles).not.toContain(".canvasShell :global(.react-flow__pane) {\n  z-index: 0;\n  pointer-events: none;")
  })

  it("uses React Flow controlled nodes while retaining positions during stage updates", () => {
    expect(source).toContain("useNodesState")
    expect(source).toContain("const [flowNodes, setFlowNodes, onNodesChange]")
    expect(source).toContain("currentNodesById")
    expect(source).toContain("position: currentNode.position")
    expect(source).toContain("nodesDraggable")
    expect(source).toContain("onNodesChange={onNodesChange}")
    expect(source).not.toContain("onNodeDragStop")
  })

  it("keeps viewport controls lower-right and workflow actions upper-right", () => {
    expect(source).toContain("zoomOnScroll")
    expect(source).toContain("WORKFLOW_CANVAS_MIN_ZOOM = 0.4")
    expect(source).toContain("WORKFLOW_CANVAS_MAX_ZOOM = 1.6")
    expect(source).toContain('position="top-right"')
    expect(source).toContain('position="bottom-right"')
    expect(source).toContain("useReactFlow")
    expect(source).toContain("useViewport")
    expect(source).toContain("data-workflow-canvas-zoom-controls")
    expect(source).toContain("zoomTo")
    expect(styles).toContain(".zoomControls")
    expect(styles).toContain(".canvasActions")
    expect(styles).toContain(".canvasPrimaryActions")
    expect(source).toContain("const autoLayoutStages")
    expect(source).toContain("<FocusCentered")
    expect(source).toContain("getViewportForBounds")
    expect(source).not.toContain("await fitView({ padding: 0.16, maxZoom: 1, duration: 200 })")
    expect(styles).not.toContain("transform: translateX(-50%);")
  })

  it("centers automatic layouts in the desktop canvas area not obscured by the engine library", () => {
    const actionsStart = source.indexOf("function WorkflowCanvasActions")
    const actionsEnd = source.indexOf("function EngineLibraryPanel")
    const actionsBlock = source.slice(actionsStart, actionsEnd)

    expect(source).toContain("WORKFLOW_ENGINE_LIBRARY_OCCLUSION_PX = 304")
    expect(actionsBlock).toContain("getNodes")
    expect(actionsBlock).toContain("getNodesBounds")
    expect(actionsBlock).toContain("setViewport")
    expect(actionsBlock).toContain('window.matchMedia("(min-width: 768px)").matches')
    expect(actionsBlock).toContain("canvasWidth - libraryOcclusion")
    expect(actionsBlock).toContain("viewport.x += libraryOcclusion")
    expect(actionsBlock).toContain("setViewport(viewport, { duration: 200 })")
    expect(actionsBlock).not.toContain("fitView(")
  })

  it("keeps React Flow background dots visible in dark themes", () => {
    expect(source).toContain("<Background gap={24} size={1.5} />")
    expect(styles).toContain("--xy-background-pattern-color: color-mix(in oklab, var(--muted-foreground) 58%, transparent);")
    expect(styles).toContain("--xy-background-pattern-color: color-mix(in oklab, white 48%, transparent);")
    expect(styles).toContain(".canvasShell :global(.react-flow__background-pattern.dots)")
    expect(styles).toContain("fill: var(--xy-background-pattern-color);")
    expect(styles).not.toContain("--xy-background-pattern-color: var(--border);")
  })

  it("centralizes workbench surface colors in canvas-scoped variables", () => {
    expect(source).toContain("styles.stageNode")
    expect(source).toContain("styles.stepRow")
    expect(source).toContain("styles.engineButton")
    expect(source).toContain("styles.floatingEnginePanel")
    expect(styles).toContain("--workflow-canvas-background")
    expect(styles).toContain(".workbench")
    expect(styles).toContain("--workflow-stage-background")
    expect(styles).toContain("--workflow-step-background")
    expect(styles).toContain("--workflow-engine-background")
    expect(styles).toContain("--workflow-stage-selected-border")
    expect(styles).toContain(".floatingEnginePanel")
    expect(styles).toContain("border-radius: var(--radius-overlay);")
    expect(styles).not.toContain("bg-primary/5")
  })

  it("keeps the add-stage handle small and opaque", () => {
    expect(source).toContain("data-add-serial-handle=\"true\"")
    expect(source).toContain("size-5")
    expect(source).toContain("size-7")
    expect(source).toContain("bg-[var(--workflow-add-stage-pad-background)]")
    expect(source).toContain("bg-[var(--workflow-add-stage-background)]")
    expect(source).toContain("text-[var(--workflow-add-stage-foreground)]")
    expect(styles).toContain("--workflow-add-stage-background: var(--card);")
    expect(styles).toContain("--workflow-add-stage-pad-background")
    expect(styles).toContain("--workflow-add-stage-border: var(--border);")
    expect(styles).not.toContain("--workflow-add-stage-background: color-mix(in oklab, var(--card) 88%, var(--foreground) 12%);")
  })

  it("shows a drag affordance in the library without adding engine glyphs to stage steps", () => {
    const libraryStart = source.indexOf("function EngineLibraryPanel")
    const libraryEnd = source.indexOf("function WorkflowYamlDialog")
    const libraryBlock = source.slice(libraryStart, libraryEnd)

    expect(source).not.toContain("const EngineIcon = engine.icon")
    expect(source).not.toContain("<EngineIcon")
    expect(libraryBlock).toContain("<GripVerticalIcon")
    expect(libraryBlock).toContain("size-4 shrink-0 self-center text-muted-foreground")
    expect(source).toContain("<IconTrash")
    expect(libraryBlock).not.toContain("<ChevronLeft")
    expect(libraryBlock).not.toContain("<ChevronRight")
  })

  it("derives stage step presentation from catalog metadata and exact identity", () => {
    const stageNodeStart = source.indexOf("function WorkflowStageNode")
    const stageNodeEnd = source.indexOf("function buildGraph")
    const stageNodeBlock = source.slice(stageNodeStart, stageNodeEnd)

    expect(stageNodeBlock).toContain("engine?.displayName ?? step.engineId")
    expect(stageNodeBlock).toContain("engine?.description")
    expect(stageNodeBlock).not.toContain("step.engineLabel")
  })
})
