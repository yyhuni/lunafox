"use client"

import type { ComponentProps } from "react"
import { useMemo } from "react"
import { motion, useReducedMotion } from "motion/react"
import DottedMap from "dotted-map"
import { ComposableMap, Geography, Geographies, Marker, ZoomableGroup } from "react-simple-maps"
import countries from "world-atlas/countries-110m.json"

import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { cn } from "@/lib/utils"

const WORLD_MAP_REGION = {
  lat: { min: -60, max: 85 },
  lng: { min: -180, max: 180 },
} as const

const WORLD_MAP_WIDTH = 248
const WORLD_MAP_HEIGHT = 100
const WORLD_MAP_SCALE = WORLD_MAP_WIDTH / (2 * Math.PI)
const WORLD_MAP_CENTER: [number, number] = [0, 12.5]
const WORLD_MAP_TRANSLATE_EXTENT: [[number, number], [number, number]] = [[0, 0], [WORLD_MAP_WIDTH, WORLD_MAP_HEIGHT]]
const WORLD_MAP_INITIAL_ZOOM = 1.05
const WORLD_MAP_MAX_ZOOM = 5
const WORLD_MAP_DOT_RADIUS = 0.22
const WORLD_MAP_CONNECTION_INSET = 2
const WORLD_MAP_ACTIVE_DISPATCH_STROKE_WIDTH = 0.52
const WORLD_MAP_ACTIVE_DISPATCH_HALF_LENGTH = WORLD_MAP_ACTIVE_DISPATCH_STROKE_WIDTH / 2
const WORLD_MAP_BEACON_COUNT = 2
const WORLD_MAP_BEACON_DURATION_SECONDS = 3.6
const WORLD_MAP_MARKER_HIT_RADIUS = {
  agent: 5.5,
  server: 7,
} as const

const COUNTRY_STYLE = {
  default: {
    fill: "transparent",
    stroke: "var(--border)",
    strokeOpacity: 0.24,
    strokeWidth: 0.28,
    outline: "none",
    vectorEffect: "non-scaling-stroke",
  },
  hover: {
    fill: "var(--primary)",
    fillOpacity: 0.08,
    stroke: "var(--primary)",
    strokeOpacity: 0.72,
    strokeWidth: 0.72,
    outline: "none",
    vectorEffect: "non-scaling-stroke",
  },
  pressed: {
    fill: "var(--primary)",
    fillOpacity: 0.12,
    stroke: "var(--primary)",
    strokeOpacity: 0.82,
    strokeWidth: 0.78,
    outline: "none",
    vectorEffect: "non-scaling-stroke",
  },
} satisfies ComponentProps<typeof Geography>["style"]

const ANTARCTICA_NAMES = new Set(["antarctica", "fr. s. antarctic lands"])

export type WorldMapPoint = {
  lat: number
  lng: number
  label?: string
}

export type WorldMapConnection = {
  id?: string
  start: WorldMapPoint
  end: WorldMapPoint
  curveDirection?: 1 | -1
  active?: boolean
}

export type WorldMapMarker = {
  id: string
  lat: number
  lng: number
  color: string
  kind?: "agent" | "server"
  locationState?: "current" | "expired"
  label: string
  description?: string
  details?: string[]
  ariaLabel?: string
}

type WorldMapProps = {
  dots?: WorldMapConnection[]
  markers?: WorldMapMarker[]
  lineColor?: string
  ariaLabel?: string
  className?: string
}

type ProjectedPoint = {
  x: number
  y: number
}

type ProjectedConnection = {
  id: string
  start: ProjectedPoint
  end: ProjectedPoint
  curveDirection: 1 | -1
  active: boolean
}

export const WORLD_MAP_SURFACE_CLASS = "relative aspect-[248/100] w-full overflow-hidden font-sans"

function toProjectedPoint(map: DottedMap, point: WorldMapPoint): ProjectedPoint | null {
  if (!Number.isFinite(point.lat) || !Number.isFinite(point.lng)) {
    return null
  }

  const projected = map.getPin({ lat: point.lat, lng: point.lng })
  return projected ? { x: projected.x, y: projected.y } : null
}

function clampConnectionCoordinate(value: number, maximum: number) {
  return Math.min(Math.max(value, WORLD_MAP_CONNECTION_INSET), maximum - WORLD_MAP_CONNECTION_INSET)
}

export function createBoundedConnectionPath(
  start: ProjectedPoint,
  end: ProjectedPoint,
  width: number,
  height: number,
  curveDirection: 1 | -1,
) {
  const boundedStart = {
    x: clampConnectionCoordinate(start.x, width),
    y: clampConnectionCoordinate(start.y, height),
  }
  const boundedEnd = {
    x: clampConnectionCoordinate(end.x, width),
    y: clampConnectionCoordinate(end.y, height),
  }
  const deltaX = boundedEnd.x - boundedStart.x
  const deltaY = boundedEnd.y - boundedStart.y
  const distance = Math.max(Math.hypot(deltaX, deltaY), 1)
  const midX = (boundedStart.x + boundedEnd.x) / 2
  const midY = (boundedStart.y + boundedEnd.y) / 2
  const curvature = Math.min(Math.max(distance * 0.12, 3), height * 0.2)
  const normalX = -deltaY / distance
  const normalY = deltaX / distance
  const controlX = clampConnectionCoordinate(midX + normalX * curvature * curveDirection, width)
  const controlY = clampConnectionCoordinate(midY + normalY * curvature * curveDirection, height)
  return `M ${boundedStart.x} ${boundedStart.y} Q ${controlX} ${controlY} ${boundedEnd.x} ${boundedEnd.y}`
}

function createDottedMapPath(map: DottedMap) {
  return map.getPoints().map(({ x, y }) => {
    const diameter = WORLD_MAP_DOT_RADIUS * 2
    return `M ${x - WORLD_MAP_DOT_RADIUS} ${y} a ${WORLD_MAP_DOT_RADIUS} ${WORLD_MAP_DOT_RADIUS} 0 1 0 ${diameter} 0 a ${WORLD_MAP_DOT_RADIUS} ${WORLD_MAP_DOT_RADIUS} 0 1 0 -${diameter} 0`
  }).join("")
}

function isAntarctica(geography: { properties?: { name?: string } }) {
  return ANTARCTICA_NAMES.has(geography.properties?.name?.toLowerCase() ?? "")
}

export default function WorldMap({
  dots = [],
  markers = [],
  lineColor = "var(--primary)",
  ariaLabel = "World map",
  className,
}: WorldMapProps) {
  const prefersReducedMotion = useReducedMotion()
  const map = useMemo(
    () => new DottedMap({
      height: WORLD_MAP_HEIGHT,
      grid: "diagonal",
      projection: { name: "equirectangular" },
      region: WORLD_MAP_REGION,
    }),
    [],
  )
  const svgMap = useMemo(
    () => createDottedMapPath(map),
    [map],
  )
  const projectedConnections = useMemo(
    () => dots.flatMap((dot, index): ProjectedConnection[] => {
      const start = toProjectedPoint(map, dot.start)
      const end = toProjectedPoint(map, dot.end)
      return start && end ? [{
        id: dot.id ?? `connection-${index}`,
        start,
        end,
        curveDirection: dot.curveDirection ?? (index % 2 === 0 ? 1 : -1),
        active: dot.active ?? false,
      }] : []
    }),
    [dots, map],
  )
  return (
    <div
      role="group"
      aria-label={ariaLabel}
      className={cn(WORLD_MAP_SURFACE_CLASS, "isolate select-none", className)}
      data-slot="world-map"
    >
      <TooltipProvider delay={120}>
        <ComposableMap
          className="absolute inset-0 h-full w-full cursor-grab touch-none active:cursor-grabbing"
          height={WORLD_MAP_HEIGHT}
          projection="geoEquirectangular"
          projectionConfig={{ center: WORLD_MAP_CENTER, scale: WORLD_MAP_SCALE }}
          width={WORLD_MAP_WIDTH}
        >
          <ZoomableGroup
            center={WORLD_MAP_CENTER}
            maxZoom={WORLD_MAP_MAX_ZOOM}
            minZoom={1}
            translateExtent={WORLD_MAP_TRANSLATE_EXTENT}
            zoom={WORLD_MAP_INITIAL_ZOOM}
          >
            <path
              aria-hidden="true"
              className="pointer-events-none"
              d={svgMap}
              fill="var(--muted-foreground)"
              fillOpacity="0.52"
            />

            <g aria-hidden="true" className="pointer-events-none">
              {projectedConnections.map((connection, index) => {
                const path = createBoundedConnectionPath(
                  connection.start,
                  connection.end,
                  WORLD_MAP_WIDTH,
                  map.image.height,
                  connection.curveDirection,
                )
                return (
                  <g key={connection.id}>
                    <motion.path
                      d={path}
                      fill="none"
                      pathLength={1}
                      stroke={lineColor}
                      strokeDasharray="2 2"
                      strokeLinecap="round"
                      strokeOpacity={connection.active ? 0.34 : 0.18}
                      strokeWidth="0.32"
                      initial={prefersReducedMotion ? false : { pathLength: 0 }}
                      animate={prefersReducedMotion ? undefined : { pathLength: 1 }}
                      transition={{ duration: 0.7, delay: index * 0.08, ease: "easeOut" }}
                    />
                    {connection.active ? (
                      <line
                        aria-hidden="true"
                        x1={prefersReducedMotion ? connection.start.x - WORLD_MAP_ACTIVE_DISPATCH_HALF_LENGTH : -WORLD_MAP_ACTIVE_DISPATCH_HALF_LENGTH}
                        x2={prefersReducedMotion ? connection.start.x + WORLD_MAP_ACTIVE_DISPATCH_HALF_LENGTH : WORLD_MAP_ACTIVE_DISPATCH_HALF_LENGTH}
                        y1={prefersReducedMotion ? connection.start.y : 0}
                        y2={prefersReducedMotion ? connection.start.y : 0}
                        stroke={lineColor}
                        strokeLinecap="butt"
                        strokeOpacity={prefersReducedMotion ? 0.58 : 0.92}
                        strokeWidth={WORLD_MAP_ACTIVE_DISPATCH_STROKE_WIDTH}
                      >
                        {/* Native path motion keeps the dispatch dash at a fixed physical length without React-side coordinate updates. */}
                        {prefersReducedMotion ? null : (
                          <animateMotion
                            begin={`${index * 0.14}s`}
                            calcMode="paced"
                            dur="2.8s"
                            path={path}
                            repeatCount="indefinite"
                            rotate="auto"
                          />
                        )}
                      </line>
                    ) : null}
                  </g>
                )
              })}
            </g>

            <Geographies geography={countries}>
              {({ geographies }) => geographies
                .filter((geography) => !isAntarctica(geography))
                .map((geography) => {
                  const countryName = geography.properties?.name ?? "Unknown country"
                  return (
                    <Geography
                      key={geography.rsmKey}
                      aria-label={countryName}
                      data-world-map-country={countryName}
                      geography={geography}
                      style={COUNTRY_STYLE}
                    />
                  )
                })}
            </Geographies>

            {markers.map((marker, index) => (
              <Marker key={marker.id} coordinates={[marker.lng, marker.lat]}>
                <Tooltip>
                  <TooltipTrigger
                    render={(
                      <g
                        aria-label={marker.ariaLabel ?? marker.label}
                        className="group cursor-pointer outline-none"
                        data-world-map-marker={marker.id}
                        data-world-map-location-state={marker.locationState}
                        role="button"
                        tabIndex={0}
                      />
                    )}
                  >
                    <circle
                      aria-hidden="true"
                      fill="transparent"
                      pointerEvents="all"
                      r={marker.kind === "server" ? WORLD_MAP_MARKER_HIT_RADIUS.server : WORLD_MAP_MARKER_HIT_RADIUS.agent}
                    />
                    {marker.locationState === "expired" ? (
                      <circle
                        aria-hidden="true"
                        fill="none"
                        r={marker.kind === "server" ? 3.6 : 2.6}
                        stroke={marker.color}
                        strokeDasharray="1.2 1.2"
                        strokeOpacity="0.82"
                        strokeWidth="0.48"
                      />
                    ) : null}
                    <circle
                      aria-hidden="true"
                      className="pointer-events-none opacity-0 group-focus-visible:opacity-100"
                      fill="none"
                      r={marker.kind === "server" ? 4.6 : 3.3}
                      stroke={marker.kind === "server" ? marker.color : "var(--ring)"}
                      strokeWidth="0.65"
                    />
                    {marker.locationState === "current" && !prefersReducedMotion ? (
                      // Offset rings keep the live-location beacon continuous instead of restarting as a flash.
                      Array.from({ length: WORLD_MAP_BEACON_COUNT }, (_, beaconIndex) => (
                        <motion.circle
                          key={beaconIndex}
                          aria-hidden="true"
                          animate={{
                            opacity: [0, 0.24, 0.1, 0],
                            r: marker.kind === "server" ? [1.65, 4.4] : [1, 3.1],
                          }}
                          fill={marker.kind === "server" ? marker.color : "none"}
                          initial={{ opacity: 0, r: 1 }}
                          stroke={marker.color}
                          strokeWidth="0.48"
                          transition={{
                            duration: WORLD_MAP_BEACON_DURATION_SECONDS,
                            delay: index * 0.08 + beaconIndex * (WORLD_MAP_BEACON_DURATION_SECONDS / WORLD_MAP_BEACON_COUNT),
                            ease: "easeOut",
                            repeat: Infinity,
                          }}
                        />
                      ))
                    ) : null}
                    <circle
                      aria-hidden="true"
                      fill={marker.color}
                      r={marker.kind === "server" ? 1.65 : 1}
                      style={{ filter: `drop-shadow(0 0 1.5px ${marker.color})` }}
                    />
                  </TooltipTrigger>
                  <TooltipContent side="top" sideOffset={6}>
                    <div className="flex flex-col gap-0.5">
                      <span>{marker.label}</span>
                      {marker.description ? <span className="text-muted-foreground">{marker.description}</span> : null}
                      {marker.details?.map((detail) => (
                        <span key={detail} className="text-muted-foreground">{detail}</span>
                      ))}
                    </div>
                  </TooltipContent>
                </Tooltip>
              </Marker>
            ))}
          </ZoomableGroup>
        </ComposableMap>
      </TooltipProvider>
    </div>
  )
}
