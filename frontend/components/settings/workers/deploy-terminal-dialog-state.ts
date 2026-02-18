import { useCallback, useEffect, useRef, useState } from "react"
import type { Terminal } from "@xterm/xterm"
import type { FitAddon } from "@xterm/addon-fit"
import type { WorkerNode } from "@/types/worker.types"

const FALLBACK_WS_URL = "ws://localhost:8080"

const getWsBaseUrl = () => {
  if (typeof window === "undefined") return FALLBACK_WS_URL

  if (process.env.NEXT_PUBLIC_WS_URL) {
    return process.env.NEXT_PUBLIC_WS_URL
  }

  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"
  const host = window.location.host
  return `${protocol}//${host}`
}

type UseDeployTerminalDialogStateProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  worker: WorkerNode | null
  onDeployComplete?: () => void
  tTerminal: (key: string) => string
}

export function useDeployTerminalDialogState({
  open,
  onOpenChange,
  worker,
  onDeployComplete,
  tTerminal,
}: UseDeployTerminalDialogStateProps) {
  const [isConnected, setIsConnected] = useState(false)
  const [, setError] = useState<string | null>(null)
  const [localStatus, setLocalStatus] = useState<string | null>(null)
  const [uninstallDialogOpen, setUninstallDialogOpen] = useState(false)

  const terminalRef = useRef<HTMLDivElement>(null)
  const terminalInstanceRef = useRef<Terminal | null>(null)
  const fitAddonRef = useRef<FitAddon | null>(null)
  const wsRef = useRef<WebSocket | null>(null)

  const currentStatus = localStatus || worker?.status || null

  useEffect(() => {
    const fitAddon = fitAddonRef.current
    if (!fitAddon) return

    const handleResize = () => fitAddon.fit()
    window.addEventListener("resize", handleResize)

    return () => {
      window.removeEventListener("resize", handleResize)
    }
  }, [open, worker])

  const connectWs = useCallback(() => {
    if (!worker || !terminalInstanceRef.current) return

    const terminal = terminalInstanceRef.current
    if (wsRef.current) {
      wsRef.current.close()
    }

    const ws = new WebSocket(`${getWsBaseUrl()}/ws/workers/${worker.id}/deploy/`)
    ws.binaryType = "arraybuffer"
    wsRef.current = ws

    ws.onopen = () => {
      terminal.writeln(`\x1b[32m✓ ${tTerminal("wsConnected")}\x1b[0m`)
    }

    ws.onmessage = (event) => {
      if (event.data instanceof ArrayBuffer) {
        const decoder = new TextDecoder()
        terminal.write(decoder.decode(event.data))
      } else {
        try {
          const data = JSON.parse(event.data)
          if (data.type === "connected") {
            setIsConnected(true)
            setError(null)
            terminal.onData((payload: string) => {
              if (ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({ type: "input", data: payload }))
              }
            })
            ws.send(JSON.stringify({
              type: "resize",
              cols: terminal.cols,
              rows: terminal.rows,
            }))
          } else if (data.type === "error") {
            terminal.writeln(`\x1b[31m✗ ${data.message}\x1b[0m`)
            setError(data.message)
          } else if (data.type === "status") {
            setLocalStatus(data.status)
            onDeployComplete?.()
          }
        } catch {
          // ignore parse errors
        }
      }
    }

    ws.onclose = () => {
      terminal.writeln("")
      terminal.writeln(`\x1b[33m${tTerminal("connectionClosed")}\x1b[0m`)
      setIsConnected(false)
    }

    ws.onerror = () => {
      terminal.writeln(`\x1b[31m✗ ${tTerminal("wsConnectionFailed")}\x1b[0m`)
      setError(tTerminal("connectionFailed"))
    }
  }, [worker, onDeployComplete, tTerminal])

  const initTerminal = useCallback(async () => {
    if (!terminalRef.current || terminalInstanceRef.current) return

    const { Terminal } = await import("@xterm/xterm")
    const { FitAddon } = await import("@xterm/addon-fit")
    const { WebLinksAddon } = await import("@xterm/addon-web-links")

    const terminal = new Terminal({
      cursorBlink: true,
      fontSize: 12,
      fontFamily: "Menlo, Monaco, \"Courier New\", monospace",
      theme: {
        background: "#1a1b26",
        foreground: "#a9b1d6",
        cursor: "#c0caf5",
        black: "#32344a",
        red: "#f7768e",
        green: "#9ece6a",
        yellow: "#e0af68",
        blue: "#7aa2f7",
        magenta: "#ad8ee6",
        cyan: "#449dab",
        white: "#787c99",
      },
    })

    const fitAddon = new FitAddon()
    terminal.loadAddon(fitAddon)
    terminal.loadAddon(new WebLinksAddon())

    terminal.open(terminalRef.current)
    fitAddon.fit()

    terminalInstanceRef.current = terminal
    fitAddonRef.current = fitAddon

    terminal.writeln(`\x1b[90m${tTerminal("connecting")}\x1b[0m`)
    connectWs()
  }, [connectWs, tTerminal])

  useEffect(() => {
    if (open && worker) {
      const timer = setTimeout(initTerminal, 100)
      return () => clearTimeout(timer)
    }
  }, [open, worker, initTerminal])

  useEffect(() => {
    if (!isConnected || !wsRef.current || !terminalInstanceRef.current) return

    const terminal = terminalInstanceRef.current
    const handleResize = () => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(JSON.stringify({
          type: "resize",
          cols: terminal.cols,
          rows: terminal.rows,
        }))
      }
    }

    const resizeDisposable = terminal.onResize?.(handleResize)
    return () => {
      resizeDisposable?.dispose?.()
    }
  }, [isConnected])

  const handleClose = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }
    if (terminalInstanceRef.current) {
      terminalInstanceRef.current.dispose()
      terminalInstanceRef.current = null
    }
    fitAddonRef.current = null
    setIsConnected(false)
    setError(null)
    setLocalStatus(null)
    onDeployComplete?.()
    onOpenChange(false)
  }, [onDeployComplete, onOpenChange])

  const handleDeploy = useCallback(() => {
    if (!wsRef.current || !isConnected) return
    setLocalStatus("deploying")
    onDeployComplete?.()
    wsRef.current.send(JSON.stringify({ type: "deploy" }))
  }, [isConnected, onDeployComplete])

  const handleAttach = useCallback(() => {
    if (!wsRef.current || !isConnected) return
    wsRef.current.send(JSON.stringify({ type: "attach" }))
  }, [isConnected])

  const handleUninstallClick = useCallback(() => {
    if (!wsRef.current || !isConnected) return
    setUninstallDialogOpen(true)
  }, [isConnected])

  const handleUninstallConfirm = useCallback(() => {
    if (!wsRef.current || !isConnected) return
    setUninstallDialogOpen(false)
    wsRef.current.send(JSON.stringify({ type: "uninstall" }))
  }, [isConnected])

  return {
    terminalRef,
    isConnected,
    currentStatus,
    uninstallDialogOpen,
    setUninstallDialogOpen,
    handleClose,
    handleDeploy,
    handleAttach,
    handleUninstallClick,
    handleUninstallConfirm,
    connectWs,
  }
}
