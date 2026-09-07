import fs from "node:fs/promises"
import path from "node:path"
import { spawn, spawnSync } from "node:child_process"

const STATE_FILE_NAME = "smoke-dev-server-state.json"
const LOCK_FILE_NAME = ".smoke-dev-server.lock"
const LOCK_RETRY_MS = 100
const LOCK_TIMEOUT_MS = 120_000

function nowIso() {
  return new Date().toISOString()
}

function waitMs(ms) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms)
  })
}

function listChildPids(parentPid) {
  const result = spawnSync("pgrep", ["-P", String(parentPid)], {
    encoding: "utf8",
  })
  if (result.status !== 0 || !result.stdout) {
    return []
  }
  return result.stdout
    .split(/\s+/)
    .map((value) => Number.parseInt(value, 10))
    .filter((value) => Number.isInteger(value) && value > 0)
}

function collectProcessTreePids(rootPid) {
  const root = Number.parseInt(String(rootPid), 10)
  if (!Number.isInteger(root) || root <= 0) {
    return []
  }
  const visited = new Set([root])
  const queue = [root]
  const ordered = [root]
  while (queue.length > 0) {
    const current = queue.shift()
    const children = listChildPids(current)
    for (const childPid of children) {
      if (visited.has(childPid)) {
        continue
      }
      visited.add(childPid)
      queue.push(childPid)
      ordered.push(childPid)
    }
  }
  return ordered
}

function isPidAlive(pid) {
  try {
    process.kill(pid, 0)
    return true
  } catch {
    return false
  }
}

function signalPid(pid, signal) {
  try {
    process.kill(pid, signal)
  } catch {}
}

async function cleanupProcessTree(rootPid) {
  const initialPids = collectProcessTreePids(rootPid).reverse()
  if (initialPids.length === 0) {
    return
  }
  for (const pid of initialPids) {
    if (isPidAlive(pid)) {
      signalPid(pid, "SIGTERM")
    }
  }
  await waitMs(250)
  for (const pid of initialPids) {
    if (isPidAlive(pid)) {
      signalPid(pid, "SIGKILL")
    }
  }
}

async function isBaseUrlReachable(url) {
  try {
    const controller = new AbortController()
    const timeout = setTimeout(() => controller.abort(), 1500)
    try {
      const response = await fetch(url, {
        method: "HEAD",
        redirect: "manual",
        signal: controller.signal,
      })
      return response.status > 0
    } finally {
      clearTimeout(timeout)
    }
  } catch {
    return false
  }
}

function canAutoStartForBaseUrl(url, autoStartServer) {
  if (!autoStartServer) {
    return false
  }
  try {
    const parsed = new URL(url)
    return parsed.hostname === "127.0.0.1" || parsed.hostname === "localhost" || parsed.hostname === "0.0.0.0"
  } catch {
    return false
  }
}

function startDevServer({ url, projectRoot }) {
  const parsed = new URL(url)
  const hostname = parsed.hostname === "localhost" ? "127.0.0.1" : parsed.hostname
  const port = parsed.port || "3000"
  return spawn("pnpm", ["run", "dev", "--hostname", hostname, "--port", port], {
    cwd: projectRoot,
    env: {
      ...process.env,
      NEXT_PUBLIC_SKIP_AUTH: "true",
      NEXT_PUBLIC_USE_MOCK: process.env.NEXT_PUBLIC_USE_MOCK ?? "true",
      NEXT_PUBLIC_MOCK_SCENARIO: process.env.NEXT_PUBLIC_MOCK_SCENARIO ?? "happy",
    },
    stdio: "ignore",
  })
}

function getStatePaths(projectRoot) {
  const stateDir = path.join(projectRoot, "test-plan")
  return {
    stateDir,
    statePath: path.join(stateDir, STATE_FILE_NAME),
    lockPath: path.join(stateDir, LOCK_FILE_NAME),
  }
}

async function readState(statePath) {
  try {
    const parsed = JSON.parse(await fs.readFile(statePath, "utf8"))
    const pid = Number.parseInt(String(parsed?.pid ?? ""), 10)
    return {
      pid: Number.isInteger(pid) && pid > 0 ? pid : null,
      baseUrl: typeof parsed?.baseUrl === "string" ? parsed.baseUrl : "",
      startedAt: typeof parsed?.startedAt === "string" ? parsed.startedAt : "",
      leases: Array.isArray(parsed?.leases) ? parsed.leases.filter((item) => typeof item === "string") : [],
    }
  } catch (error) {
    if (error?.code === "ENOENT") {
      return null
    }
    throw error
  }
}

async function writeState(statePath, state) {
  await fs.writeFile(statePath, `${JSON.stringify(state, null, 2)}\n`, "utf8")
}

async function removeState(statePath) {
  await fs.unlink(statePath).catch((error) => {
    if (error?.code !== "ENOENT") {
      throw error
    }
  })
}

function isUsableOwnedServer(state, baseUrl) {
  return Boolean(state?.pid && state.baseUrl === baseUrl && isPidAlive(state.pid))
}

async function acquireFileLock(lockPath) {
  const startedAt = Date.now()
  await fs.mkdir(path.dirname(lockPath), { recursive: true })

  while (Date.now() - startedAt < LOCK_TIMEOUT_MS) {
    let handle = null
    try {
      handle = await fs.open(lockPath, "wx")
      await handle.writeFile(`${JSON.stringify({ pid: process.pid, startedAt: nowIso() }, null, 2)}\n`, "utf8")
      let released = false
      return async () => {
        if (released) {
          return
        }
        released = true
        await handle.close().catch(() => {})
        await fs.unlink(lockPath).catch((error) => {
          if (error?.code !== "ENOENT") {
            throw error
          }
        })
      }
    } catch (error) {
      if (handle) {
        await handle.close().catch(() => {})
      }
      if (error?.code !== "EEXIST") {
        throw error
      }
      const stale = await readState(lockPath).catch(() => null)
      if (stale?.pid && !isPidAlive(stale.pid)) {
        await fs.unlink(lockPath).catch(() => {})
        continue
      }
      await waitMs(LOCK_RETRY_MS)
    }
  }

  throw new Error(`timed out acquiring smoke dev server lock: ${lockPath}`)
}

function addLease(state, leaseId) {
  return {
    ...state,
    leases: Array.from(new Set([...(state.leases ?? []), leaseId])),
  }
}

function removeLease(state, leaseId) {
  return {
    ...state,
    leases: (state.leases ?? []).filter((item) => item !== leaseId),
  }
}

export async function acquireSmokeDevServerLease({
  baseUrl,
  autoStartServer,
  projectRoot,
  readyTimeoutMs = 90_000,
}) {
  const leaseId = `${process.pid}:${Date.now()}:${Math.random().toString(36).slice(2)}`
  const { statePath, lockPath } = getStatePaths(projectRoot)
  const releaseLock = await acquireFileLock(lockPath)

  try {
    let state = await readState(statePath)
    if (isUsableOwnedServer(state, baseUrl)) {
      state = addLease(state, leaseId)
      await writeState(statePath, state)
      return {
        leaseId,
        baseUrl,
        pid: state.pid,
        owned: true,
      }
    }

    if (state?.pid && !isPidAlive(state.pid)) {
      await removeState(statePath)
      state = null
    }

    if (await isBaseUrlReachable(baseUrl)) {
      return {
        leaseId: null,
        baseUrl,
        pid: null,
        owned: false,
      }
    }

    if (!canAutoStartForBaseUrl(baseUrl, autoStartServer)) {
      return {
        leaseId: null,
        baseUrl,
        pid: null,
        owned: false,
      }
    }

    const processRef = startDevServer({ url: baseUrl, projectRoot })
    const pid = processRef.pid ?? null
    const deadline = Date.now() + readyTimeoutMs
    while (Date.now() < deadline) {
      if (await isBaseUrlReachable(baseUrl)) {
        const nextState = {
          pid,
          baseUrl,
          startedAt: nowIso(),
          leases: [leaseId],
        }
        await writeState(statePath, nextState)
        return {
          leaseId,
          baseUrl,
          pid,
          owned: true,
        }
      }
      await waitMs(1000)
    }
    if (pid) {
      await cleanupProcessTree(pid)
    }
    throw new Error(`dev server failed to become ready: ${baseUrl}`)
  } finally {
    await releaseLock()
  }
}

export async function releaseDevServerLease({ projectRoot, lease }) {
  if (!lease?.leaseId || !lease.owned) {
    return
  }

  const { statePath, lockPath } = getStatePaths(projectRoot)
  const releaseLock = await acquireFileLock(lockPath)
  try {
    const state = await readState(statePath)
    if (!isUsableOwnedServer(state, lease.baseUrl)) {
      await removeState(statePath)
      return
    }

    const nextState = removeLease(state, lease.leaseId)
    if (nextState.leases.length > 0) {
      await writeState(statePath, nextState)
      return
    }

    await cleanupProcessTree(state.pid)
    await removeState(statePath)
  } finally {
    await releaseLock()
  }
}
