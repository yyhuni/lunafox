// Scan-workbench popovers (duration picker, headers editor) sit on the
// compact 28px control tier — one step below the form's 32px inputs — so
// popover contents stay denser than the surrounding form. Every control
// inside these popovers references this single constant; band paddings pair
// with it as px-3 py-1.5. Do not introduce per-popover height overrides.
export const compactControlTier = "h-7 text-xs"
