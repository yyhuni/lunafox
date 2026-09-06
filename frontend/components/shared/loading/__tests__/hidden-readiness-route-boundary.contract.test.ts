import { describe, expect, it } from "vitest"
import { existsSync, readFileSync } from "node:fs"
import path from "node:path"

const sourcePath = path.resolve(process.cwd(), "components/shared/loading/hidden-readiness-route-boundary.tsx")

function readSource() {
  if (!existsSync(sourcePath)) {
    expect.fail("components/shared/loading/hidden-readiness-route-boundary.tsx is missing")
  }
  return readFileSync(sourcePath, "utf8")
}

describe("hidden-readiness-route-boundary contract", () => {
  it("owns a narrow route-level hidden-readiness wrapper contract", () => {
    const source = readSource()

    expect(source).toContain("export interface HiddenReadinessRouteBoundaryProps")
    expect(source).toContain("owner: string")
    expect(source).toContain("skeleton: React.ReactNode")
    expect(source).toContain("children: (props: HiddenReadinessRouteBoundaryRenderProps) => React.ReactNode")
    expect(source).toContain("layer?: LoadingLayer")
    expect(source).toContain("intent?: LoadingIntent")
    expect(source).toContain("skeletonClassName?: string")
    expect(source).toContain("contentClassName?: string")
    expect(source).toContain("initiallyDeferSkeleton?: boolean")
    expect(source).toContain('transitionMode?: "crossfade" | "replace"')
    expect(source).toContain("export function HiddenReadinessRouteBoundary")
    expect(source).toContain("const [shouldDeferInitialSkeleton] = React.useState(() => initiallyDeferSkeleton)")
    expect(source).toContain("const [isReady, setIsReady] = React.useState(() => !shouldDeferInitialSkeleton)")
    expect(source).toContain("const deferInitialSkeleton = shouldDeferInitialSkeleton && !isReady")
    expect(source).toContain("setIsReady(true)")
    expect(source).toContain("<ContentHandoff")
    expect(source).toContain('layer = "route"')
    expect(source).toContain('intent = "route"')
    expect(source).toContain("layer={layer}")
    expect(source).toContain("intent={intent}")
    expect(source).toContain("isLoading={deferInitialSkeleton}")
    expect(source).toContain("mountContentWhileLoading")
    expect(source).toContain("prepareContentBeforeHandoff")
    expect(source).toContain("transitionMode={transitionMode}")
    expect(source).toContain("children({ onReady: handleReady, deferInitialSkeleton })")
  })
})
