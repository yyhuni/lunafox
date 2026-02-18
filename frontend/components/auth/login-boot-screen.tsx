"use client"

import * as React from "react"
import Image from "next/image"

import { cn } from "@/lib/utils"

const STATUS_MESSAGES = [
  "Initializing security modules…",
  "Loading vulnerability database…",
  "Connecting to scan engine…",
  "Preparing templates…",
  "Almost ready…",
]

export function LoginBootScreen({ className }: { className?: string; success?: boolean }) {
  const [statusIndex, setStatusIndex] = React.useState(0)
  const [statusVisible, setStatusVisible] = React.useState(true)
  const logoSrc = "/images/icon-256.png"

  // Status text rotation
  React.useEffect(() => {
    let timeoutId: ReturnType<typeof setTimeout> | null = null
    const interval = setInterval(() => {
      setStatusVisible(false)
      if (timeoutId) {
        clearTimeout(timeoutId)
      }
      timeoutId = setTimeout(() => {
        setStatusIndex((prev) => (prev + 1) % STATUS_MESSAGES.length)
        setStatusVisible(true)
      }, 200)
    }, 3000)

    return () => {
      clearInterval(interval)
      if (timeoutId) {
        clearTimeout(timeoutId)
      }
    }
  }, [])

  return (
    <div className={cn("relative flex min-h-svh flex-col bg-[#0a0a0f] overflow-hidden", className)}>
      {/* Animated Gradient Background */}
      <div className="fixed inset-0 z-0 overflow-hidden">
        <div 
          className="absolute top-[-50%] left-[-50%] w-[200%] h-[200%] opacity-30 animate-blob"
          style={{
            background: "conic-gradient(from 0deg at 50% 50%, #0a0a0f 0deg, #3f3f46 60deg, #0a0a0f 120deg, #52525b 180deg, #0a0a0f 240deg, #3f3f46 300deg, #0a0a0f 360deg)",
            filter: "blur(80px)",
          }}
        />
        <div className="absolute inset-0 bg-[#0a0a0f]/80" /> {/* Overlay to darken */}
      </div>

      {/* Background grid */}
      <div 
        className="fixed inset-0 z-0 opacity-40"
        style={{
          backgroundImage: `
            linear-gradient(rgba(255, 255, 255, 0.1) 1px, transparent 1px),
            linear-gradient(90deg, rgba(255, 255, 255, 0.1) 1px, transparent 1px)
          `,
          backgroundSize: "50px 50px",
          maskImage: "radial-gradient(circle at center, black, transparent 80%)",
        }}
      />

      {/* Main content */}
      <div className="relative z-10 flex-1 flex items-center justify-center">
        <div className="text-center">
          {/* Logo container */}
          <div className="relative w-[240px] h-[240px] mx-auto mb-10">
            {/* Spinning rings */}
            <div className="logo-spinner" />
            {/* Logo image */}
            <Image
              src={logoSrc}
              alt="LunaFox Logo"
              width={160}
              height={160}
              className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 z-10"
              unoptimized
              priority
            />
          </div>

          {/* Title */}
          <h1 className="text-[32px] font-bold tracking-tight mb-2">
            <span className="bg-gradient-to-br from-[#d4d4d8] to-[#f4f4f5] bg-clip-text text-transparent">Luna</span>
            <span className="bg-gradient-to-br from-[#a1a1aa] to-[#e4e4e7] bg-clip-text text-transparent">Fox</span>
          </h1>

          {/* Loading status */}
          <div className="flex items-center justify-center gap-3 mt-6">
            <div className="status-spinner" />
            <span 
              className={cn(
                "text-sm text-gray-500 font-medium transition-opacity duration-200",
                statusVisible ? "opacity-100" : "opacity-0"
              )}
            >
              {STATUS_MESSAGES[statusIndex]}
            </span>
          </div>

          {/* Progress bar */}
          <div className="w-60 h-1 bg-[rgba(255,255,255,0.1)] rounded mx-auto mt-6 overflow-hidden">
            <div className="progress-fill h-full rounded" />
          </div>

          {/* Dots */}
          <div className="flex justify-center gap-1.5 mt-8">
            {[0, 1, 2, 3, 4].map((i) => (
              <div 
                key={i} 
                className="dot-pulse"
                style={{ animationDelay: `${i * 0.2}s` }}
              />
            ))}
          </div>
        </div>
      </div>

      {/* Styles */}
      <style jsx>{`
        .logo-spinner {
          position: absolute;
          top: 0;
          left: 0;
          width: 100%;
          height: 100%;
          border-radius: 50%;
          border: 4px solid transparent;
          border-top-color: rgba(255, 255, 255, 0.8);
          animation: spin 1s linear infinite;
        }
        
        .logo-spinner::before {
          content: '';
          position: absolute;
          top: -4px;
          left: -4px;
          right: -4px;
          bottom: -4px;
          border-radius: 50%;
          border: 4px solid transparent;
          border-top-color: rgba(200, 200, 200, 0.6);
          animation: spin 1.5s linear infinite reverse;
        }
        
        .logo-spinner::after {
          content: '';
          position: absolute;
          top: 8px;
          left: 8px;
          right: 8px;
          bottom: 8px;
          border-radius: 50%;
          border: 2px solid transparent;
          border-bottom-color: rgba(255, 255, 255, 0.3);
          animation: spin 2s linear infinite;
        }
        
        .status-spinner {
          width: 16px;
          height: 16px;
          border: 2px solid rgba(255, 255, 255, 0.2);
          border-top-color: rgba(255, 255, 255, 0.8);
          border-radius: 50%;
          animation: spin 0.8s linear infinite;
        }
        
        .progress-fill {
          background: linear-gradient(90deg, #a1a1aa, #f4f4f5);
          animation: progress 3s ease-in-out infinite;
        }
        
        .dot-pulse {
          width: 6px;
          height: 6px;
          background: rgba(255, 255, 255, 0.3);
          border-radius: 50%;
          animation: dot-pulse 1.5s ease-in-out infinite;
        }
        
        @keyframes spin {
          to { transform: rotate(360deg); }
        }
        
        @keyframes progress {
          0% { width: 0%; }
          50% { width: 70%; }
          100% { width: 100%; }
        }
        
        @keyframes dot-pulse {
          0%, 100% {
            background: rgba(255, 255, 255, 0.3);
            transform: scale(1);
          }
          50% {
            background: rgba(255, 255, 255, 0.8);
            transform: scale(1.3);
          }
        }

        @keyframes blob {
          0% { transform: translate(0, 0) rotate(0deg); }
          33% { transform: translate(2%, 2%) rotate(120deg); }
          66% { transform: translate(-2%, 2%) rotate(240deg); }
          100% { transform: translate(0, 0) rotate(360deg); }
        }
      `}</style>
    </div>
  )
}
