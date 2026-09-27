import { describe, expect, it } from "vitest"
import {
  assertProductionRuntimeFlags,
  getRequiredVercelBackendOrigin,
} from "@/lib/production-runtime-config"

describe("production runtime flags", () => {
  it("rejects mock mode in production", () => {
    expect(() => assertProductionRuntimeFlags({ nodeEnv: "production", useMock: true })).toThrow(
      "NEXT_PUBLIC_USE_MOCK",
    )
  })

  it("rejects no-auth mode in production", () => {
    expect(() => assertProductionRuntimeFlags({ nodeEnv: "production", skipAuth: true })).toThrow(
      "NEXT_PUBLIC_SKIP_AUTH",
    )
  })

  it("preserves explicit development switches", () => {
    expect(() =>
      assertProductionRuntimeFlags({ nodeEnv: "development", useMock: true, skipAuth: true }),
    ).not.toThrow()
  })

  it("requires a root HTTPS backend origin for Vercel rewrites", () => {
    expect(getRequiredVercelBackendOrigin("https://api.example.test")).toBe("https://api.example.test")
    expect(() => getRequiredVercelBackendOrigin()).toThrow("BACKEND_URL")
    expect(() => getRequiredVercelBackendOrigin("http://api.example.test")).toThrow("HTTPS origin")
    expect(() => getRequiredVercelBackendOrigin("https://api.example.test/v1")).toThrow("HTTPS origin")
    expect(() => getRequiredVercelBackendOrigin("https://api.example.test", "https://public.example.test")).toThrow(
      "NEXT_PUBLIC_BACKEND_URL",
    )
  })
})
