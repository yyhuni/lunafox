export interface BlacklistPolicy {
  name: string
  patterns: string[]
  etag: string
  updateTime: string
}

export interface UpdateBlacklistPolicyInput {
  patterns: string[]
  etag: string
}
