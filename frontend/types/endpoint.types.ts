// Endpoint specific data type definitions
// Note: Backend returns snake_case, but api-client.ts automatically converts to camelCase

import type { BatchCreateResponse } from './api-response.types'

export interface Endpoint {
  id: number
  url: string

  // HTTP metadata (may be null in some scenarios)
  method?: string
	statusCode: number | null       // Backend: status_code (pointer type, may be null)
	title: string
	contentLength: number | null    // Backend: content_length (pointer type, may be null)
	contentType?: string | null     // Backend: content_type (optional)
	responseTime?: number | null    // Backend: response_time (in seconds, optional)

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
  domainId?: number               // Backend: domain_id
  subdomainId?: number            // Backend: subdomain_id
  domain?: string
  subdomain?: string
  updatedAt?: string              // Backend: updated_at
}

// Endpoint list request parameters
// Backend sorts by update time descending
export interface GetEndpointsRequest {
  page?: number
  pageSize?: number
  search?: string
}

// Endpoint list response data
// Note: Backend returns snake_case, but api-client.ts automatically converts to camelCase
export interface GetEndpointsResponse {
  endpoints: Endpoint[]
  total: number
  page: number
  pageSize: number      // Backend returns camelCase format
  totalPages: number    // Backend returns camelCase format
  // Compatibility fields (backward compatible)
  page_size?: number
  total_pages?: number
}

// Create Endpoint request parameters
export interface CreateEndpointRequest {
  url: string                      // Required
  method?: string                  // Optional
	statusCode?: number | null       // Optional
	title?: string                   // Optional
	contentLength?: number | null    // Optional
	contentType?: string | null      // Optional
	responseTime?: number | null     // Optional
	domain?: string                  // Optional
	subdomain?: string               // Optional
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
	responseTime?: number | null
	domain?: string
	subdomain?: string
}

// Batch delete Endpoint request parameters
export interface BatchDeleteEndpointsRequest {
  endpointIds: number[]
}

// Batch delete Endpoint response data
export interface BatchDeleteEndpointsResponse {
  message: string
  deletedCount: number
}
