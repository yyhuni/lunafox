import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/providers/query-provider.tsx"), "utf8")

describe("query-provider contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function QueryProvider")
    expect(source).toContain("from \"react\"")
  })

  it("keeps a stable browser QueryClient across same-document refreshes", () => {
    expect(source).toContain("QueryClientGlobal")
    expect(source).toContain("__lunafoxQueryClient")
    expect(source).toContain("globalForQueryClient.__lunafoxQueryClient ??= createQueryClient()")
    expect(source).toContain("const [queryClient] = React.useState(getQueryClient)")
  })
})
