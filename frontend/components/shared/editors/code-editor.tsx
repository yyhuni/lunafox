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
import { codeMirrorDarkTheme, codeMirrorLightTheme } from "@/components/shared/editors/codemirror-theme"
import { useColorTheme } from "@/hooks/use-color-theme"
import { cn } from "@/lib/utils"

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
      currentTheme.isDark ? codeMirrorDarkTheme : codeMirrorLightTheme,
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
