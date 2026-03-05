import { describe, expect, it } from "vitest"
import { sanitizeBarShapeProps } from "./bar-shape"

describe("sanitizeBarShapeProps", () => {
  it("只保留 rect 几何属性，忽略 Recharts 私有字段", () => {
    const input = {
      x: 12,
      y: 24,
      width: 36,
      height: 48,
      index: 2,
      tooltipPayload: [{ value: 123 }],
      payload: { date: "2026-03-05" },
    }

    const result = sanitizeBarShapeProps(input)
    expect(result).toEqual({
      x: 12,
      y: 24,
      width: 36,
      height: 48,
      index: 2,
    })
    expect("tooltipPayload" in (result as Record<string, unknown>)).toBe(false)
    expect("payload" in (result as Record<string, unknown>)).toBe(false)
  })

  it("无效值回退为安全默认值", () => {
    const result = sanitizeBarShapeProps({
      x: "12",
      y: null,
      width: -20,
      height: Number.NaN,
      index: "bad",
    })

    expect(result).toEqual({
      x: 0,
      y: 0,
      width: 0,
      height: 0,
      index: undefined,
    })
  })
})
