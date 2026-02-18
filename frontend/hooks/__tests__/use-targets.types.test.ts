import { describe, expectTypeOf, it } from "vitest"
import { useTargets, type UseTargetsResult } from "@/hooks/use-targets"
import type { Target } from "@/types/target.types"

type UseTargetsObjectCall = (
  params: { page?: number; pageSize?: number; organizationId?: number; filter?: string },
  options?: { enabled?: boolean }
) => UseTargetsResult

type UseTargetsPositionalCall = (
  page?: number,
  pageSize?: number,
  type?: string,
  filter?: string
) => UseTargetsResult

describe("useTargets types", () => {
  it("对象参数与位置参数调用都返回稳定类型", () => {
    expectTypeOf(useTargets).toMatchTypeOf<UseTargetsObjectCall>()
    expectTypeOf(useTargets).toMatchTypeOf<UseTargetsPositionalCall>()
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
