"use client"

import React, { useEffect, useRef } from "react"
import { EditorState, Extension } from "@codemirror/state"
import { 
  EditorView, 
  lineNumbers, 
  highlightActiveLine, 
  highlightSpecialChars,
  keymap,
  placeholder as placeholderExt
} from "@codemirror/view"
import { defaultKeymap, indentWithTab, history, historyKeymap } from "@codemirror/commands"
import { indentOnInput, bracketMatching, foldGutter, foldKeymap } from "@codemirror/language"
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
    caretColor: "hsl(var(--foreground))",
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
  ".cm-foldGutter .cm-gutterElement": {
    padding: "0 4px",
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
    borderLeftWidth: "2px",
  },
  ".cm-placeholder": {
    color: "hsl(var(--muted-foreground))",
  },
  ".cm-foldPlaceholder": {
    backgroundColor: "hsl(var(--muted))",
    border: "none",
    color: "hsl(var(--muted-foreground))",
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
  ".cm-foldGutter .cm-gutterElement": {
    padding: "0 4px",
  },
  ".cm-line": {
    padding: "0 16px",
  },
}, { dark: true })

export type CodeLanguage = "yaml" | "plaintext"

interface CodeEditorProps {
  value: string
  onChange?: (value: string) => void
  language?: CodeLanguage
  placeholder?: string
  readOnly?: boolean
  className?: string
  showLineNumbers?: boolean
  showFoldGutter?: boolean
}

export function CodeEditor({ 
  value, 
  onChange,
  language = "yaml",
  placeholder,
  readOnly = false,
  className,
  showLineNumbers = true,
  showFoldGutter = true,
}: CodeEditorProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const viewRef = useRef<EditorView | null>(null)
  const { currentTheme } = useColorTheme()
  
  // Store callbacks in refs to avoid recreating editor on every render
  const onChangeRef = useRef(onChange)
  onChangeRef.current = onChange
  const valueRef = useRef(value)
  valueRef.current = value

  useEffect(() => {
    if (!containerRef.current) return

    const extensions: Extension[] = [
      highlightSpecialChars(),
      history(),
      bracketMatching(),
      indentOnInput(),
      keymap.of([
        ...defaultKeymap,
        ...historyKeymap,
        ...foldKeymap,
        indentWithTab,
      ]),
      EditorView.lineWrapping,
      // Theme
      currentTheme.isDark ? oneDark : lightTheme,
      currentTheme.isDark ? darkThemeExt : [],
    ]

    // Language support
    if (language === "yaml") {
      extensions.push(yaml())
    }

    // Line numbers
    if (showLineNumbers) {
      extensions.push(lineNumbers())
      extensions.push(highlightActiveLine())
    }

    // Fold gutter
    if (showFoldGutter && language === "yaml") {
      extensions.push(foldGutter())
    }

    // Placeholder
    if (placeholder) {
      extensions.push(placeholderExt(placeholder))
    }

    // Read-only mode
    if (readOnly) {
      extensions.push(EditorState.readOnly.of(true))
      extensions.push(EditorView.editable.of(false))
    } else {
      // Update listener for editable mode
      extensions.push(EditorView.updateListener.of((update) => {
        if (update.docChanged && onChangeRef.current) {
          onChangeRef.current(update.state.doc.toString())
        }
      }))
    }

    const state = EditorState.create({
      doc: valueRef.current,
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
  }, [currentTheme.isDark, language, showLineNumbers, showFoldGutter, placeholder, readOnly])

  // Update content when value changes externally
  useEffect(() => {
    const view = viewRef.current
    if (!view) return
    
    const currentValue = view.state.doc.toString()
    if (currentValue !== value) {
      view.dispatch({
        changes: {
          from: 0,
          to: currentValue.length,
          insert: value,
        },
      })
    }
  }, [value])

  return (
    <div 
      ref={containerRef} 
      className={cn(
        "overflow-auto rounded-lg border bg-muted/30 h-full",
        className
      )}
    />
  )
}
