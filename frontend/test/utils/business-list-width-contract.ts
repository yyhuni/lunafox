import { existsSync, readFileSync } from "node:fs"
import path from "node:path"

export const BUSINESS_LIST_DESKTOP_WIDTH_BUDGET_PX = 1542

export interface ColumnWidthContract {
  size: number | null
  minSize: number | null
  maxSize: number | null
}

const sourcePathRegistry = new Map<string, string>()

export function readComponentSource(relativePath: string) {
  const source = readFileSync(path.resolve(process.cwd(), relativePath), "utf8")
  sourcePathRegistry.set(source, relativePath)
  return source
}

export function getDefaultColumnSizeTotal(source: string) {
  return [...source.matchAll(/\bsize:\s*(\d+)/g)].reduce((total, match) => {
    return total + Number(match[1])
  }, 0)
}

export function getColumnWidthContract(source: string, accessorKey: string): ColumnWidthContract {
  const marker = `accessorKey: "${accessorKey}"`
  const start = source.indexOf(marker)

  if (start === -1) {
    throw new Error(`Could not find column accessorKey "${accessorKey}" in source`)
  }

  const objectStart = findColumnObjectStart(source, start)
  const objectEnd = findMatchingBraceIndex(source, objectStart)
  const block = source.slice(objectStart, objectEnd + 1)

  return {
    size: readNumericProperty(block, "size", source),
    minSize: readNumericProperty(block, "minSize", source),
    maxSize: readNumericProperty(block, "maxSize", source),
  }
}

export function getColumnResizeHeadroom(contract: ColumnWidthContract) {
  const size = contract.size ?? 0
  const maxSize = contract.maxSize ?? size
  return maxSize - size
}

function readNumericProperty(block: string, propertyName: string, source: string) {
  const match = block.match(new RegExp(`\\b${propertyName}:\\s*(\\d+)`))
  if (match) {
    return Number(match[1])
  }

  const referenceMatch = block.match(
    new RegExp(`\\b${propertyName}:\\s*([A-Za-z_$][\\w$.]*)`)
  )

  if (!referenceMatch) {
    return null
  }

  return resolveNumericReference(referenceMatch[1], source)
}

function resolveNumericReference(reference: string, source: string) {
  const sourceRelativePath = sourcePathRegistry.get(source)

  if (!sourceRelativePath) {
    return null
  }

  const [bindingName, nestedKey, nestedProperty] = reference.split(".")

  if (!bindingName || !nestedKey || !nestedProperty) {
    return null
  }

  const importPathMatch = source.match(
    new RegExp(`import\\s*\\{[^}]*\\b${bindingName}\\b[^}]*\\}\\s*from\\s*"([^"]+)"`)
  )

  if (!importPathMatch) {
    return null
  }

  const importedRelativePath = resolveImportPath(sourceRelativePath, importPathMatch[1])
  const importedSource = readComponentSource(importedRelativePath)
  const objectStartMarker = `export const ${bindingName} = {`
  const objectStart = importedSource.indexOf(objectStartMarker)

  if (objectStart === -1) {
    return null
  }

  const braceStart = importedSource.indexOf("{", objectStart)
  const braceEnd = findMatchingBraceIndex(importedSource, braceStart)
  const objectBlock = importedSource.slice(braceStart, braceEnd + 1)
  const nestedMatch = objectBlock.match(
    new RegExp(`${nestedKey}:\\s*\\{[^}]*\\b${nestedProperty}:\\s*(\\d+)`, "s")
  )

  return nestedMatch ? Number(nestedMatch[1]) : null
}

function resolveImportPath(sourceRelativePath: string, importPath: string) {
  const baseDir = path.posix.dirname(sourceRelativePath)
  const candidates = [
    path.posix.normalize(path.posix.join(baseDir, `${importPath}.ts`)),
    path.posix.normalize(path.posix.join(baseDir, `${importPath}.tsx`)),
    path.posix.normalize(path.posix.join(baseDir, importPath, "index.ts")),
    path.posix.normalize(path.posix.join(baseDir, importPath, "index.tsx")),
  ]

  for (const candidate of candidates) {
    if (existsSync(path.resolve(process.cwd(), candidate))) {
      return candidate
    }
  }

  throw new Error(`Could not resolve import path "${importPath}" from "${sourceRelativePath}"`)
}

function findColumnObjectStart(source: string, accessorKeyIndex: number) {
  for (let index = accessorKeyIndex; index >= 0; index -= 1) {
    if (source[index] === "{") {
      return index
    }
  }

  throw new Error(`Could not find object start for accessorKey at index ${accessorKeyIndex}`)
}

function findMatchingBraceIndex(source: string, objectStart: number) {
  let depth = 0
  let inSingleQuote = false
  let inDoubleQuote = false
  let inTemplate = false
  let inLineComment = false
  let inBlockComment = false
  let escaped = false

  for (let index = objectStart; index < source.length; index += 1) {
    const char = source[index]
    const nextChar = source[index + 1]

    if (inLineComment) {
      if (char === "\n") {
        inLineComment = false
      }
      continue
    }

    if (inBlockComment) {
      if (char === "*" && nextChar === "/") {
        inBlockComment = false
        index += 1
      }
      continue
    }

    if (inSingleQuote) {
      if (!escaped && char === "'") {
        inSingleQuote = false
      }
      escaped = !escaped && char === "\\"
      continue
    }

    if (inDoubleQuote) {
      if (!escaped && char === "\"") {
        inDoubleQuote = false
      }
      escaped = !escaped && char === "\\"
      continue
    }

    if (inTemplate) {
      if (!escaped && char === "`") {
        inTemplate = false
      }
      escaped = !escaped && char === "\\"
      continue
    }

    escaped = false

    if (char === "/" && nextChar === "/") {
      inLineComment = true
      index += 1
      continue
    }

    if (char === "/" && nextChar === "*") {
      inBlockComment = true
      index += 1
      continue
    }

    if (char === "'") {
      inSingleQuote = true
      continue
    }

    if (char === "\"") {
      inDoubleQuote = true
      continue
    }

    if (char === "`") {
      inTemplate = true
      continue
    }

    if (char === "{") {
      depth += 1
      continue
    }

    if (char === "}") {
      depth -= 1
      if (depth === 0) {
        return index
      }
    }
  }

  throw new Error(`Could not find matching brace for object starting at index ${objectStart}`)
}
