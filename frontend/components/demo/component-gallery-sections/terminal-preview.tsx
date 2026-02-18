"use client"

import { Terminal, TypingAnimation } from "@/components/ui/terminal"

export function TerminalPreview() {
  return (
    <Terminal className="max-w-full">
      <TypingAnimation>lunafox init --mode stealth</TypingAnimation>
      <TypingAnimation delay={800}>fetching assets...</TypingAnimation>
      <TypingAnimation delay={1400}>scan running...</TypingAnimation>
    </Terminal>
  )
}
