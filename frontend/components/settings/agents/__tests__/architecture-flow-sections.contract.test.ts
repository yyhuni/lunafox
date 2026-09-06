import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/agents/architecture-flow-sections.tsx"), "utf8")
const layoutSource = readFileSync(path.resolve(process.cwd(), "components/settings/agents/architecture-flow-layout.tsx"), "utf8")
const globalStyles = readFileSync(path.resolve(process.cwd(), "app/globals.css"), "utf8")

describe("architecture-flow-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("derives connector paths from node frames instead of hard-coded coordinates", () => {
    expect(source).toContain("function createConnectorPath")
    expect(source).toContain("leftCenter")
    expect(source).toContain("rightCenter")
    expect(layoutSource).toContain("function centerY")
    expect(source).not.toContain("M196 340 H302")
    expect(source).not.toContain("M538 514 H640")
    expect(source).not.toContain("M860 89 H894")
  })

  it("routes result return paths through Agent before Server", () => {
    expect(source).toContain("resultEngineTapPaths")
    expect(source).toContain("resultEngineSpinePaths")
    expect(source).toContain("resultEngineToAgentPaths")
    expect(source).toContain("resultAgentTapPaths")
    expect(source).toContain("resultAgentSpinePath")
    expect(source).toContain("resultAgentToServerPaths")
    expect(source).toContain("resultEngineAnchor")
    expect(source).toContain("resultAgentOutputAnchor")
    expect(source).toContain("resultAgentInputAnchor")
    expect(source).toContain("resultServerAnchor")
    expect(source).toContain("const resultArrowTipRefX = 8")
    expect(source).toContain('id="flow-arrow-result"')
    expect(source).toContain("refX={resultArrowTipRefX}")
    expect(source).not.toContain("resultWorker")
    expect(source).not.toContain("resultTapPaths")
    expect(source).not.toContain("resultReturnBottomPath")
    expect(source).not.toContain("resultBusX")
  })

  it("keeps bidirectional start markers oriented by svg instead of pre-flipping the path", () => {
    expect(source).toContain("orient=\"auto-start-reverse\"")
    expect(source).toContain("markerStart={index === 1 ? \"url(#flow-arrow-agent-start)\" : undefined}")
    expect(source).not.toContain("M8,0 L0,4 L8,8 Z")
  })

  it("animates connector direction without moving the topology", () => {
    expect(source).toContain("architecture-flow-connector-control")
    expect(source).toContain("architecture-flow-connector-engine")
    expect(source).toContain("architecture-flow-connector-result-pulse")
    expect(source).toContain("pathLength={1}")
    expect(globalStyles).toContain("@keyframes architecture-flow-dash")
    expect(globalStyles).toContain("stroke-dashoffset: -12")
    expect(globalStyles).toContain("@keyframes architecture-flow-result-pulse")
    expect(globalStyles).toContain(".architecture-flow-connector-result-pulse")
    expect(globalStyles).toContain("@media (prefers-reduced-motion: reduce)")
  })

  it("keeps group frames large enough to wrap the Engine execution nodes", () => {
    expect(layoutSource).toContain("export const GROUP_X = 252")
    expect(layoutSource).toContain("export const GROUP_WIDTH = 652")
    expect(layoutSource).toContain("{ x: GROUP_X, y: 28, width: GROUP_WIDTH, height: 208 }")
    expect(layoutSource).toContain("{ x: GROUP_X, y: 412, width: GROUP_WIDTH, height: 172 }")
    expect(layoutSource).not.toContain("workerFrames")
  })

  it("marks semantic topology elements so only empty canvas space starts a pan", () => {
    expect(source).toContain('data-architecture-flow-node="server"')
    expect(source).toContain('data-architecture-flow-node="agent"')
    expect(source).toContain('data-architecture-flow-node="engine"')
    expect(source).toContain('data-architecture-flow-node="vps-group"')
  })

  it("does not render simulated runtime status in the static topology", () => {
    expect(source).not.toContain("node.status")
    expect(source).not.toContain("roleDotClassName")
  })

  it("renders the topology in an undecorated pannable viewport", () => {
    expect(layoutSource).toContain("data-architecture-flow-viewport")
    expect(layoutSource).toContain("setPointerCapture")
    expect(layoutSource).toContain("releasePointerCapture")
    expect(layoutSource).toContain("fillAvailableSpace")
    expect(layoutSource).toContain("Math.min(widthScale, heightScale)")
    expect(layoutSource).toContain("contentOffset.x + pan.x")
    expect(layoutSource).toContain('closest("[data-architecture-flow-node]")')
    expect(layoutSource).toContain("overlay?: ReactNode")
    expect(layoutSource).toContain("pointer-events-none absolute inset-x-0 bottom-3")
    expect(layoutSource).not.toContain("rounded-md border bg-gradient-to-br")
  })
})
