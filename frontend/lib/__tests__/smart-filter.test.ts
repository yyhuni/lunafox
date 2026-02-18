import { beforeEach, describe, expect, it } from "vitest"
import {
  buildDefaultPlaceholder,
  getFieldHistory,
  getGhostText,
  parseFilterExpression,
  PREDEFINED_FIELDS,
  saveQueryHistory,
} from "@/lib/smart-filter"

describe("smart-filter helpers", () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it("parseFilterExpression 解析多个条件", () => {
    const filters = parseFilterExpression('ip="1.1.1.1" && port!="80"')
    expect(filters).toEqual([
      {
        field: "ip",
        operator: "=",
        value: "1.1.1.1",
        raw: 'ip="1.1.1.1"',
      },
      {
        field: "port",
        operator: "!=",
        value: "80",
        raw: 'port!="80"',
      },
    ])
  })

  it("saveQueryHistory 会记录字段历史", () => {
    saveQueryHistory('ip="1.1.1.1" && port="443"')
    expect(getFieldHistory("ip")).toEqual(["1.1.1.1"])
    expect(getFieldHistory("port")).toEqual(["443"])
  })

  it("getGhostText 能基于字段前缀与历史值补全", () => {
    saveQueryHistory('ip="1.1.1.1"')
    const fields = [PREDEFINED_FIELDS.ip, PREDEFINED_FIELDS.port]

    expect(getGhostText("i", fields)).toBe('p="')
    expect(getGhostText('ip="1', fields)).toBe('.1.1.1"')
  })

  it("buildDefaultPlaceholder 优先使用示例", () => {
    const placeholder = buildDefaultPlaceholder(
      [PREDEFINED_FIELDS.ip, PREDEFINED_FIELDS.port],
      ['ip="1.1.1.1"']
    )
    expect(placeholder).toBe('ip="1.1.1.1"')
  })
})
