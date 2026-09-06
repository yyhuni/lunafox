import { renderHook } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

vi.mock("next-intl", () => ({
  useFormatter: () => ({
    dateTime: (_date: Date, options: Intl.DateTimeFormatOptions) => (
      options.year && options.second
        ? "full-precise-date-time"
        : options.second
          ? "current-year-compact-second-precise-date-time"
          : "current-year-compact-date-time"
    ),
    relativeTime: () => "relative-time",
  }),
  useNow: () => new Date("2026-07-27T12:00:00Z"),
  useLocale: () => "zh-CN",
}))

import { useFormatHeartbeatTime } from "../i18n-format"

describe("useFormatHeartbeatTime", () => {
  it("uses a year-free but second-precise absolute display for old heartbeats in the current year when compact", () => {
    const { result } = renderHook(() => useFormatHeartbeatTime(7, "compact"))

    expect(result.current("2026-01-01T12:00:00Z")).toEqual({
      display: "current-year-compact-second-precise-date-time",
      title: "full-precise-date-time",
    })
  })

  it("retains the year and seconds for a cross-year heartbeat in compact mode", () => {
    const { result } = renderHook(() => useFormatHeartbeatTime(7, "compact"))

    expect(result.current("2024-12-29T18:12:00Z")).toEqual({
      display: "full-precise-date-time",
      title: "full-precise-date-time",
    })
  })

  it("keeps recent heartbeats relative regardless of density", () => {
    const { result } = renderHook(() => useFormatHeartbeatTime(7, "compact"))

    expect(result.current("2026-07-27T11:00:00Z")).toEqual({
      display: "relative-time",
      title: "full-precise-date-time",
    })
  })

  it("preserves seconds in the default absolute display", () => {
    const { result } = renderHook(() => useFormatHeartbeatTime())

    expect(result.current("2024-12-29T18:12:00Z")).toEqual({
      display: "full-precise-date-time",
      title: "full-precise-date-time",
    })
  })
})
