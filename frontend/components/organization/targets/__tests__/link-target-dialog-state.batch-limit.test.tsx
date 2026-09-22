import { act, renderHook, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { useLinkTargetDialogState } from "@/components/organization/targets/link-target-dialog-state"
import { MAX_TARGET_BATCH_SIZE } from "@/lib/target-validator"

const targetHooks = vi.hoisted(() => ({
  useBatchCreateTargets: vi.fn(),
}))

vi.mock("@/hooks/use-targets", () => targetHooks)

function createMutation() {
  return { mutate: vi.fn(), isPending: false }
}

function createTargetInput(count: number, includeBlankLines = false) {
  const targets = Array.from({ length: count }, (_, index) => `target-${index}.example.com`)
  return includeBlankLines ? ["", ...targets, "   "].join("\n\n") : targets.join("\n")
}

function renderState() {
  return renderHook(() => useLinkTargetDialogState({
    organizationId: 42,
    open: true,
    t: (key) => key,
  }))
}

describe("useLinkTargetDialogState batch limit", () => {
  let mutation: ReturnType<typeof createMutation>

  beforeEach(() => {
    vi.clearAllMocks()
    mutation = createMutation()
    targetHooks.useBatchCreateTargets.mockReturnValue(mutation)
  })

  it("accepts 5,000 parsed targets while excluding blank lines from the submitted payload", async () => {
    const { result } = renderState()
    const input = createTargetInput(MAX_TARGET_BATCH_SIZE, true)

    await act(async () => {
      result.current.form.setValue("targets", input, { shouldValidate: true })
    })

    await waitFor(() => {
      expect(result.current.isFormValid).toBe(true)
    })
    expect(result.current.targetValidation.count).toBe(MAX_TARGET_BATCH_SIZE)
    expect(result.current.isTargetBatchOverLimit).toBe(false)

    act(() => {
      result.current.onSubmit({ targets: input })
    })

    expect(mutation.mutate).toHaveBeenCalledTimes(1)
    const [payload] = mutation.mutate.mock.calls[0]
    expect(payload.organizationIds).toEqual([42])
    expect(payload.targets).toHaveLength(MAX_TARGET_BATCH_SIZE)
    expect(payload.targets[0]).toEqual({ name: "target-0.example.com" })
    expect(payload.targets[MAX_TARGET_BATCH_SIZE - 1]).toEqual({
      name: `target-${MAX_TARGET_BATCH_SIZE - 1}.example.com`,
    })
  })

  it("blocks 5,001 parsed targets in the form state and submit handler", async () => {
    const { result } = renderState()
    const input = createTargetInput(MAX_TARGET_BATCH_SIZE + 1)

    await act(async () => {
      result.current.form.setValue("targets", input, { shouldValidate: true })
    })

    await waitFor(() => {
      expect(result.current.targetValidation.count).toBe(MAX_TARGET_BATCH_SIZE + 1)
    })
    expect(result.current.isTargetBatchOverLimit).toBe(true)
    expect(result.current.isFormValid).toBe(false)

    act(() => {
      result.current.onSubmit({ targets: input })
    })

    expect(mutation.mutate).not.toHaveBeenCalled()
  })
})
