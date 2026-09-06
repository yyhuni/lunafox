type StorageKind = "local" | "session"

function resolveStorage(kind: StorageKind): Storage | null {
  if (typeof window === "undefined") return null

  try {
    return kind === "session" ? window.sessionStorage : window.localStorage
  } catch {
    return null
  }
}

export function isStorageAvailable(kind: StorageKind = "local"): boolean {
  return resolveStorage(kind) !== null
}

export function readStorageValue(key: string): string | null {
  try {
    return resolveStorage("local")?.getItem(key) ?? null
  } catch {
    return null
  }
}

export function writeStorageValue(key: string, value: string): boolean {
  try {
    const storage = resolveStorage("local")
    if (!storage) return false
    storage.setItem(key, value)
    return true
  } catch {
    return false
  }
}

export function removeStorageValue(key: string): boolean {
  try {
    const storage = resolveStorage("local")
    if (!storage) return false
    storage.removeItem(key)
    return true
  } catch {
    return false
  }
}

export function readJsonStorage<T>(key: string, fallback: T): T {
  const raw = readStorageValue(key)
  if (!raw) return fallback

  try {
    return JSON.parse(raw) as T
  } catch {
    return fallback
  }
}

export function writeJsonStorage(key: string, value: unknown): boolean {
  try {
    return writeStorageValue(key, JSON.stringify(value))
  } catch {
    return false
  }
}

export function readSessionValue(key: string): string | null {
  try {
    return resolveStorage("session")?.getItem(key) ?? null
  } catch {
    return null
  }
}

export function writeSessionValue(key: string, value: string): boolean {
  try {
    const storage = resolveStorage("session")
    if (!storage) return false
    storage.setItem(key, value)
    return true
  } catch {
    return false
  }
}

export function removeSessionValue(key: string): boolean {
  try {
    const storage = resolveStorage("session")
    if (!storage) return false
    storage.removeItem(key)
    return true
  } catch {
    return false
  }
}
