import { fireEvent, render, screen } from "@testing-library/react"
import React from "react"
import { describe, expect, it, vi } from "vitest"

import {
  durationPartsFromSeconds,
  durationSecondsFromParts,
  EngineDurationInput,
} from "@/components/scan/engine-duration-input"

vi.mock("@ncdai/react-wheel-picker", () => ({
  WheelPicker: ({
    value,
    options,
    onValueChange,
  }: {
    value: number
    options: Array<{ value: number; label: React.ReactNode }>
    onValueChange: (value: number) => void
  }) => (
    <select
      aria-label={`wheel-${options.length}`}
      value={value}
      onChange={(event) => onValueChange(Number(event.target.value))}
    >
      {options.map((option) => (
        <option key={option.value} value={option.value}>
          {option.label}
        </option>
      ))}
    </select>
  ),
  WheelPickerWrapper: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

function openDurationPicker() {
  fireEvent.click(screen.getByRole("button", { name: /chooseDuration/ }))
}

describe("EngineDurationInput", () => {
  it("decomposes and recomposes canonical seconds", () => {
    const parts = durationPartsFromSeconds(90061)
    expect(parts).toEqual({ days: 1, hours: 1, minutes: 1, seconds: 1 })
    expect(durationSecondsFromParts(parts)).toBe(90061)
  })

  it("keeps wheel edits local until Apply and writes one integer value", () => {
    const onChange = vi.fn()
    render(<EngineDurationInput id="timeout" value={3600} minimum={60} maximum={604800} onChange={onChange} />)

    openDurationPicker()
    fireEvent.change(screen.getByRole("combobox", { name: "wheel-24" }), { target: { value: "2" } })
    expect(onChange).not.toHaveBeenCalled()

    fireEvent.click(screen.getByRole("button", { name: /apply/i }))
    expect(onChange).toHaveBeenCalledWith(7200)
  })

  it("discards a wheel draft on Cancel", () => {
    const onChange = vi.fn()
    render(<EngineDurationInput id="timeout" value={3600} minimum={60} onChange={onChange} />)

    openDurationPicker()
    fireEvent.change(screen.getAllByRole("combobox", { name: "wheel-60" })[0]!, { target: { value: "30" } })
    fireEvent.click(screen.getByRole("button", { name: /cancel/i }))
    expect(onChange).not.toHaveBeenCalled()

    openDurationPicker()
    expect(screen.getAllByRole("combobox", { name: "wheel-60" })[0]!).toHaveValue("0")
  })

  it("blocks Apply for an out-of-range draft instead of clamping it", () => {
    const onChange = vi.fn()
    render(<EngineDurationInput id="timeout" value={3600} minimum={60} maximum={3600} onChange={onChange} />)

    openDurationPicker()
    fireEvent.change(screen.getByRole("combobox", { name: "wheel-24" }), { target: { value: "2" } })
    expect(screen.getAllByText(/durationOutOfRange/).length).toBeGreaterThan(0)
    expect(screen.getByRole("button", { name: /apply/i })).toBeDisabled()
    expect(onChange).not.toHaveBeenCalled()
  })

  it("loads the catalog default into the draft without applying it automatically", () => {
    const onChange = vi.fn()
    render(<EngineDurationInput id="timeout" value={3600} defaultValue={7200} minimum={60} onChange={onChange} />)

    openDurationPicker()
    fireEvent.click(screen.getByRole("button", { name: /restoreDefault/i }))
    expect(onChange).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole("button", { name: /apply/i }))
    expect(onChange).toHaveBeenCalledWith(7200)
  })
})
