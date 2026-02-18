"use client"

import { useEffect, useState } from "react"
import { IconBrandGithub } from "@/components/icons"
import { cn } from "@/lib/utils"

interface GithubStarButtonProps {
  className?: string
}

export function GithubStarButton({ className }: GithubStarButtonProps) {
  const [stars, setStars] = useState<number | null>(null)

  useEffect(() => {
    // Simple caching mechanism to avoid rate limit caused by frequent refresh during development
    const cached = sessionStorage.getItem("github-stars")
    const cachedTime = sessionStorage.getItem("github-stars-time")
    const now = Date.now()

    if (cached && cachedTime && now - parseInt(cachedTime) < 5 * 60 * 1000) {
      setStars(parseInt(cached))
      return
    }

    fetch("https://api.github.com/repos/yyhuni/xingrin")
      .then((res) => {
        if (!res.ok) throw new Error("Failed to fetch")
        return res.json()
      })
      .then((data) => {
        if (typeof data.stargazers_count === "number") {
          setStars(data.stargazers_count)
          sessionStorage.setItem("github-stars", data.stargazers_count.toString())
          sessionStorage.setItem("github-stars-time", now.toString())
        }
      })
      .catch(() => {
        // Fail silently without affecting user experience
      })
  }, [])

  return (
    <a
      href="https://github.com/yyhuni/xingrin"
      target="_blank"
      rel="noopener noreferrer"
      className={cn(
        "group relative hidden sm:flex items-center h-9 bg-muted/50 text-foreground border border-border hover:border-[#00C8FF] transition-[background-color,border-color,color,box-shadow] overflow-hidden min-w-[120px]",
        className
      )}
    >
      {/* Diagonal background element */}
      <div className="absolute inset-0 bg-background transform -skew-x-12 translate-x-[-150%] group-hover:translate-x-[-20%] transition-transform duration-500 opacity-80" />
      
      <div className="relative flex items-center px-3 z-10 w-full justify-between">
        <div className="flex items-center gap-2">
          <IconBrandGithub className="size-4 text-muted-foreground group-hover:text-[#00C8FF] transition-colors" />
          <span className="font-sans font-bold uppercase text-[10px] tracking-widest text-muted-foreground group-hover:text-foreground transition-colors">Star</span>
        </div>
        {stars !== null && (
          <span className="font-mono text-[10px] text-muted-foreground group-hover:text-[#00C8FF] transition-colors">
            [{stars.toLocaleString()}]
          </span>
        )}
      </div>
      
      {/* Corner accents */}
      <div className="absolute top-0 right-0 w-1 h-1 bg-[#00C8FF] opacity-0 group-hover:opacity-100 transition-opacity" />
      <div className="absolute bottom-0 left-0 w-1 h-1 bg-[#00C8FF] opacity-0 group-hover:opacity-100 transition-opacity" />
    </a>
  )
}
