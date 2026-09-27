const LEGACY_MOCK_WORKER_PATH = "/mockServiceWorker.js"

type ServiceWorkerLike = {
  scriptURL: string
  postMessage: (message: unknown) => void
}
type ServiceWorkerRegistrationLike = {
  active: ServiceWorkerLike | null
  waiting: ServiceWorkerLike | null
  installing: ServiceWorkerLike | null
  unregister: () => Promise<boolean>
}
type ServiceWorkerContainerLike = {
  getRegistrations: () => Promise<readonly ServiceWorkerRegistrationLike[]>
}

let cleanupPromise: Promise<void> | null = null

function resolveServiceWorkerContainer(): ServiceWorkerContainerLike | null {
  if (typeof navigator === "undefined" || !navigator.serviceWorker) {
    return null
  }

  return navigator.serviceWorker
}

function isLegacyMockWorker(worker: ServiceWorkerLike): boolean {
  const scriptUrl = worker.scriptURL

  try {
    return new URL(scriptUrl, window.location.origin).pathname === LEGACY_MOCK_WORKER_PATH
  } catch {
    return false
  }
}

function getLegacyMockWorkers(registration: ServiceWorkerRegistrationLike): ServiceWorkerLike[] {
  return [registration.active, registration.waiting, registration.installing]
    .filter((worker): worker is ServiceWorkerLike => worker !== null)
    .filter(isLegacyMockWorker)
}

async function cleanupRegistration(registration: ServiceWorkerRegistrationLike): Promise<void> {
  for (const worker of getLegacyMockWorkers(registration)) {
    try {
      worker.postMessage("CLIENT_CLOSED")
    } catch {
      // An already-stopped worker can reject the close signal; unregister still applies.
    }
  }

  try {
    await registration.unregister()
  } catch {
    // Cleanup is best effort. Real API traffic must remain available if the browser rejects it.
  }
}

async function cleanupRegistrations(container: ServiceWorkerContainerLike | null): Promise<void> {
  if (!container) return

  let registrations: readonly ServiceWorkerRegistrationLike[]
  try {
    registrations = await container.getRegistrations()
  } catch {
    return
  }

  await Promise.all(
    registrations
      .filter((registration) => getLegacyMockWorkers(registration).length > 0)
      .map((registration) => cleanupRegistration(registration)),
  )
}

/**
 * Remove only the old MSW worker left by a previous mock-mode deployment.
 * An injected container is supported for deterministic contract tests.
 */
export function cleanupLegacyMockWorker(container?: ServiceWorkerContainerLike | null): Promise<void> {
  if (container !== undefined) {
    return cleanupRegistrations(container)
  }

  if (!cleanupPromise) {
    cleanupPromise = cleanupRegistrations(resolveServiceWorkerContainer())
  }

  return cleanupPromise
}
