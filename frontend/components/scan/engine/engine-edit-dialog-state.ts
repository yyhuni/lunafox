import React from "react"
import * as yaml from "js-yaml"
import { toast } from "sonner"
import type { ScanEngine } from "@/types/engine.types"

export type EngineEditYamlError = {
  message: string
  line?: number
  column?: number
} | null

type UseEngineEditDialogStateProps = {
  engine: ScanEngine | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onSave?: (engineId: number, yamlContent: string) => Promise<void>
  t: (key: string, params?: Record<string, string | number | Date>) => string
  tToast: (key: string, params?: Record<string, string | number | Date>) => string
}

const generateSampleYaml = (engine: ScanEngine) => {
  return `# Engine name: ${engine.name}

# ==================== Subdomain Discovery ====================
subdomain_discovery:
  tools:
    subfinder:
      enabled: true
      timeout: 600      # 10 minutes (required)


# ==================== Port Scan ====================
port_scan:
  tools:
    naabu_active:
      enabled: true
      timeout: auto     # Auto calculate
      threads: 5
      top-ports: 100
      rate: 10
      
    naabu_passive:
      enabled: true
      timeout: auto


# ==================== Site Scan ====================
site_scan:
  tools:
    httpx:
      enabled: true
      timeout: auto         # Auto calculate
      # screenshot: true    # Enable site screenshot (requires Chromium)


# ==================== Directory Scan ====================
directory_scan:
  tools:
    ffuf:
      enabled: true
      timeout: auto                            # Auto calculate timeout
      wordlist: ~/Desktop/dirsearch_dicc.txt   # Wordlist file path (required)
      delay: 0.1-2.0
      threads: 10
      request_timeout: 10
      match_codes: 200,201,301,302,401,403


# ==================== URL Fetch ====================
url_fetch:
  tools:
    waymore:
      enabled: true
      timeout: auto
    
    katana:
      enabled: true
      timeout: auto
      depth: 5
      threads: 10
      rate-limit: 30
      random-delay: 1
      retry: 2
      request-timeout: 12
    
    uro:
      enabled: true
      timeout: auto
    
    httpx:
      enabled: true
      timeout: auto
`
}

export function useEngineEditDialogState({
  engine,
  open,
  onOpenChange,
  onSave,
  t,
  tToast,
}: UseEngineEditDialogStateProps) {
  const [yamlContent, setYamlContent] = React.useState("")
  const [isSubmitting, setIsSubmitting] = React.useState(false)
  const [hasChanges, setHasChanges] = React.useState(false)
  const [yamlError, setYamlError] = React.useState<EngineEditYamlError>(null)

  React.useEffect(() => {
    if (engine && open) {
      const content = engine.configuration || generateSampleYaml(engine)
      setYamlContent(content)
      setHasChanges(false)
      setYamlError(null)
    }
  }, [engine, open])

  const validateYaml = React.useCallback((content: string) => {
    if (!content.trim()) {
      setYamlError(null)
      return true
    }

    try {
      yaml.load(content)
      setYamlError(null)
      return true
    } catch (error) {
      const yamlError = error as yaml.YAMLException
      setYamlError({
        message: yamlError.message,
        line: yamlError.mark?.line ? yamlError.mark.line + 1 : undefined,
        column: yamlError.mark?.column ? yamlError.mark.column + 1 : undefined,
      })
      return false
    }
  }, [])

  const handleEditorChange = React.useCallback((value: string) => {
    setYamlContent(value)
    setHasChanges(true)
    validateYaml(value)
  }, [validateYaml])

  const handleSave = React.useCallback(async () => {
    if (!engine) return

    if (!yamlContent.trim()) {
      toast.error(tToast("configRequired"))
      return
    }

    if (!validateYaml(yamlContent)) {
      toast.error(tToast("yamlSyntaxError"), {
        description: yamlError?.message,
      })
      return
    }

    setIsSubmitting(true)
    try {
      if (onSave) {
        await onSave(engine.id, yamlContent)
      } else {
        await new Promise((resolve) => setTimeout(resolve, 1000))
      }

      setHasChanges(false)
      onOpenChange(false)
    } catch {
      // Error toast handled upstream
    } finally {
      setIsSubmitting(false)
    }
  }, [engine, onOpenChange, onSave, tToast, validateYaml, yamlContent, yamlError?.message])

  const handleClose = React.useCallback(() => {
    if (hasChanges) {
      const confirmed = window.confirm(t("confirmClose"))
      if (!confirmed) return
    }
    onOpenChange(false)
  }, [hasChanges, onOpenChange, t])

  return {
    yamlContent,
    isSubmitting,
    hasChanges,
    yamlError,
    handleEditorChange,
    handleSave,
    handleClose,
  }
}
