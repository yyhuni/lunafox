"use client"

import * as React from "react"
import { cn } from "@/lib/utils"
import type { ShuffleRef } from "@/components/Shuffle"
import type { LoginStep, TerminalLine, TerminalLoginTranslations } from "@/components/ui/terminal-login-types"
import {
  TerminalDesktopPanel,
  TerminalLoginBanner,
  TerminalLoginHeader,
  TerminalMobilePanel,
} from "@/components/ui/terminal-login-sections"

interface TerminalLoginProps {
  onLogin: (username: string, password: string) => Promise<void>
  authDone?: boolean
  isPending?: boolean
  className?: string
  translations: TerminalLoginTranslations
}

export function TerminalLogin({
  onLogin,
  authDone = false,
  isPending = false,
  className,
  translations: t,
}: TerminalLoginProps) {
  const [step, setStep] = React.useState<LoginStep>("username")
  const [username, setUsername] = React.useState("")
  const [password, setPassword] = React.useState("")
  const [lines, setLines] = React.useState<TerminalLine[]>([])
  const [cursorPosition, setCursorPosition] = React.useState(0)
  const [isFocused, setIsFocused] = React.useState(false)
  const inputRef = React.useRef<HTMLInputElement>(null)
  const containerRef = React.useRef<HTMLDivElement>(null)
  const shuffleRef = React.useRef<ShuffleRef>(null)
  const hasTriggeredShuffleRef = React.useRef(false)

  // Focus input on mount and when step changes
  React.useEffect(() => {
    inputRef.current?.focus()
  }, [step])

  // Focus input when pointer-down on non-interactive blank areas.
  const handleContainerPointerDown = (event: React.PointerEvent<HTMLDivElement>) => {
    const target = event.target as HTMLElement
    if (target.closest("button, a, input, textarea, select, [role='button']")) {
      return
    }
    // Prevent default focus behavior from blurring the hidden input after we focus it.
    event.preventDefault()
    inputRef.current?.focus()
  }

  const addLine = (line: TerminalLine) => {
    setLines((prev) => [...prev, line])
  }

  const getCurrentValue = () => {
    if (step === "username") return username
    if (step === "password") return password
    return ""
  }

  const setCurrentValue = (value: string) => {
    if (step === "username") {
      setUsername(value)
      setCursorPosition(value.length)
    } else if (step === "password") {
      setPassword(value)
      setCursorPosition(value.length)
    }
  }

  const handleKeyDown = async (e: React.KeyboardEvent<HTMLInputElement>) => {
    const value = getCurrentValue()

    // Ctrl+C - Cancel/Clear current input
    if (e.ctrlKey && e.key === "c") {
      e.preventDefault()
      if (step === "username" || step === "password") {
        addLine({ text: `^C`, type: "warning" })
        setCurrentValue("")
        setCursorPosition(0)
      }
      return
    }

    // Ctrl+U - Clear line (delete from cursor to start)
    if (e.ctrlKey && e.key === "u") {
      e.preventDefault()
      setCurrentValue("")
      setCursorPosition(0)
      return
    }

    // Ctrl+A - Move cursor to start
    if (e.ctrlKey && e.key === "a") {
      e.preventDefault()
      setCursorPosition(0)
      if (inputRef.current) {
        inputRef.current.setSelectionRange(0, 0)
      }
      return
    }

    // Ctrl+E - Move cursor to end
    if (e.ctrlKey && e.key === "e") {
      e.preventDefault()
      setCursorPosition(value.length)
      if (inputRef.current) {
        inputRef.current.setSelectionRange(value.length, value.length)
      }
      return
    }

    // Ctrl+W - Delete word before cursor
    if (e.ctrlKey && e.key === "w") {
      e.preventDefault()
      const beforeCursor = value.slice(0, cursorPosition)
      const afterCursor = value.slice(cursorPosition)
      const lastSpace = beforeCursor.trimEnd().lastIndexOf(" ")
      const newBefore = lastSpace === -1 ? "" : beforeCursor.slice(0, lastSpace + 1)
      setCurrentValue(newBefore + afterCursor)
      setCursorPosition(newBefore.length)
      return
    }

    // Tab - Move to next field (username -> password)
    if (e.key === "Tab" && step === "username") {
      e.preventDefault()
      if (!username.trim()) return
      addLine({ text: `> ${t.usernamePrompt}: `, type: "prompt" })
      addLine({ text: username, type: "input" })
      setStep("password")
      setCursorPosition(0)
      return
    }

    // Enter - Submit
    if (e.key === "Enter") {
      if (step === "username") {
        if (!username.trim()) return
        addLine({ text: `> ${t.usernamePrompt}: `, type: "prompt" })
        addLine({ text: username, type: "input" })
        setStep("password")
        setCursorPosition(0)
      } else if (step === "password") {
        if (!password.trim()) return
        addLine({ text: `> ${t.passwordPrompt}: `, type: "prompt" })
        addLine({ text: "*".repeat(password.length), type: "input" })
        addLine({ text: "", type: "info" })
        setStep("authenticating")

        try {
          await onLogin(username, password)
          addLine({ text: `> ${t.accessGranted}`, type: "success" })
          addLine({ text: `> ${t.welcomeMessage}`, type: "success" })
          // Keep showing the authenticating progress bar until navigation happens.
        } catch {
          addLine({ text: `> ${t.authFailed}`, type: "error" })
          addLine({ text: `> ${t.invalidCredentials}`, type: "error" })
          addLine({ text: "", type: "info" })
          setStep("error")
          setTimeout(() => {
            setUsername("")
            setPassword("")
            setLines([])
            setCursorPosition(0)
            setStep("username")
          }, 2000)
        }
      }
      return
    }
  }

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value
    setCurrentValue(value)
    setCursorPosition(e.target.selectionStart || value.length)

    // Trigger shuffle animation on first username input
    if (step === "username" && value.length === 1 && !hasTriggeredShuffleRef.current) {
      hasTriggeredShuffleRef.current = true
      shuffleRef.current?.play()
    }
  }

  const handleSelect = (e: React.SyntheticEvent<HTMLInputElement>) => {
    const target = e.target as HTMLInputElement
    setCursorPosition(target.selectionStart || 0)
  }

  const isInputDisabled = step === "authenticating" || step === "success" || isPending

  const getCurrentPrompt = () => {
    if (step === "username") return `> ${t.usernamePrompt}: `
    if (step === "password") return `> ${t.passwordPrompt}: `
    return "> "
  }

  const getDisplayValue = () => {
    if (step === "username") return username
    if (step === "password") return "*".repeat(password.length)
    return ""
  }

  // Render cursor at position
  const renderInputWithCursor = () => {
    const displayValue = getDisplayValue()
    const before = displayValue.slice(0, cursorPosition)
    const after = displayValue.slice(cursorPosition)
    const cursorChar = after[0] || ""

    if (!isFocused) {
      return <span className="text-foreground">{displayValue}</span>
    }

    return (
      <>
        <span className="text-foreground">{before}</span>
        <span className="animate-blink inline-block min-w-[0.6em] bg-primary text-primary-foreground">
          {cursorChar || "\u00A0"}
        </span>
        <span className="text-foreground">{after.slice(1)}</span>
      </>
    )
  }

  const handleMobileSubmit = async () => {
    if (!username.trim() || !password.trim()) return
    setStep("authenticating")
    try {
      await onLogin(username, password)
    } catch {
      setStep("error")
      setTimeout(() => {
        setUsername("")
        setPassword("")
        setStep("username")
      }, 2000)
    }
  }

  return (
    <div
      ref={containerRef}
      onPointerDown={handleContainerPointerDown}
      className={cn(
        "border-border bg-card/70 text-foreground backdrop-blur-sm z-0 w-full max-w-xl rounded-xl border cursor-text",
        className
      )}
    >
      <TerminalLoginHeader title={t.title} />

      {/* Terminal content */}
      <div className="p-4 font-mono text-sm min-h-[280px]">
        <TerminalLoginBanner subtitle={t.subtitle} shuffleRef={shuffleRef} />

        {/* ========== Mobile Form ========== */}
        <TerminalMobilePanel
          step={step}
          username={username}
          password={password}
          isInputDisabled={isInputDisabled}
          authDone={authDone}
          onUsernameChange={setUsername}
          onPasswordChange={setPassword}
          onSubmit={handleMobileSubmit}
          t={t}
        />

        {/* ========== Desktop Terminal ========== */}
        <TerminalDesktopPanel
          step={step}
          lines={lines}
          t={t}
          prompt={getCurrentPrompt()}
          renderInput={renderInputWithCursor}
          inputRef={inputRef}
          authDone={authDone}
          inputProps={{
            value: getCurrentValue(),
            onChange: handleInputChange,
            onKeyDown: handleKeyDown,
            onSelect: handleSelect,
            onFocus: () => setIsFocused(true),
            onBlur: () => setIsFocused(false),
            disabled: isInputDisabled,
            type: step === "password" ? "password" : "text",
            autoComplete: step === "username" ? "username" : "current-password",
          }}
        />
      </div>
    </div>
  )
}
