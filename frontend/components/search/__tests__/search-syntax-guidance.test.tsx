import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import {
  GLOBAL_ASSET_SEARCH_FEATURED_GUIDANCE,
  GLOBAL_ASSET_SEARCH_FIELDS,
  GLOBAL_ASSET_SEARCH_GUIDANCE,
  parseGlobalAssetSearchQuery,
} from "@/lib/global-asset-search-query"
import {
  appendGlobalAssetSearchCondition,
  getGlobalAssetSearchInlineCompletion,
  GLOBAL_ASSET_SEARCH_SYNTAX_EXAMPLES,
  SearchSyntaxManualContent,
  SearchSyntaxGuidance,
} from "../search-syntax-guidance"

describe("SearchSyntaxGuidance", () => {
  it("uses the parser whitelist and keeps every suggested example valid", () => {
    expect(GLOBAL_ASSET_SEARCH_FIELDS).toEqual(["url", "host", "title", "statusCode", "tech"])
    expect(GLOBAL_ASSET_SEARCH_SYNTAX_EXAMPLES).toHaveLength(GLOBAL_ASSET_SEARCH_GUIDANCE.length)
    expect(GLOBAL_ASSET_SEARCH_FEATURED_GUIDANCE).toHaveLength(6)
    expect(GLOBAL_ASSET_SEARCH_FEATURED_GUIDANCE.map((entry) => entry.query)).toEqual([
      'url="admin"',
      'host="api"',
      'title="Login"',
      'statusCode=="200"',
      'tech="nginx"',
      'host="api" && statusCode=="200"',
    ])
    expect(() => GLOBAL_ASSET_SEARCH_SYNTAX_EXAMPLES.forEach(parseGlobalAssetSearchQuery)).not.toThrow()
  })

  it("builds an editable condition without replacing the existing structured draft", () => {
    expect(appendGlobalAssetSearchCondition("", "host")).toBe('host=""')
    expect(appendGlobalAssetSearchCondition('tech="nginx"', "statusCode")).toBe('tech="nginx" && statusCode=""')
  })

  it("offers only draft-editing shortcuts for allowed syntax", () => {
    const onSelectField = vi.fn()
    const onSelectExample = vi.fn()

    render(
      <SearchSyntaxGuidance
        t={(key) => key}
        onSelectField={onSelectField}
        onSelectExample={onSelectExample}
        onOpenManual={vi.fn()}
      />
    )

    fireEvent.click(screen.getByRole("button", { name: 'host=""' }))
    fireEvent.click(screen.getByRole("button", { name: 'statusCode=="200"' }))

    expect(onSelectField).toHaveBeenCalledWith("host")
    expect(onSelectExample).toHaveBeenCalledWith('statusCode=="200"')
    expect(screen.queryByText("!=")).not.toBeInTheDocument()
    expect(screen.queryByText("||")).not.toBeInTheDocument()
  })

  it("renders field-specific matching semantics in the full manual", () => {
    render(<SearchSyntaxManualContent t={(key) => key} onSelectExample={vi.fn()} />)

    expect(screen.getByText("manual.semanticsTitle")).toBeInTheDocument()
    expect(screen.getByText("manual.textSemantics")).toBeInTheDocument()
    expect(screen.getByText("manual.typedSemantics")).toBeInTheDocument()
  })

  it("completes only strict field and connector syntax", () => {
    expect(getGlobalAssetSearchInlineCompletion("ho")).toBe('st="')
    expect(getGlobalAssetSearchInlineCompletion("host")).toBe('="')
    expect(getGlobalAssetSearchInlineCompletion('statusCode==')).toBe('"')
    expect(getGlobalAssetSearchInlineCompletion('host="api')).toBe('"')
    expect(getGlobalAssetSearchInlineCompletion('host="api"')).toBe(" && ")
    expect(getGlobalAssetSearchInlineCompletion('host="api" && te')).toBe('ch="')
    expect(getGlobalAssetSearchInlineCompletion("domain")).toBe("")
    expect(getGlobalAssetSearchInlineCompletion("||")).toBe("")
  })
})
