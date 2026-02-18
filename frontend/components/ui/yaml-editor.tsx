"use client"

import React, { useState, useCallback, useEffect } from "react"
import * as yaml from "js-yaml"
import { AlertCircle } from "@/components/icons"
import { useTranslations } from "next-intl"
import { cn } from "@/lib/utils"
import { CodeEditor } from "@/components/ui/code-editor"

interface YamlEditorProps {
  value: string
  onChange: (value: string) => void
  placeholder?: string
  disabled?: boolean
  className?: string
  onValidationChange?: (isValid: boolean, error?: { message: string; line?: number; column?: number }) => void
}

/**
 * YAML Editor component with CodeMirror
 * Provides syntax highlighting and validation
 */
export function YamlEditor({
  value,
  onChange,
  placeholder,
  disabled = false,
  className,
  onValidationChange,
}: YamlEditorProps) {
  const t = useTranslations("common.yamlEditor")
  const [yamlError, setYamlError] = useState<{ message: string; line?: number; column?: number } | null>(null)

  // Check for duplicate keys in YAML content
  const checkDuplicateKeys = useCallback((content: string): { key: string; line: number } | null => {
    const lines = content.split('\n')
    const keyStack: { indent: number; keys: Set<string> }[] = [{ indent: -1, keys: new Set() }]
    
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i]
      // Skip empty lines and comments
      if (!line.trim() || line.trim().startsWith('#')) continue
      
      // Match top-level keys (no leading whitespace, ends with colon)
      const topLevelMatch = line.match(/^([a-zA-Z_][a-zA-Z0-9_-]*):\s*(?:#.*)?$/)
      if (topLevelMatch) {
        const key = topLevelMatch[1]
        const currentLevel = keyStack[0]
        
        if (currentLevel.keys.has(key)) {
          return { key, line: i + 1 }
        }
        currentLevel.keys.add(key)
      }
    }
    
    return null
  }, [])

  // Validate YAML syntax
  const validateYaml = useCallback((content: string) => {
    if (!content.trim()) {
      setYamlError(null)
      onValidationChange?.(true)
      return true
    }

    // First check for duplicate keys
    const duplicateKey = checkDuplicateKeys(content)
    if (duplicateKey) {
      const errorInfo = {
        message: t("duplicateKey", { key: duplicateKey.key }),
        line: duplicateKey.line,
        column: 1,
      }
      setYamlError(errorInfo)
      onValidationChange?.(false, errorInfo)
      return false
    }

    try {
      yaml.load(content)
      setYamlError(null)
      onValidationChange?.(true)
      return true
    } catch (error) {
      const yamlException = error as yaml.YAMLException
      const errorInfo = {
        message: yamlException.message,
        line: yamlException.mark?.line ? yamlException.mark.line + 1 : undefined,
        column: yamlException.mark?.column ? yamlException.mark.column + 1 : undefined,
      }
      setYamlError(errorInfo)
      onValidationChange?.(false, errorInfo)
      return false
    }
  }, [onValidationChange, checkDuplicateKeys, t])

  // Handle editor content change
  const handleEditorChange = useCallback((newValue: string) => {
    onChange(newValue)
    validateYaml(newValue)
  }, [onChange, validateYaml])

  // Validate initial content
  useEffect(() => {
    validateYaml(value)
  }, [validateYaml, value])

  return (
    <div className={cn("flex flex-col h-full", className)}>
      {/* CodeMirror Editor */}
      <CodeEditor
        value={value}
        onChange={handleEditorChange}
        language="yaml"
        placeholder={placeholder}
        readOnly={disabled}
        className={cn("flex-1", yamlError ? 'border-destructive' : '')}
        showLineNumbers={false}
        showFoldGutter
      />

      {/* Error message display */}
      {yamlError && (
        <div className="flex items-start gap-2 p-2 bg-destructive/10 border-t border-destructive/20">
          <AlertCircle className="h-3.5 w-3.5 text-destructive mt-0.5 flex-shrink-0" />
          <div className="flex-1 text-xs">
            <p className="font-medium text-destructive">
              {yamlError.line && yamlError.column
                ? t("errorLocation", { line: yamlError.line, column: yamlError.column })
                : t("syntaxError")}
            </p>
            <p className="text-muted-foreground truncate">{yamlError.message}</p>
          </div>
        </div>
      )}
    </div>
  )
}
