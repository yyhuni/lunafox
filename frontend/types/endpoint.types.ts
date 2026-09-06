// Endpoint specific data type definitions

import type { BatchCreateResponse } from './api-response.types'

export interface Endpoint {
  id: number
  url: string

  // HTTP metadata (may be null in some scenarios)
  method?: string
  statusCode: number | null
  title: string
  contentLength: number | null
  contentType?: string | null

  // Site/endpoint dimension additional info (used by both asset table and snapshot table)
  host?: string
  location?: string
  webserver?: string
  responseBody?: string
  tech?: string[]
  vhost?: boolean | null
  responseHeaders?: string
  createdAt?: string

  // Legacy domain association fields (may not exist in some APIs)
  domainId?: number
  subdomainId?: number
  domain?: string
  subdomain?: string
  updatedAt?: string
}

export interface EndpointListQueryParams {
  pageSize?: number
  pageToken?: string
  filter?: string
  orderBy?: string
}

export type GetEndpointsRequest = EndpointListQueryParams

export interface GetEndpointsResponse {
  results: Endpoint[]
  total: number
  page: number
  pageSize: number
  totalPages?: number
  totalSize?: number
  nextPageToken?: string
}

export type EndpointFilterOptionField = "statusCode" | "tech" | "webserver" | "contentType" | "vhost"

export interface EndpointFilterOption {
  value: string
  label: string
  count?: number
}

export interface EndpointFilterOptionsResponse {
  results: EndpointFilterOption[]
}

// Create Endpoint request parameters
export interface CreateEndpointRequest {
  url: string
  method?: string
  statusCode?: number | null
  title?: string
  contentLength?: number | null
  contentType?: string | null
  domain?: string
  subdomain?: string
}

// Create Endpoint response (extends common batch create response)
export type CreateEndpointsResponse = BatchCreateResponse

// Update Endpoint request parameters
export interface UpdateEndpointRequest {
  id: number
  url?: string
  method?: string
  statusCode?: number
  title?: string
  contentLength?: number
  contentType?: string | null
  domain?: string
  subdomain?: string
}

// Batch delete Endpoint request parameters
export interface BatchDeleteEndpointsRequest {
  targetId: number
  ids: number[]
}

// Batch delete Endpoint response data
export interface BatchDeleteEndpointsResponse {
  message: string
  deletedCount: number
}
