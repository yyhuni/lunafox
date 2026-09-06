"use client"

import { type CSSProperties, type ReactNode } from "react"
import { Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts"
import type { ValueType } from "recharts/types/component/DefaultTooltipContent"

type OverviewAreaChartDatum = Record<string, number | string>

type OverviewAreaChartMargin = {
  top?: number
  right?: number
  bottom?: number
  left?: number
}

type OverviewAreaChartPadding = {
  left?: number
  right?: number
}

type OverviewAreaChartDot = false | {
  r?: number
  stroke?: string
  strokeWidth?: number
  fill?: string
}

type OverviewAreaChartTooltipFormatter = (value: number) => [ReactNode, ReactNode]
type OverviewAreaChartTooltipLabelFormatter = (label: number | string) => ReactNode

export type OverviewAreaChartProps<TData extends OverviewAreaChartDatum> = {
  data: TData[]
  dataKey: keyof TData & string
  dot?: OverviewAreaChartDot
  gradientId: string
  margin: OverviewAreaChartMargin
  tooltipFormatter: OverviewAreaChartTooltipFormatter
  tooltipLabelFormatter: OverviewAreaChartTooltipLabelFormatter
  xAxisDataKey: keyof TData & string
  activeDotRadius?: number
  color?: string
  contentStyle?: CSSProperties
  isAnimationActive?: boolean
  minHeight?: number
  showGrid?: boolean
  showXAxis?: boolean
  showYAxis?: boolean
  xAxisInterval?: number
  xAxisMinTickGap?: number
  xAxisPadding?: OverviewAreaChartPadding
  xAxisTickFontSize?: number
  xAxisTickFormatter?: (value: number | string) => string
  xAxisTickMargin?: number
  yAxisDomain?: [number, number]
  yAxisTickFormatter?: (value: number) => string
  yAxisTickMargin?: number
  yAxisTicks?: number[]
  yAxisWidth?: number
}

const DEFAULT_X_AXIS_TICK_FONT_SIZE = 11
const DEFAULT_X_AXIS_TICK_MARGIN = 6
const DEFAULT_Y_AXIS_TICK_MARGIN = 4
const DEFAULT_Y_AXIS_WIDTH = 44

function toFiniteNumber(value: ValueType) {
  if (Array.isArray(value)) {
    return toFiniteNumber(value[0] ?? 0)
  }

  const parsed = Number(value)

  return Number.isFinite(parsed) ? parsed : 0
}

function buildAxisTick(fontSize: number) {
  return {
    fill: "var(--muted-foreground)",
    fontSize,
  }
}

export function OverviewAreaChart<TData extends OverviewAreaChartDatum>({
  data,
  dataKey,
  dot = false,
  gradientId,
  margin,
  tooltipFormatter,
  tooltipLabelFormatter,
  xAxisDataKey,
  activeDotRadius = 4,
  color = "var(--primary)",
  contentStyle,
  isAnimationActive = false,
  minHeight,
  showGrid = true,
  showXAxis = true,
  showYAxis = true,
  xAxisInterval = 1,
  xAxisMinTickGap,
  xAxisPadding = { left: 0, right: 0 },
  xAxisTickFontSize = DEFAULT_X_AXIS_TICK_FONT_SIZE,
  xAxisTickFormatter,
  xAxisTickMargin = DEFAULT_X_AXIS_TICK_MARGIN,
  yAxisDomain,
  yAxisTickFormatter,
  yAxisTickMargin = DEFAULT_Y_AXIS_TICK_MARGIN,
  yAxisTicks,
  yAxisWidth = DEFAULT_Y_AXIS_WIDTH,
}: OverviewAreaChartProps<TData>) {
  const activeDot = {
    fill: "var(--card)",
    r: activeDotRadius,
    stroke: color,
    strokeWidth: 2,
  }

  return (
    <ResponsiveContainer width="100%" height="100%" minHeight={minHeight}>
      <AreaChart data={data} margin={margin}>
        <defs>
          <linearGradient id={gradientId} x1="0" x2="0" y1="0" y2="1">
            <stop offset="0%" stopColor={color} stopOpacity={0.2} />
            <stop offset="72%" stopColor={color} stopOpacity={0.08} />
            <stop offset="100%" stopColor={color} stopOpacity={0.02} />
          </linearGradient>
        </defs>
        {showGrid ? <CartesianGrid vertical={false} stroke="var(--border)" strokeOpacity={0.45} /> : null}
        {showXAxis ? (
          <XAxis
            dataKey={xAxisDataKey}
            axisLine={false}
            interval={xAxisInterval}
            minTickGap={xAxisMinTickGap}
            padding={xAxisPadding}
            tick={buildAxisTick(xAxisTickFontSize)}
            tickFormatter={xAxisTickFormatter}
            tickLine={false}
            tickMargin={xAxisTickMargin}
          />
        ) : null}
        {showYAxis ? (
          <YAxis
            axisLine={false}
            domain={yAxisDomain}
            tick={buildAxisTick(DEFAULT_X_AXIS_TICK_FONT_SIZE)}
            tickFormatter={yAxisTickFormatter ? (value) => yAxisTickFormatter(Number(value)) : undefined}
            tickLine={false}
            tickMargin={yAxisTickMargin}
            ticks={yAxisTicks}
            width={yAxisWidth}
          />
        ) : null}
        <Tooltip
          contentStyle={contentStyle}
          cursor={{ stroke: "var(--border)" }}
          formatter={(value: ValueType) => tooltipFormatter(toFiniteNumber(value))}
          labelFormatter={(label: number | string) => tooltipLabelFormatter(label)}
        />
        <Area
          activeDot={activeDot}
          dataKey={dataKey}
          dot={dot}
          fill={`url(#${gradientId})`}
          isAnimationActive={isAnimationActive}
          stroke={color}
          strokeWidth={2}
          type="monotone"
        />
      </AreaChart>
    </ResponsiveContainer>
  )
}
