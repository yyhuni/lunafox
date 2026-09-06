"use client"

import React, { useEffect, useRef } from "react"
import { EditorState } from "@codemirror/state"
import { EditorView, lineNumbers, highlightSpecialChars } from "@codemirror/view"
import { yaml } from "@codemirror/lang-yaml"
import { codeMirrorDarkTheme, codeMirrorLightTheme } from "@/components/shared/editors/codemirror-theme"
import { useColorTheme } from "@/hooks/use-color-theme"
import { cn } from "@/lib/utils"

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
      currentTheme.isDark ? codeMirrorDarkTheme : codeMirrorLightTheme,
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
