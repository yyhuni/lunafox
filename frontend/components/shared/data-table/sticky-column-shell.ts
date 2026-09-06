export function getDataTableStickyColumnShellClassName(
  stickyRight: boolean | undefined,
  slot: "header" | "cell"
) {
  if (!stickyRight) {
    return undefined
  }

  return slot === "header"
    ? "sticky right-0 z-20 bg-inherit"
    : "sticky right-0 z-10 bg-inherit"
}
