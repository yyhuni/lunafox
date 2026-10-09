import { fireEvent, render, screen } from "@testing-library/react"
import React from "react"
import { describe, expect, it, vi } from "vitest"

import { EngineHttpHeadersPopover } from "@/components/scan/engine-http-headers-popover"

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string, params?: Record<string, unknown>) => {
    if (key === "headersConfigured" && params?.count) {
      return `已配置 ${params.count} 项标头`
    }
    if (key === "noHeadersConfigured") return "点击配置 Cookie / 请求头..."
    if (key === "configureHeaders") return "配置请求头与 Cookie"
    if (key === "headerNamePlaceholder") return "标头名称"
    if (key === "headerValuePlaceholder") return "标头值 (密文保护)"
    if (key === "addCustomHeader") return "添加自定义标头"
    if (key === "emptyTitle") return "暂未设置任何标头"
    if (key === "done") return "完成"
    if (key === "cancel") return "取消"
    if (key === "clearAllHeaders") return "清空"
    return key
  },
}))

function openPopover(triggerName: RegExp) {
  fireEvent.click(screen.getByRole("button", { name: triggerName }))
}

describe("EngineHttpHeadersPopover", () => {
  it("renders unconfigured state when value is empty", () => {
    const onChange = vi.fn()
    render(<EngineHttpHeadersPopover fieldId="headers-field" value={[]} onChange={onChange} />)

    expect(screen.getByRole("button", { name: /点击配置 Cookie \/ 请求头\.\.\./i })).toBeDefined()
  })

  it("renders configured count summary when value has items", () => {
    const onChange = vi.fn()
    render(
      <EngineHttpHeadersPopover
        fieldId="headers-field"
        value={["Cookie: PHPSESSID=test_123", "Authorization: Bearer token_abc"]}
        onChange={onChange}
      />
    )

    expect(screen.getByText(/已配置 2 项标头/)).toBeDefined()
    expect(screen.getByText(/Cookie, Authorization/)).toBeDefined()
  })

  it("adds a Cookie preset row that commits only after the user enters a value", () => {
    const onChange = vi.fn()
    render(<EngineHttpHeadersPopover fieldId="headers-field" value={[]} onChange={onChange} />)

    openPopover(/点击配置 Cookie \/ 请求头\.\.\./i)
    fireEvent.click(screen.getByRole("button", { name: /\+ Cookie/i }))

    // Preset prefills only the name; an empty-value row must not serialize.
    const valueInput = screen.getByLabelText("标头值 (密文保护)") as HTMLInputElement
    fireEvent.change(valueInput, { target: { value: "A=1; B=2" } })
    fireEvent.click(screen.getByRole("button", { name: "完成" }))

    expect(onChange).toHaveBeenCalledTimes(1)
    expect(onChange).toHaveBeenCalledWith(["Cookie: A=1; B=2"])
  })

  it("never serializes an incomplete preset row with an empty value", () => {
    const onChange = vi.fn()
    render(<EngineHttpHeadersPopover fieldId="headers-field" value={[]} onChange={onChange} />)

    openPopover(/点击配置 Cookie \/ 请求头\.\.\./i)
    fireEvent.click(screen.getByRole("button", { name: /\+ Cookie/i }))
    fireEvent.click(screen.getByRole("button", { name: "完成" }))

    expect(onChange).toHaveBeenCalledWith([])
  })

  it("prefills only the Bearer scheme prefix for the Token preset", () => {
    const onChange = vi.fn()
    render(<EngineHttpHeadersPopover fieldId="headers-field" value={[]} onChange={onChange} />)

    openPopover(/点击配置 Cookie \/ 请求头\.\.\./i)
    fireEvent.click(screen.getByRole("button", { name: /\+ Token/i }))

    const valueInput = screen.getByLabelText("标头值 (密文保护)") as HTMLInputElement
    expect(valueInput.value).toBe("Bearer ")

    fireEvent.change(valueInput, { target: { value: "Bearer unit.test" } })
    fireEvent.click(screen.getByRole("button", { name: "完成" }))

    expect(onChange).toHaveBeenCalledWith(["Authorization: Bearer unit.test"])
  })

  it("clear only resets drafts and keeps the committed value until Done", () => {
    const onChange = vi.fn()
    render(
      <EngineHttpHeadersPopover
        fieldId="headers-field"
        value={["Cookie: session=123"]}
        onChange={onChange}
      />
    )

    openPopover(/已配置 1 项标头/i)
    fireEvent.click(screen.getByRole("button", { name: "清空" }))

    expect(onChange).not.toHaveBeenCalled()
    expect(screen.getByText("暂未设置任何标头")).toBeDefined()

    fireEvent.click(screen.getByRole("button", { name: "完成" }))
    expect(onChange).toHaveBeenCalledTimes(1)
    expect(onChange).toHaveBeenCalledWith([])
  })
})
