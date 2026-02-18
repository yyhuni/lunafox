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
  createdAt: string  // First creation time
}

export interface GetIPAddressesParams {
  page?: number
  pageSize?: number
  filter?: string  // Smart filter syntax string
}

export interface GetIPAddressesResponse {
  results: IPAddress[]
  total: number
  page: number
  pageSize: number
  totalPages: number
}
