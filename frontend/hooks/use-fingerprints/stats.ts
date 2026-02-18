import { useQuery } from "@tanstack/react-query"
import { FingerprintService } from "@/services/fingerprint.service"
import { fingerprintKeys } from "./keys"

export function useFingerprintStats() {
  return useQuery({
    queryKey: fingerprintKeys.stats(),
    queryFn: () => FingerprintService.getStats(),
  })
}
