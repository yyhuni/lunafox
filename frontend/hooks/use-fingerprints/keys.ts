import type { FingerprintListParams } from "@/hooks/_shared/fingerprint-hooks"

export const fingerprintKeys = {
  all: ["fingerprints"] as const,
  stats: () => [...fingerprintKeys.all, "stats"] as const,
  ehole: {
    all: () => [...fingerprintKeys.all, "ehole"] as const,
    list: (params: FingerprintListParams) =>
      [...fingerprintKeys.ehole.all(), "list", params] as const,
    detail: (id: number) => [...fingerprintKeys.ehole.all(), "detail", id] as const,
  },
  goby: {
    all: () => [...fingerprintKeys.all, "goby"] as const,
    list: (params: FingerprintListParams) =>
      [...fingerprintKeys.goby.all(), "list", params] as const,
    detail: (id: number) => [...fingerprintKeys.goby.all(), "detail", id] as const,
  },
  wappalyzer: {
    all: () => [...fingerprintKeys.all, "wappalyzer"] as const,
    list: (params: FingerprintListParams) =>
      [...fingerprintKeys.wappalyzer.all(), "list", params] as const,
    detail: (id: number) => [...fingerprintKeys.wappalyzer.all(), "detail", id] as const,
  },
  fingers: {
    all: () => [...fingerprintKeys.all, "fingers"] as const,
    list: (params: FingerprintListParams) =>
      [...fingerprintKeys.fingers.all(), "list", params] as const,
    detail: (id: number) => [...fingerprintKeys.fingers.all(), "detail", id] as const,
  },
  fingerprinthub: {
    all: () => [...fingerprintKeys.all, "fingerprinthub"] as const,
    list: (params: FingerprintListParams) =>
      [...fingerprintKeys.fingerprinthub.all(), "list", params] as const,
    detail: (id: number) =>
      [...fingerprintKeys.fingerprinthub.all(), "detail", id] as const,
  },
  arl: {
    all: () => [...fingerprintKeys.all, "arl"] as const,
    list: (params: FingerprintListParams) =>
      [...fingerprintKeys.arl.all(), "list", params] as const,
    detail: (id: number) => [...fingerprintKeys.arl.all(), "detail", id] as const,
  },
}
