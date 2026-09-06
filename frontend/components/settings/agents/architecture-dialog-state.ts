import React from "react"
import { useTranslations } from "next-intl"

export function useArchitectureDialogState() {
  const t = useTranslations("pages.agents")
  const [open, setOpen] = React.useState(false)

  return {
    t,
    open,
    setOpen,
  }
}
