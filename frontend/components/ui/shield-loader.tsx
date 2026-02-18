"use client"

import * as React from "react"
import { useTranslations } from "next-intl"

import { cn } from "@/lib/utils"

type ShieldLoaderProps = {
  size?: "sm" | "md" | "lg"
  className?: string
}

const sizeMap = {
  sm: "140px",
  md: "180px",
  lg: "220px",
}

function ShieldLoader({ size = "md", className }: ShieldLoaderProps) {
  const t = useTranslations("common.ui")

  return (
    <div
      role="status"
      aria-label={t("loading")}
      className={cn("shield-loader", className)}
      style={{ "--shield-size": sizeMap[size] } as React.CSSProperties}
    >
      <div className="shield-loader__energy" />
      <div className="shield-loader__ring shield-loader__ring--s4" />
      <div className="shield-loader__ring shield-loader__ring--s3" />
      <div className="shield-loader__ring shield-loader__ring--s2" />
      <div className="shield-loader__ring shield-loader__ring--s1" />
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img
        className="shield-loader__logo"
        src="/images/icon-256.png"
        alt=""
        aria-hidden="true"
        width={256}
        height={256}
      />
    </div>
  )
}

export { ShieldLoader }
