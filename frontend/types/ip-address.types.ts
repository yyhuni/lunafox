export interface Port {
  number: number
  serviceName: string
  description: string
  isUncommon: boolean
}

export interface IPAddress {
  ip: string  // IP address (unique identifier)
  hosts: string[]  // Associated hostname list
  ports: number[]  // Associated port list
  createdAt: string  // First creation time in the aggregated IP group
}

export interface GetIPAddressesParams {
  pageSize?: number
  pageToken?: string
  filter?: string
  orderBy?: string
}

export interface GetIPAddressesResponse {
  results: IPAddress[]
  total: number
  totalSize?: number
  page: number
  pageSize: number
  totalPages: number
  nextPageToken?: string
}

export interface FilterOption {
  value: string
  label: string
  count?: number
}

export interface FilterOptionsResponse {
  results: FilterOption[]
}
