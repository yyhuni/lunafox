"use client"

import {
  CircleCheckIcon,
  InfoIcon,
  Loader2Icon,
  OctagonXIcon,
  TriangleAlertIcon,
} from "@/components/icons"
import { useColorTheme } from "@/hooks/use-color-theme"
import { Toaster as Sonner, ToasterProps } from "sonner"

const Toaster = ({ ...props }: ToasterProps) => {
  const { currentTheme } = useColorTheme()

  return (
    <Sonner
      theme={currentTheme.isDark ? "dark" : "light"}
      position="top-center"
      expand={false}
      visibleToasts={3}
      duration={4000}
      gap={12}
      className="group toaster"
      icons={{
        success: <CircleCheckIcon className="size-4" />,
        info: <InfoIcon className="size-4" />,
        warning: <TriangleAlertIcon className="size-4" />,
        error: <OctagonXIcon className="size-4" />,
        loading: <Loader2Icon aria-hidden="true" data-slot="toast-loading-icon" className="animate-spin size-4" />,
      }}
      style={
        {
          "--normal-bg": "var(--card)",
          "--normal-text": "var(--card-foreground)",
          "--normal-border": "var(--border)",
          "--border-radius": "var(--radius-overlay)",
        } as React.CSSProperties
      }
      {...props}
    />
  )
}

export { Toaster }
