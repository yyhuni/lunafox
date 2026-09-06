import * as React from "react"
import { fireEvent, screen, waitFor } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { WorkflowCompositionCanvas } from "@/components/scan/workflow/workflow-composition-canvas"
import { renderWithProviders } from "@/test/utils/render-with-providers"

const { getViewportForBoundsMock, setViewportMock } = vi.hoisted(() => ({
  getViewportForBoundsMock: vi.fn(() => ({ x: 100, y: 200, zoom: 1 })),
  setViewportMock: vi.fn().mockResolvedValue(true),
}))

vi.mock("@xyflow/react", () => ({
  Background: () => null,
  getViewportForBounds: getViewportForBoundsMock,
  Handle: () => null,
  MarkerType: { ArrowClosed: "arrowclosed" },
  MiniMap: ({
    ariaLabel,
    pannable,
    position,
    zoomable,
  }: {
    ariaLabel: string
    pannable: boolean
    position: string
    zoomable: boolean
  }) => (
    <div
      role="img"
      aria-label={ariaLabel}
      data-minimap-pannable={pannable}
      data-minimap-position={position}
      data-minimap-zoomable={zoomable}
    />
  ),
  Panel: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Position: { Left: "left", Right: "right" },
  ReactFlow: ({
    children,
    nodeTypes,
    nodes,
    onNodesChange,
  }: {
    children: React.ReactNode
    nodeTypes: Record<string, React.ComponentType<{ data: unknown }>>
    nodes: Array<{ id: string; type: string; data: unknown; position: { x: number; y: number } }>
    onNodesChange: (changes: Array<{ id: string; type: "position"; position: { x: number; y: number } }>) => void
  }) => (
    <div>
      <button
        type="button"
        aria-label="move-stage-1"
        onClick={() => onNodesChange([{
          id: "stage-1",
          type: "position",
          position: { x: 480, y: 160 },
        }])}
      />
      {nodes.map((node) => {
        const NodeComponent = nodeTypes[node.type]
        return (
          <div key={node.id} data-flow-node-id={node.id} data-flow-node-position={`${node.position.x},${node.position.y}`}>
            <NodeComponent data={node.data} />
          </div>
        )
      })}
      {children}
    </div>
  ),
  useNodesState: (initialNodes: Array<{ id: string; position: { x: number; y: number } }>) => {
    const [nodes, setNodes] = React.useState(initialNodes)
    const onNodesChange = React.useCallback((changes: Array<{ id: string; type: "position"; position: { x: number; y: number } }>) => {
      setNodes((currentNodes) => currentNodes.map((node) => {
        const positionChange = changes.find((change) => change.id === node.id && change.type === "position")
        return positionChange ? { ...node, position: positionChange.position } : node
      }))
    }, [])

    return [nodes, setNodes, onNodesChange]
  },
  useReactFlow: () => ({
    zoomTo: vi.fn(),
    getNodes: vi.fn(() => []),
    getNodesBounds: vi.fn(() => ({ x: 0, y: 0, width: 400, height: 120 })),
    setViewport: setViewportMock,
  }),
  useStore: (selector: (state: { width: number; height: number }) => unknown) => selector({ width: 1200, height: 800 }),
  useViewport: () => ({ zoom: 1 }),
}))

const canvasCatalogProps = {
  engines: [
    {
      engineId: "engine.example.custom",
      displayName: "Custom Engine",
      description: "Custom engine description",
    },
    {
      engineId: "engine.example.second",
      displayName: "Second Engine",
      description: "Second engine description",
    },
  ],
  engineCatalogStatus: "ready" as const,
}

describe("WorkflowCompositionCanvas", () => {
  it("auto-layouts and focuses stages when the canvas is opened", async () => {
    vi.spyOn(window, "matchMedia").mockImplementation((query) => ({
      matches: query === "(min-width: 768px)",
    }) as MediaQueryList)
    getViewportForBoundsMock.mockClear()
    setViewportMock.mockClear()

    const { container } = renderWithProviders(<WorkflowCompositionCanvas {...canvasCatalogProps} />)

    await waitFor(() => expect(setViewportMock).toHaveBeenCalledTimes(1))

    expect(container.querySelector('[data-flow-node-id="stage-1"]')).toHaveAttribute("data-flow-node-position", "0,0")
    expect(container.querySelector('[data-flow-node-id="stage-2"]')).toHaveAttribute("data-flow-node-position", "400,56")
    expect(getViewportForBoundsMock).toHaveBeenCalledWith(
      { x: 0, y: 0, width: 400, height: 120 },
      896,
      800,
      0.4,
      1,
      0.16,
    )
  })

  it("auto-layouts stages with room for curved serial connectors", async () => {
    const { container } = renderWithProviders(<WorkflowCompositionCanvas {...canvasCatalogProps} />)
    vi.spyOn(window, "matchMedia").mockImplementation((query) => ({
      matches: query === "(min-width: 768px)",
    }) as MediaQueryList)
    getViewportForBoundsMock.mockClear()
    setViewportMock.mockClear()

    fireEvent.click(screen.getByRole("button", { name: "canvas.autoLayout" }))

    await waitFor(() => expect(setViewportMock).toHaveBeenCalledTimes(1))

    expect(container.querySelector('[data-flow-node-id="stage-1"]')).toHaveAttribute("data-flow-node-position", "0,0")
    expect(container.querySelector('[data-flow-node-id="stage-2"]')).toHaveAttribute("data-flow-node-position", "400,56")
    expect(getViewportForBoundsMock).toHaveBeenCalledWith(
      { x: 0, y: 0, width: 400, height: 120 },
      896,
      800,
      0.4,
      1,
      0.16,
    )
    expect(setViewportMock).toHaveBeenCalledWith({ x: 404, y: 200, zoom: 1 }, { duration: 200 })
  })

  it("retains a dragged stage position after stage data changes", async () => {
    const { container } = renderWithProviders(<WorkflowCompositionCanvas {...canvasCatalogProps} />)

    fireEvent.click(screen.getByRole("button", { name: "move-stage-1" }))
    const firstNode = container.querySelector<HTMLElement>('[data-flow-node-id="stage-1"]')!
    expect(firstNode).toHaveAttribute("data-flow-node-position", "480,160")

    fireEvent.click(screen.getByRole("button", {
      name: "Custom EngineCustom engine description",
    }))
    await waitFor(() => expect(firstNode).toHaveAttribute("data-flow-node-position", "480,160"))
  })

  it("keeps the engine library visible", () => {
    renderWithProviders(<WorkflowCompositionCanvas {...canvasCatalogProps} />)

    expect(screen.getByText("canvas.engineLibrary")).toBeInTheDocument()
    expect(document.querySelector("[data-engine-library]")).toBeInTheDocument()
  })

  it("blocks engine insertion and saving while the catalog is loading or failed", () => {
    const onSave = vi.fn()
    const { rerender } = renderWithProviders(
      <WorkflowCompositionCanvas
        engines={[]}
        engineCatalogStatus="loading"
        displayName="Discovery"
        onSave={onSave}
      />
    )

    expect(screen.getByText("canvas.engineLibraryLoading")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "management.save" })).toBeDisabled()

    rerender(
      <WorkflowCompositionCanvas
        engines={[]}
        engineCatalogStatus="error"
        displayName="Discovery"
        onSave={onSave}
      />
    )
    expect(screen.getByRole("alert")).toHaveTextContent("canvas.engineLibraryError")
    expect(screen.getByRole("button", { name: "management.save" })).toBeDisabled()
    expect(onSave).not.toHaveBeenCalled()
  })

  it("renders an explicit empty state for a successfully loaded empty catalog", () => {
    renderWithProviders(<WorkflowCompositionCanvas engines={[]} engineCatalogStatus="ready" />)

    expect(screen.getByText("canvas.engineLibraryEmpty")).toBeInTheDocument()
    expect(document.querySelector('[data-engine-library] button[draggable="true"]')).toBeNull()
  })

  it("preserves an unavailable engine identity until the operator removes the step", async () => {
    const onSave = vi.fn()
    const stages = [{
      stageId: "stage-1",
      steps: [{ stageId: "stage-1", stepId: "missing-step", engineId: "engine.example.missing", profileDefaultEnabled: true }],
    }]
    const { container } = renderWithProviders(
      <WorkflowCompositionCanvas
        {...canvasCatalogProps}
        stages={stages}
        displayName="Discovery"
        onSave={onSave}
      />
    )

    const unavailableStep = container.querySelector<HTMLElement>('[data-engine-unavailable="true"]')!
    expect(unavailableStep).toHaveTextContent("engine.example.missing")
    expect(unavailableStep).toHaveTextContent("canvas.engineUnavailable")
    expect(screen.getByRole("button", { name: "management.save" })).toBeDisabled()

    fireEvent.click(container.querySelector<HTMLElement>('[data-stage-drop-target="stage-1"]')!)
    await waitFor(() => expect(screen.getByRole("button", { name: "canvas.removeEngine" })).toBeInTheDocument())
    fireEvent.click(screen.getByRole("button", { name: "canvas.removeEngine" }))

    expect(container.querySelector('[data-engine-unavailable="true"]')).toBeNull()
    fireEvent.click(screen.getByRole("button", { name: "close" }))
    await waitFor(() => expect(screen.getByRole("button", { name: "management.save" })).toBeVisible())
    expect(screen.getByRole("button", { name: "management.save" })).toBeEnabled()
  })

  it("renders an interactive canvas overview in the lower-left corner", () => {
    renderWithProviders(<WorkflowCompositionCanvas {...canvasCatalogProps} />)

    const miniMap = screen.getByRole("img", { name: "canvas.miniMap" })
    expect(miniMap).toHaveAttribute("data-minimap-position", "bottom-left")
    expect(miniMap).toHaveAttribute("data-minimap-pannable", "true")
    expect(miniMap).toHaveAttribute("data-minimap-zoomable", "true")
  })

  it("serializes the editable draft as pure orchestration when saved", () => {
    const onSave = vi.fn()
    renderWithProviders(<WorkflowCompositionCanvas {...canvasCatalogProps} displayName="Discovery" onSave={onSave} />)

    fireEvent.click(screen.getByRole("button", {
      name: "Custom EngineCustom engine description",
    }))
    fireEvent.click(screen.getByRole("button", { name: "management.save" }))

    expect(onSave).toHaveBeenCalledWith(expect.arrayContaining([
      expect.objectContaining({
        stageId: "stage-1",
        steps: [expect.objectContaining({
          stageId: "stage-1",
          stepId: "step-local-100",
          engineId: "engine.example.custom",
        })],
      }),
    ]))
  })

  it("collects metadata in a dialog before the first save", () => {
    const onSave = vi.fn()

    function WorkflowCanvasWithMetadata() {
      const [displayName, setDisplayName] = React.useState("")
      const [description, setDescription] = React.useState("")

      return (
        <WorkflowCompositionCanvas {...canvasCatalogProps}
          displayName={displayName}
          description={description}
          onDisplayNameChange={setDisplayName}
          onDescriptionChange={setDescription}
          onSave={onSave}
        />
      )
    }

    renderWithProviders(<WorkflowCanvasWithMetadata />)

    expect(screen.queryByRole("textbox", { name: "canvas.workflowName" })).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole("button", { name: "management.save" }))

    expect(screen.getByRole("heading", { name: "canvas.saveMetadataTitle" })).toBeInTheDocument()
    fireEvent.change(screen.getByRole("textbox", { name: "canvas.workflowName" }), {
      target: { value: "Discovery" },
    })
    fireEvent.change(screen.getByRole("textbox", { name: "canvas.workflowDescription" }), {
      target: { value: "Discover assets" },
    })
    fireEvent.click(screen.getByRole("button", { name: "canvas.confirmSaveMetadata" }))

    expect(onSave).toHaveBeenCalledWith(expect.any(Array))
    expect(screen.queryByRole("heading", { name: "canvas.saveMetadataTitle" })).not.toBeInTheDocument()
  })

  it("omits every mutation control in read-only mode", () => {
    renderWithProviders(<WorkflowCompositionCanvas {...canvasCatalogProps} readOnly displayName="Built-in" onSave={vi.fn()} />)

    expect(screen.queryByRole("button", { name: "management.save" })).not.toBeInTheDocument()
    expect(document.querySelector("[data-engine-library]")).not.toBeInTheDocument()
    expect(screen.queryByLabelText("canvas.workflowName")).not.toBeInTheDocument()
  })

  it("starts with empty stages and lets the user add then remove the only engine", async () => {
    renderWithProviders(<WorkflowCompositionCanvas {...canvasCatalogProps} />)

    expect(screen.getAllByText("Custom Engine")).toHaveLength(1)
    expect(screen.getAllByText("canvas.emptyStageDropHint")).toHaveLength(2)

    fireEvent.click(screen.getByRole("button", {
      name: "Custom EngineCustom engine description",
    }))
    const firstStage = document.querySelector<HTMLElement>('[data-stage-drop-target="stage-1"]')!
    expect(firstStage.querySelector('[data-stage-execution-mode="serial"]')).toHaveTextContent("canvas.serialExecution")
    expect(firstStage).toHaveTextContent("Custom Engine")
    expect(firstStage).toHaveTextContent("Custom engine description")
    expect(screen.getAllByText("canvas.emptyStageDropHint")).toHaveLength(1)

    fireEvent.click(firstStage)
    await waitFor(() => expect(screen.getByRole("button", { name: "canvas.removeEngine" })).toBeInTheDocument())
    fireEvent.click(screen.getByRole("button", { name: "canvas.removeEngine" }))
    expect(firstStage).not.toHaveTextContent("Custom Engine")
    expect(firstStage.querySelector("[data-stage-execution-mode]")).not.toBeInTheDocument()
    expect(firstStage).toHaveTextContent("canvas.emptyStageDropHint")
  })

  it("changes a stage from serial to parallel when a second engine is added", () => {
    renderWithProviders(<WorkflowCompositionCanvas {...canvasCatalogProps} />)

    const firstStage = document.querySelector<HTMLElement>('[data-stage-drop-target="stage-1"]')!
    expect(firstStage.querySelector("[data-stage-execution-mode]")).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole("button", {
      name: "Custom EngineCustom engine description",
    }))
    expect(firstStage.querySelector('[data-stage-execution-mode="serial"]')).toHaveTextContent("canvas.serialExecution")

    fireEvent.click(screen.getByRole("button", {
      name: "Second EngineSecond engine description",
    }))
    expect(firstStage.querySelector('[data-stage-execution-mode="parallel"]')).toHaveTextContent("canvas.parallelExecution")
  })

  it("creates an empty stage when the user appends a serial stage", () => {
    const { container } = renderWithProviders(<WorkflowCompositionCanvas {...canvasCatalogProps} />)

    fireEvent.click(screen.getByRole("button", { name: "canvas.addSerialStage" }))

    const stage = container.querySelector('[data-stage-drop-target="stage-3"]')
    expect(stage).not.toBeNull()
    expect(stage).toHaveTextContent("canvas.emptyStageDropHint")

    fireEvent.click(screen.getByRole("button", {
      name: "Custom EngineCustom engine description",
    }))
    expect(container.querySelector('[data-stage-drop-target="stage-3"]')).toHaveTextContent("Custom Engine")
  })

  it("keeps generated stage IDs stable while allowing display names to change", async () => {
    const { container } = renderWithProviders(<WorkflowCompositionCanvas {...canvasCatalogProps} />)

    const firstStage = container.querySelector<HTMLElement>('[data-stage-drop-target="stage-1"]')!
    fireEvent.click(firstStage)
    await waitFor(() => expect(screen.getByText("canvas.stageEngines")).toBeInTheDocument())
    fireEvent.click(screen.getByRole("button", { name: "canvas.renameStage" }))
    expect(screen.getByRole("heading", { name: "canvas.renameStageTitle" })).toBeInTheDocument()
    fireEvent.change(screen.getByRole("textbox", { name: "canvas.stageDisplayName" }), {
      target: { value: "资产发现" },
    })
    fireEvent.click(screen.getByRole("button", { name: "canvas.cancelStageRename" }))
    expect(firstStage).not.toHaveTextContent("资产发现")

    fireEvent.click(screen.getByRole("button", { name: "canvas.renameStage" }))
    fireEvent.change(screen.getByRole("textbox", { name: "canvas.stageDisplayName" }), {
      target: { value: "资产发现" },
    })
    fireEvent.click(screen.getByRole("button", { name: "canvas.confirmStageRename" }))
    expect(firstStage).toHaveTextContent("资产发现")
    expect(firstStage).toHaveAttribute("data-stage-drop-target", "stage-1")
    expect(firstStage).not.toHaveTextContent("stage-1")

    fireEvent.click(screen.getByRole("button", { name: "close" }))
    await waitFor(() => expect(screen.getByRole("button", { name: "canvas.addSerialStage" })).toBeInTheDocument())
    fireEvent.click(screen.getByRole("button", { name: "canvas.addSerialStage" }))
    const thirdStage = container.querySelector<HTMLElement>('[data-stage-drop-target="stage-3"]')!
    fireEvent.click(thirdStage)
    await waitFor(() => expect(screen.getByRole("button", { name: "canvas.deleteStage" })).toBeInTheDocument())
    fireEvent.click(screen.getByRole("button", { name: "canvas.deleteStage" }))
    fireEvent.click(screen.getByRole("button", { name: "canvas.confirmStageDelete" }))
    fireEvent.click(screen.getByRole("button", { name: "canvas.addSerialStage" }))

    expect(container.querySelector('[data-stage-drop-target="stage-3"]')).toBeNull()
    expect(container.querySelector('[data-stage-drop-target="stage-4"]')).not.toBeNull()
  }, 15_000)

  it("deletes a stage through the inspector and selects an adjacent stage", async () => {
    const { container } = renderWithProviders(<WorkflowCompositionCanvas {...canvasCatalogProps} />)

    fireEvent.click(screen.getByRole("button", { name: "canvas.addSerialStage" }))

    const stage = container.querySelector<HTMLElement>('[data-stage-drop-target="stage-3"]')!
    fireEvent.click(stage)
    await waitFor(() => expect(screen.getByRole("button", { name: "canvas.deleteStage" })).toBeInTheDocument())
    fireEvent.click(screen.getByRole("button", { name: "canvas.deleteStage" }))
    fireEvent.click(screen.getByRole("button", { name: "canvas.confirmStageDelete" }))

    await waitFor(() => {
      expect(container.querySelector('[data-stage-drop-target="stage-3"]')).toBeNull()
      expect(container.querySelector('[data-stage-selected="true"]')).toHaveAttribute("data-stage-drop-target", "stage-2")
    })

    fireEvent.click(screen.getByRole("button", {
      name: "Custom EngineCustom engine description",
    }))
    expect(container.querySelector('[data-stage-drop-target="stage-2"]')).toHaveTextContent("Custom Engine")
  })
})
