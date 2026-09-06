import { mockHandlers } from "./handlers"

let mockWorkerStartPromise: Promise<ServiceWorkerRegistration | undefined> | null = null

export function startMockWorker() {
  if (typeof window === "undefined") {
    return Promise.resolve(undefined)
  }

  if (!mockWorkerStartPromise) {
    mockWorkerStartPromise = import("msw/browser").then(({ setupWorker }) => {
      const mockWorker = setupWorker(...mockHandlers)

      return mockWorker.start({
        onUnhandledRequest: "bypass",
        serviceWorker: {
          url: "/mockServiceWorker.js",
        },
      })
    })
  }

  return mockWorkerStartPromise
}
