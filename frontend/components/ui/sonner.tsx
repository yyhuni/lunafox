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

const Toaster = ({ style, toastOptions, ...props }: ToasterProps) => {
  const { currentTheme } = useColorTheme()

  const compactToastStyle: React.CSSProperties = {
    width: "fit-content",
    maxWidth: "calc(100vw - 2rem)",
    left: 0,
    right: 0,
    marginInline: "auto",
    padding: "8px 12px",
    gap: "8px",
    borderRadius: "var(--radius-control)",
    fontSize: "12px",
    ...toastOptions?.style,
  }

  return (
    <Sonner
      {...props}
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
          maxWidth: "calc(100vw - 2rem)",
          ...style,
        } as React.CSSProperties
      }
      toastOptions={{
        ...toastOptions,
        style: compactToastStyle,
        classNames: {
          title: "!text-xs !leading-4",
          description: "!text-xs !leading-4",
          ...toastOptions?.classNames,
        },
      }}
    />
  )
}

export { Toaster }
