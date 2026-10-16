import { describe, expect, it } from "vitest"
import {
  assertProductionRuntimeFlags,
  assertVercelDemoHasNoBackend,
  getRequiredVercelBackendOrigin,
} from "@/lib/production-runtime-config"

describe("production runtime flags", () => {
  it("rejects mock mode in production", () => {
    expect(() =>
      assertProductionRuntimeFlags({ nodeEnv: "production", demoSite: false, useMock: true }),
    ).toThrow("NEXT_PUBLIC_USE_MOCK")
  })

  it("rejects no-auth mode in production", () => {
    expect(() =>
      assertProductionRuntimeFlags({ nodeEnv: "production", demoSite: false, skipAuth: true }),
    ).toThrow("NEXT_PUBLIC_SKIP_AUTH")
  })

  it("preserves explicit development switches", () => {
    expect(() =>
      assertProductionRuntimeFlags({ nodeEnv: "development", useMock: true, skipAuth: true }),
    ).not.toThrow()
  })

  it("allows a complete Vercel demo build", () => {
    expect(() =>
      assertProductionRuntimeFlags({
        nodeEnv: "production",
        isVercel: true,
        demoSite: true,
        useMock: true,
        skipAuth: true,
      }),
    ).not.toThrow()
    expect(() => assertVercelDemoHasNoBackend(undefined, undefined)).not.toThrow()
  })

  it("rejects a demo build outside Vercel", () => {
    expect(() =>
      assertProductionRuntimeFlags({
        nodeEnv: "production",
        isVercel: false,
        demoSite: true,
        useMock: true,
        skipAuth: true,
      }),
    ).toThrow("DEMO_SITE")
  })

  it("rejects a Vercel demo build that omits either public flag", () => {
    expect(() =>
      assertProductionRuntimeFlags({
        nodeEnv: "production",
        isVercel: true,
        demoSite: true,
        useMock: false,
        skipAuth: true,
      }),
    ).toThrow("NEXT_PUBLIC_USE_MOCK")
    expect(() =>
      assertProductionRuntimeFlags({
        nodeEnv: "production",
        isVercel: true,
        demoSite: true,
        useMock: true,
        skipAuth: false,
      }),
    ).toThrow("NEXT_PUBLIC_SKIP_AUTH")
  })

  it("rejects a Vercel demo build that configures a backend", () => {
    expect(() => assertVercelDemoHasNoBackend("https://api.example.test", undefined)).toThrow("BACKEND_URL")
    expect(() => assertVercelDemoHasNoBackend(undefined, "https://public.example.test")).toThrow(
      "NEXT_PUBLIC_BACKEND_URL",
    )
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
