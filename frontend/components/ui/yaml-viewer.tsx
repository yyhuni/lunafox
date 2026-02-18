"use client"

import React, { useEffect, useRef } from "react"
import { EditorState } from "@codemirror/state"
import { EditorView, lineNumbers, highlightSpecialChars } from "@codemirror/view"
import { yaml } from "@codemirror/lang-yaml"
import { oneDark } from "@codemirror/theme-one-dark"
import { useColorTheme } from "@/hooks/use-color-theme"
import { cn } from "@/lib/utils"

// Light theme - matches shadcn/ui style
const lightTheme = EditorView.theme({
  "&": {
    backgroundColor: "transparent",
    fontSize: "13px",
  },
  ".cm-content": {
    fontFamily: "ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Monaco, Consolas, monospace",
    padding: "12px 0",
  },
  ".cm-gutters": {
    backgroundColor: "transparent",
    border: "none",
    color: "hsl(var(--muted-foreground))",
  },
  ".cm-lineNumbers .cm-gutterElement": {
    padding: "0 12px 0 16px",
    minWidth: "40px",
  },
  ".cm-line": {
    padding: "0 16px",
  },
  ".cm-activeLine": {
    backgroundColor: "hsl(var(--muted) / 0.5)",
  },
  ".cm-selectionBackground": {
    backgroundColor: "hsl(var(--primary) / 0.2) !important",
  },
  "&.cm-focused .cm-selectionBackground": {
    backgroundColor: "hsl(var(--primary) / 0.3) !important",
  },
  ".cm-cursor": {
    borderLeftColor: "hsl(var(--foreground))",
  },
}, { dark: false })

// Dark theme extension
const darkThemeExt = EditorView.theme({
  "&": {
    backgroundColor: "transparent",
    fontSize: "13px",
  },
  ".cm-content": {
    fontFamily: "ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Monaco, Consolas, monospace",
    padding: "12px 0",
  },
  ".cm-gutters": {
    backgroundColor: "transparent",
    border: "none",
  },
  ".cm-lineNumbers .cm-gutterElement": {
    padding: "0 12px 0 16px",
    minWidth: "40px",
  },
  ".cm-line": {
    padding: "0 16px",
  },
}, { dark: true })

interface YamlViewerProps {
  value: string
  className?: string
  showLineNumbers?: boolean
  maxHeight?: string
}

export function YamlViewer({ 
  value, 
  className,
  showLineNumbers = true,
  maxHeight = "100%"
}: YamlViewerProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const viewRef = useRef<EditorView | null>(null)
  const { currentTheme } = useColorTheme()

  useEffect(() => {
    if (!containerRef.current) return

    const extensions = [
      yaml(),
      EditorState.readOnly.of(true),
      EditorView.editable.of(false),
      highlightSpecialChars(),
      currentTheme.isDark ? oneDark : lightTheme,
      currentTheme.isDark ? darkThemeExt : [],
    ].flat()

    if (showLineNumbers) {
      extensions.push(lineNumbers())
    }

    const state = EditorState.create({
      doc: value,
      extensions,
    })

    // Destroy previous view if exists
    if (viewRef.current) {
      viewRef.current.destroy()
    }

    const view = new EditorView({
      state,
      parent: containerRef.current,
    })

    viewRef.current = view

    return () => {
      view.destroy()
    }
  }, [value, currentTheme.isDark, showLineNumbers])

  return (
    <div 
      ref={containerRef} 
      className={cn(
        "overflow-auto rounded-lg border bg-muted/30",
        className
      )}
      style={{ maxHeight }}
    />
  )
}
