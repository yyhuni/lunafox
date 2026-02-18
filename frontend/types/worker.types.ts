/**
 * Worker node related type definitions
 */

// Worker status enum (unified between frontend and backend)
export type WorkerStatus = 'pending' | 'deploying' | 'online' | 'offline' | 'updating' | 'outdated'

// Worker node
export interface WorkerNode {
  id: number
  name: string
  ipAddress: string
  sshPort: number
  username: string
  status: WorkerStatus
  isLocal: boolean  // Whether it's a local node (inside Docker container)
  createdAt: string
  updatedAt?: string
  info?: {
    cpuPercent?: number
    memoryPercent?: number
  }
}

// Create Worker request
export interface CreateWorkerRequest {
  name: string
  ipAddress: string
  sshPort?: number
  username?: string
  password: string
}

// Update Worker request
export interface UpdateWorkerRequest {
  name?: string
  sshPort?: number
  username?: string
  password?: string
}

// Worker list response
export interface WorkersResponse {
  results: WorkerNode[]
  total: number
  page: number
  pageSize: number
}

