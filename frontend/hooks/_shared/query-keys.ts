export type KeyParamFactory<TArgs extends unknown[]> = (...args: TArgs) => unknown

export type ResourceKeyFactories<
  TListArgs extends unknown[] = [],
  TDetailArgs extends unknown[] = [number],
> = {
  list?: KeyParamFactory<TListArgs>
  detail?: KeyParamFactory<TDetailArgs>
}

export function createResourceKeys<
  TListArgs extends unknown[] = [],
  TDetailArgs extends unknown[] = [number],
>(resource: string, factories?: ResourceKeyFactories<TListArgs, TDetailArgs>) {
  const all = [resource] as const
  const lists = () => [...all, "list"] as const
  const details = () => [...all, "detail"] as const

  const list = (...args: TListArgs) =>
    (factories?.list
      ? ([...lists(), factories.list(...args)] as const)
      : ([...lists()] as const))

  const detail = (...args: TDetailArgs) =>
    (factories?.detail
      ? ([...details(), factories.detail(...args)] as const)
      : (args.length
        ? ([...details(), ...args] as const)
        : ([...details()] as const)))

  return {
    all,
    lists,
    list,
    details,
    detail,
  }
}
