"use client"

import { useEffect } from "react"
import { motion, useReducedMotion, useSpring, useTransform } from "framer-motion"

type OverviewNumberTickerProps = {
  value: number
  locale: string
}

export function OverviewNumberTicker({ value, locale }: OverviewNumberTickerProps) {
  const prefersReducedMotion = useReducedMotion()
  const spring = useSpring(0, { mass: 0.8, stiffness: 75, damping: 15 })
  const display = useTransform(spring, (current) => Math.round(current).toLocaleString(locale))

  useEffect(() => {
    if (prefersReducedMotion) {
      return
    }

    spring.set(value)
  }, [prefersReducedMotion, spring, value])

  if (prefersReducedMotion) {
    return <span>{value.toLocaleString(locale)}</span>
  }

  return <motion.span>{display}</motion.span>
}
