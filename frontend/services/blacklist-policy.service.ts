import { api } from "@/lib/api-client"
import {
  blacklistPolicyName,
  targetBlacklistPolicyName,
} from "@/lib/resource-name"
import type {
  BlacklistPolicy,
  UpdateBlacklistPolicyInput,
} from "@/types/blacklist-policy.types"

type BlacklistPolicyResponse = {
  name: string
  patterns: string[]
  etag: string
  updateTime: string
}

function toBlacklistPolicy(response: BlacklistPolicyResponse): BlacklistPolicy {
  return {
    name: response.name,
    patterns: [...response.patterns],
    etag: response.etag,
    updateTime: response.updateTime,
  }
}

async function patchBlacklistPolicy(
  path: string,
  name: string,
  input: UpdateBlacklistPolicyInput
): Promise<BlacklistPolicy> {
  const response = await api.patch<BlacklistPolicyResponse>(
    path,
    {
      name,
      patterns: input.patterns,
      etag: input.etag,
    },
    {
      params: { updateMask: "patterns" },
    }
  )
  return toBlacklistPolicy(response.data)
}

export async function getGlobalBlacklistPolicy(): Promise<BlacklistPolicy> {
  const response = await api.get<BlacklistPolicyResponse>("/blacklistPolicy")
  return toBlacklistPolicy(response.data)
}

export function updateGlobalBlacklistPolicy(
  input: UpdateBlacklistPolicyInput
): Promise<BlacklistPolicy> {
  return patchBlacklistPolicy("/blacklistPolicy", blacklistPolicyName(), input)
}

export async function getTargetBlacklistPolicy(targetId: number): Promise<BlacklistPolicy> {
  const name = targetBlacklistPolicyName(targetId)
  const response = await api.get<BlacklistPolicyResponse>(`/${name}`)
  return toBlacklistPolicy(response.data)
}

export function updateTargetBlacklistPolicy(
  targetId: number,
  input: UpdateBlacklistPolicyInput
): Promise<BlacklistPolicy> {
  const name = targetBlacklistPolicyName(targetId)
  return patchBlacklistPolicy(`/${name}`, name, input)
}
