type UnknownRecord = Record<string, unknown>

export type BarShapeProps = {
  x: number
  y: number
  width: number
  height: number
  index?: number
}

function asFiniteNumber(value: unknown): number {
  return typeof value === "number" && Number.isFinite(value) ? value : 0
}

export function sanitizeBarShapeProps(input: unknown): BarShapeProps {
  const record: UnknownRecord = typeof input === "object" && input !== null ? (input as UnknownRecord) : {}
  const indexValue = record.index
  return {
    x: asFiniteNumber(record.x),
    y: asFiniteNumber(record.y),
    width: Math.max(0, asFiniteNumber(record.width)),
    height: Math.max(0, asFiniteNumber(record.height)),
    index: typeof indexValue === "number" && Number.isFinite(indexValue) ? indexValue : undefined,
  }
}
