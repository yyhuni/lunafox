"use client"

import * as React from "react"
import { mergeProps } from "@base-ui/react/merge-props"
import { useRender } from "@base-ui/react/use-render"

type PolymorphicSlotProps<TTagName extends keyof React.JSX.IntrinsicElements> =
  React.ComponentPropsWithRef<TTagName> & {
    render?: React.ReactElement
  }

function PolymorphicSlot<TTagName extends keyof React.JSX.IntrinsicElements>({
  children,
  defaultTagName,
  render,
  ...props
}: PolymorphicSlotProps<TTagName> & {
  defaultTagName: TTagName
}) {
  return useRender({
    defaultTagName,
    render,
    props: mergeProps(
      defaultTagNameProps<TTagName>(),
      {
        ...props,
        children,
      } as React.ComponentPropsWithRef<TTagName>
    ),
  })
}

function defaultTagNameProps<TTagName extends keyof React.JSX.IntrinsicElements>(): React.ComponentPropsWithRef<TTagName> {
  return {} as React.ComponentPropsWithRef<TTagName>
}

export { PolymorphicSlot }
export type { PolymorphicSlotProps }
