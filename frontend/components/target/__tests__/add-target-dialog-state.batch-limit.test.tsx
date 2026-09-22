import { act, renderHook } from "@testing-library/react"
import type { FormEvent } from "react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { useAddTargetDialogState } from "@/components/target/add-target-dialog-state"
import { MAX_TARGET_BATCH_SIZE } from "@/lib/target-validator"

const organizationHooks = vi.hoisted(() => ({
  useOrganizations: vi.fn(),
}))

const targetHooks = vi.hoisted(() => ({
  useBatchCreateTargets: vi.fn(),
}))

vi.mock("@/hooks/use-organizations", () => organizationHooks)
vi.mock("@/hooks/use-targets", () => targetHooks)

function createMutation() {
  return { mutate: vi.fn(), isPending: false }
}

function createTargetInput(count: number, includeBlankLines = false) {
  const targets = Array.from({ length: count }, (_, index) => `target-${index}.example.com`)
  return includeBlankLines ? ["", ...targets, "   "].join("\n\n") : targets.join("\n")
}

function renderState() {
  return renderHook(() => useAddTargetDialogState({
    externalOpen: true,
    t: (key) => key,
  }))
}

describe("useAddTargetDialogState batch limit", () => {
  let mutation: ReturnType<typeof createMutation>

  beforeEach(() => {
    vi.clearAllMocks()
    mutation = createMutation()
    targetHooks.useBatchCreateTargets.mockReturnValue(mutation)
    organizationHooks.useOrganizations.mockReturnValue({
      data: { organizations: [], totalSize: 0 },
      isLoading: false,
      isPlaceholderData: false,
    })
  })

  it("accepts 5,000 parsed targets while excluding blank lines from the submitted payload", () => {
    const { result } = renderState()
    const input = createTargetInput(MAX_TARGET_BATCH_SIZE, true)

    act(() => {
      result.current.handleInputChange("targets", input)
    })

    expect(result.current.targetCount).toBe(MAX_TARGET_BATCH_SIZE)
    expect(result.current.isTargetBatchOverLimit).toBe(false)
    expect(result.current.isFormValid).toBe(true)

    const event = { preventDefault: vi.fn() } as unknown as FormEvent
    act(() => {
      result.current.handleSubmit(event)
    })

    expect(event.preventDefault).toHaveBeenCalledTimes(1)
    expect(mutation.mutate).toHaveBeenCalledTimes(1)
    const [payload] = mutation.mutate.mock.calls[0]
    expect(payload.targets).toHaveLength(MAX_TARGET_BATCH_SIZE)
    expect(payload.targets[0]).toEqual({ name: "target-0.example.com" })
    expect(payload.targets[MAX_TARGET_BATCH_SIZE - 1]).toEqual({
      name: `target-${MAX_TARGET_BATCH_SIZE - 1}.example.com`,
    })
  })

  it("blocks 5,001 parsed targets in the form state and submit handler", () => {
    const { result } = renderState()

    act(() => {
      result.current.handleInputChange("targets", createTargetInput(MAX_TARGET_BATCH_SIZE + 1))
    })

    expect(result.current.targetCount).toBe(MAX_TARGET_BATCH_SIZE + 1)
    expect(result.current.isTargetBatchOverLimit).toBe(true)
    expect(result.current.isFormValid).toBe(false)

    act(() => {
      result.current.handleSubmit({ preventDefault: vi.fn() } as unknown as FormEvent)
    })

    expect(mutation.mutate).not.toHaveBeenCalled()
  })
})
