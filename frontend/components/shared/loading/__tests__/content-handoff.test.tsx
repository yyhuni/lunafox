import { act, fireEvent, render, screen, waitFor } from "@testing-library/react"
import * as React from "react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { ContentHandoff } from "@/components/shared/loading/content-handoff"

type ResizeObserverCallback = ConstructorParameters<typeof ResizeObserver>[0]
type MutationObserverCallback = ConstructorParameters<typeof MutationObserver>[0]

const resizeObserverInstances: Array<{
  callback: ResizeObserverCallback
  disconnected: boolean
}> = []

const mutationObserverInstances: Array<{
  callback: MutationObserverCallback
  disconnected: boolean
}> = []

const originalResizeObserver = globalThis.ResizeObserver
const originalMutationObserver = globalThis.MutationObserver
const originalRequestAnimationFrame = globalThis.requestAnimationFrame
const originalCancelAnimationFrame = globalThis.cancelAnimationFrame

let nextAnimationFrameId = 0
const animationFrameCallbacks = new Map<number, FrameRequestCallback>()

class ResizeObserverTestDouble {
  callback: ResizeObserverCallback
  disconnected = false

  constructor(callback: ResizeObserverCallback) {
    this.callback = callback
    resizeObserverInstances.push(this)
  }

  observe() {
    this.disconnected = false
  }
  unobserve() {}
  disconnect() {
    this.disconnected = true
  }
}

class MutationObserverTestDouble {
  callback: MutationObserverCallback
  disconnected = false

  constructor(callback: MutationObserverCallback) {
    this.callback = callback
    mutationObserverInstances.push(this)
  }

  observe() {
    this.disconnected = false
  }
  disconnect() {
    this.disconnected = true
  }
  takeRecords() {
    return []
  }
}

beforeEach(() => {
  resizeObserverInstances.length = 0
  mutationObserverInstances.length = 0
  animationFrameCallbacks.clear()
  nextAnimationFrameId = 0

  vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockImplementation(function getBoundingClientRect(this: HTMLElement) {
    let height = 0

    if (this.classList.contains("loading-handoff__skeleton")) {
      height = 200
    } else if (this.classList.contains("loading-handoff__content")) {
      height = 200
    } else if (this.dataset.testid === "resolved-geometry") {
      height = 280
    } else if (this.dataset.testid === "late-resolved-geometry") {
      height = Number(this.dataset.height)
    } else if (this.dataset.testid === "skeleton-geometry") {
      height = 200
    } else if (
      this.parentElement?.classList.contains("loading-handoff__content") &&
      this.textContent?.trim()
    ) {
      height = 200
    }

    return {
      bottom: height,
      height,
      left: 0,
      right: 0,
      top: 0,
      width: 0,
      x: 0,
      y: 0,
      toJSON() {
        return this
      },
    } as DOMRect
  })

  globalThis.ResizeObserver = ResizeObserverTestDouble as typeof ResizeObserver
  globalThis.MutationObserver = MutationObserverTestDouble as typeof MutationObserver
  globalThis.requestAnimationFrame = ((callback: FrameRequestCallback) => {
    nextAnimationFrameId += 1
    animationFrameCallbacks.set(nextAnimationFrameId, callback)
    return nextAnimationFrameId
  }) as typeof requestAnimationFrame
  globalThis.cancelAnimationFrame = ((frameId: number) => {
    animationFrameCallbacks.delete(frameId)
  }) as typeof cancelAnimationFrame
})

afterEach(() => {
  vi.useRealTimers()
  vi.restoreAllMocks()
  globalThis.ResizeObserver = originalResizeObserver
  globalThis.MutationObserver = originalMutationObserver
  globalThis.requestAnimationFrame = originalRequestAnimationFrame
  globalThis.cancelAnimationFrame = originalCancelAnimationFrame
})

function flushAnimationFrame() {
  const callbacks = Array.from(animationFrameCallbacks.values())
  animationFrameCallbacks.clear()
  for (const callback of callbacks) {
    callback(0)
  }
}

function notifyActiveResizeObservers() {
  for (const observer of resizeObserverInstances) {
    if (observer.disconnected) continue
    observer.callback([], observer as unknown as ResizeObserver)
  }
}

function notifyActiveMutationObservers() {
  for (const observer of mutationObserverInstances) {
    if (observer.disconnected) continue
    observer.callback([], observer as unknown as MutationObserver)
  }
}

function ReadyProbe({ onReady }: { onReady: () => void }) {
  React.useEffect(() => {
    onReady()
  }, [onReady])

  return <div>Resolved content</div>
}

function HandoffProbe() {
  const [isReady, setIsReady] = React.useState(false)

  return (
    <ContentHandoff
      owner="content-handoff-ready-probe"
      isLoading={!isReady}
      skeleton={<div>Loading skeleton</div>}
      mountContentWhileLoading
    >
      <ReadyProbe onReady={() => setIsReady(true)} />
    </ContentHandoff>
  )
}

function DelayedIntrinsicContent() {
  const [isMounted, setIsMounted] = React.useState(false)

  React.useEffect(() => {
    setIsMounted(true)
  }, [])

  return isMounted ? <div data-testid="resolved-geometry">Resolved geometry</div> : null
}

function ExternallyReadyDelayedContentProbe() {
  const [isLoading, setIsLoading] = React.useState(true)
  const [shouldMountContent, setShouldMountContent] = React.useState(false)

  return (
    <>
      <button type="button" onClick={() => setIsLoading(false)}>mark ready</button>
      <button type="button" onClick={() => setShouldMountContent(true)}>mount content</button>
      <ContentHandoff
        owner="content-handoff-delayed-child-probe"
        isLoading={isLoading}
        skeleton={<div>Loading skeleton</div>}
      >
        {shouldMountContent ? <div>Resolved content</div> : null}
      </ContentHandoff>
    </>
  )
}

function ExternallyReadyEmptyShellProbe() {
  const [isLoading, setIsLoading] = React.useState(true)
  const [isResolved, setIsResolved] = React.useState(false)

  return (
    <>
      <button type="button" onClick={() => setIsLoading(false)}>mark ready</button>
      <button type="button" onClick={() => setIsResolved(true)}>mount resolved geometry</button>
      <ContentHandoff
        owner="content-handoff-empty-shell-probe"
        isLoading={isLoading}
        skeleton={<div>Loading skeleton</div>}
      >
        <div data-testid="empty-geometry" />
        {isResolved ? <div data-testid="resolved-geometry">Resolved geometry</div> : null}
      </ContentHandoff>
    </>
  )
}

function PreparedHandoffProbe({ transitionMode = "crossfade" }: { transitionMode?: "crossfade" | "replace" }) {
  const [isLoading, setIsLoading] = React.useState(true)

  return (
    <>
      <button type="button" onClick={() => setIsLoading(false)}>mark ready</button>
      <ContentHandoff
        owner="content-handoff-preparation-probe"
        isLoading={isLoading}
        skeleton={<div data-testid="skeleton-geometry">Loading skeleton</div>}
        prepareContentBeforeHandoff
        transitionMode={transitionMode}
      >
        <div data-testid="resolved-geometry">Resolved geometry</div>
      </ContentHandoff>
    </>
  )
}

function PreparedLateContentProbe() {
  const [isLoading, setIsLoading] = React.useState(true)
  const [height, setHeight] = React.useState(280)

  return (
    <>
      <button type="button" onClick={() => setIsLoading(false)}>mark ready</button>
      <button type="button" onClick={() => setHeight(360)}>commit late metadata</button>
      <ContentHandoff
        owner="content-handoff-late-preparation-probe"
        isLoading={isLoading}
        skeleton={<div data-testid="skeleton-geometry">Loading skeleton</div>}
        prepareContentBeforeHandoff
        transitionMode="replace"
      >
        <div data-testid="late-resolved-geometry" data-height={height}>Late resolved geometry</div>
      </ContentHandoff>
    </>
  )
}

function PreparedMultipleRootsProbe() {
  const [isLoading, setIsLoading] = React.useState(true)
  const [isResolved, setIsResolved] = React.useState(false)

  return (
    <>
      <button type="button" onClick={() => setIsLoading(false)}>mark ready</button>
      <button type="button" onClick={() => setIsResolved(true)}>mount resolved geometry</button>
      <ContentHandoff
        owner="content-handoff-multiple-roots-probe"
        isLoading={isLoading}
        skeleton={<div data-testid="skeleton-geometry">Loading skeleton</div>}
        prepareContentBeforeHandoff
        transitionMode="replace"
      >
        <div data-testid="empty-geometry" />
        {isResolved ? <div data-testid="resolved-geometry">Resolved geometry</div> : null}
      </ContentHandoff>
    </>
  )
}

describe("ContentHandoff", () => {
  it("applies skeletonClassName to the skeleton wrapper for viewport-fill handoff owners", () => {
    render(
      <ContentHandoff
        owner="content-handoff-skeleton-class-probe"
        isLoading
        skeleton={<div>Loading skeleton</div>}
        skeletonClassName="flex min-h-0 flex-1 flex-col"
      >
        <div>Resolved content</div>
      </ContentHandoff>
    )

    const skeleton = screen.getByText("Loading skeleton").closest(".loading-handoff__skeleton")

    expect(skeleton).toHaveClass("flex", "min-h-0", "flex-1", "flex-col")
  })

  it("can mount readiness-signaling content while the skeleton remains the visible loading owner", async () => {
    render(<HandoffProbe />)

    expect(screen.getByText("Loading skeleton")).toBeInTheDocument()

    await waitFor(() => {
      expect(screen.getByText("Resolved content")).toBeInTheDocument()
    })

    const handoff = screen.getByText("Resolved content").closest("[data-slot='content-handoff']")

    await waitFor(() => {
      expect(handoff).toHaveAttribute("data-loading-phase", "handoff")
    })
  })

  it("keeps the visible loading surface on skeleton geometry when hidden content arrives late", async () => {
    render(
      <ContentHandoff
        owner="content-handoff-hidden-content-probe"
        isLoading
        skeleton={<div data-testid="skeleton-geometry">Loading skeleton</div>}
        mountContentWhileLoading
      >
        <DelayedIntrinsicContent />
      </ContentHandoff>
    )

    const handoff = screen.getByText("Loading skeleton").closest("[data-slot='content-handoff']")
    const hiddenContent = screen.getByText("Resolved geometry").closest(".loading-handoff__content")
    expect(handoff).not.toBeNull()
    expect(hiddenContent).not.toBeNull()

    expect(handoff).not.toHaveStyle({ minHeight: "200px" })
    expect(handoff).toHaveStyle({ overflow: "clip" })
    expect(hiddenContent).toHaveStyle({ position: "absolute" })

    await waitFor(() => {
      expect(screen.getByTestId("resolved-geometry")).toBeInTheDocument()
    })

    act(() => {
      notifyActiveMutationObservers()
      notifyActiveResizeObservers()
    })

    expect(handoff).not.toHaveStyle({ minHeight: "280px" })
    expect(handoff).toHaveAttribute("data-loading-phase", "loading")
  })

  it("keeps the skeleton as the only visible owner until hidden content has two stable frames", async () => {
    render(<PreparedHandoffProbe />)

    const handoff = screen.getByText("Loading skeleton").closest("[data-slot='content-handoff']")
    const skeleton = screen.getByText("Loading skeleton").closest(".loading-handoff__skeleton")
    const content = screen.getByTestId("resolved-geometry").closest(".loading-handoff__content")

    expect(handoff).not.toBeNull()
    expect(skeleton).not.toBeNull()
    expect(content).not.toBeNull()
    expect(content).toHaveAttribute("aria-hidden", "true")

    fireEvent.click(screen.getByRole("button", { name: "mark ready" }))

    await waitFor(() => {
      expect(handoff).toHaveAttribute("data-loading-phase", "loading")
    })

    await act(async () => {
      flushAnimationFrame()
    })

    expect(handoff).toHaveAttribute("data-loading-phase", "loading")
    expect(handoff).not.toHaveStyle({ minHeight: "280px" })
    expect(skeleton).not.toHaveStyle({ minHeight: "280px" })
    expect(content).toHaveStyle({ position: "absolute" })

    await act(async () => {
      flushAnimationFrame()
    })

    await waitFor(() => {
      expect(handoff).toHaveAttribute("data-loading-phase", "handoff")
      expect(handoff).toHaveAttribute("data-loading-content-ready", "true")
    })
  })

  it("uses the same prepared stability gate before replace-mode content becomes visible", async () => {
    render(<PreparedHandoffProbe transitionMode="replace" />)

    const handoff = screen.getByText("Loading skeleton").closest("[data-slot='content-handoff']")
    expect(handoff).not.toBeNull()

    fireEvent.click(screen.getByRole("button", { name: "mark ready" }))

    await act(async () => {
      flushAnimationFrame()
    })

    expect(handoff).toHaveAttribute("data-loading-phase", "loading")

    await act(async () => {
      flushAnimationFrame()
    })

    await waitFor(() => {
      expect(handoff).toHaveAttribute("data-loading-phase", "content")
    })
    expect(screen.queryByText("Loading skeleton")).not.toBeInTheDocument()
  })

  it("waits for positive content geometry and measures every direct content root", async () => {
    render(<PreparedMultipleRootsProbe />)

    const handoff = screen.getByText("Loading skeleton").closest("[data-slot='content-handoff']")
    expect(handoff).not.toBeNull()

    fireEvent.click(screen.getByRole("button", { name: "mark ready" }))

    await act(async () => {
      flushAnimationFrame()
      flushAnimationFrame()
    })

    expect(handoff).toHaveAttribute("data-loading-phase", "loading")

    fireEvent.click(screen.getByRole("button", { name: "mount resolved geometry" }))
    act(() => {
      notifyActiveMutationObservers()
      notifyActiveResizeObservers()
    })

    await act(async () => {
      flushAnimationFrame()
    })
    expect(handoff).toHaveAttribute("data-loading-phase", "loading")

    await act(async () => {
      flushAnimationFrame()
    })

    await waitFor(() => {
      expect(handoff).toHaveAttribute("data-loading-phase", "content")
    })
  })

  it("restarts the prepared readiness gate for late metadata without resizing the skeleton", async () => {
    render(<PreparedLateContentProbe />)

    const handoff = screen.getByText("Loading skeleton").closest("[data-slot='content-handoff']")
    expect(handoff).not.toBeNull()

    fireEvent.click(screen.getByRole("button", { name: "mark ready" }))

    await act(async () => {
      flushAnimationFrame()
    })

    fireEvent.click(screen.getByRole("button", { name: "commit late metadata" }))

    act(() => {
      notifyActiveMutationObservers()
      notifyActiveResizeObservers()
    })

    await act(async () => {
      flushAnimationFrame()
    })

    expect(handoff).toHaveAttribute("data-loading-phase", "loading")
    expect(handoff).not.toHaveStyle({ minHeight: "360px" })

    await act(async () => {
      flushAnimationFrame()
    })

    await waitFor(() => {
      expect(handoff).toHaveAttribute("data-loading-phase", "content")
      expect(handoff).not.toHaveStyle({ minHeight: "360px" })
    })
  })

  it("keeps the outgoing skeleton opaque until delayed handoff content commits", () => {
    vi.useFakeTimers()

    render(<ExternallyReadyDelayedContentProbe />)

    const handoff = screen.getByText("Loading skeleton").closest("[data-slot='content-handoff']")
    expect(handoff).not.toBeNull()

    act(() => {
      fireEvent.click(screen.getByRole("button", { name: "mark ready" }))
    })

    expect(handoff).toHaveAttribute("data-loading-phase", "handoff")
    expect(handoff).not.toHaveAttribute("data-loading-content-ready", "true")
    const content = handoff?.querySelector(".loading-handoff__content")
    expect(content).toHaveAttribute("aria-hidden", "true")
    expect(content).toHaveAttribute("data-loading-hidden", "true")

    act(() => {
      vi.advanceTimersByTime(500)
    })

    expect(handoff).toHaveAttribute("data-loading-phase", "handoff")
    expect(screen.getByText("Loading skeleton")).toBeInTheDocument()

    act(() => {
      fireEvent.click(screen.getByRole("button", { name: "mount content" }))
    })

    act(() => {
      notifyActiveMutationObservers()
    })

    expect(handoff).toHaveAttribute("data-loading-content-ready", "true")
    expect(content).not.toHaveAttribute("aria-hidden")
    expect(content).not.toHaveAttribute("data-loading-hidden")

    act(() => {
      vi.advanceTimersByTime(180)
    })

    expect(handoff).toHaveAttribute("data-loading-phase", "content")
  })

  it("does not start the ordinary handoff exit for a zero-height content shell", () => {
    vi.useFakeTimers()

    render(<ExternallyReadyEmptyShellProbe />)

    const handoff = screen.getByText("Loading skeleton").closest("[data-slot='content-handoff']")
    expect(handoff).not.toBeNull()

    act(() => {
      fireEvent.click(screen.getByRole("button", { name: "mark ready" }))
    })

    expect(handoff).toHaveAttribute("data-loading-phase", "handoff")
    expect(handoff).not.toHaveAttribute("data-loading-content-ready", "true")

    act(() => {
      vi.advanceTimersByTime(500)
    })

    expect(handoff).toHaveAttribute("data-loading-phase", "handoff")
    expect(screen.getByText("Loading skeleton")).toBeInTheDocument()

    act(() => {
      fireEvent.click(screen.getByRole("button", { name: "mount resolved geometry" }))
    })

    act(() => {
      notifyActiveMutationObservers()
      notifyActiveResizeObservers()
    })

    expect(handoff).toHaveAttribute("data-loading-content-ready", "true")

    act(() => {
      vi.advanceTimersByTime(180)
    })

    expect(handoff).toHaveAttribute("data-loading-phase", "content")
  })
})
