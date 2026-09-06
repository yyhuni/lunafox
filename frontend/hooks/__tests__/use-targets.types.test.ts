import { describe, expectTypeOf, it } from "vitest"
import { useTargets, type UseTargetsResult } from "@/hooks/use-targets"
import type { Target } from "@/types/target.types"

type UseTargetsObjectCall = (
  params: { pageSize?: number; pageToken?: string; filter?: string; orderBy?: string },
  options?: { enabled?: boolean }
) => UseTargetsResult

describe("useTargets types", () => {
  it("对象参数调用返回稳定类型", () => {
    expectTypeOf(useTargets).toMatchTypeOf<UseTargetsObjectCall>()
  })

  it("返回类型可直接访问分页与兼容字段", () => {
    type TargetsData = NonNullable<UseTargetsResult["data"]>

    expectTypeOf<TargetsData["targets"]>().toEqualTypeOf<Target[]>()
    expectTypeOf<TargetsData["count"]>().toEqualTypeOf<number>()
    expectTypeOf<TargetsData["total"]>().toEqualTypeOf<number>()
    expectTypeOf<TargetsData["page"]>().toEqualTypeOf<number>()
    expectTypeOf<TargetsData["pageSize"]>().toEqualTypeOf<number>()
    expectTypeOf<TargetsData["totalPages"]>().toEqualTypeOf<number>()
  })
})
