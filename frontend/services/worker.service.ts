/**
 * Worker node management API service
 */

import apiClient from '@/lib/api-client'
import type {
  WorkerNode,
  WorkersResponse,
  CreateWorkerRequest,
  UpdateWorkerRequest,
} from '@/types/worker.types'
import { USE_MOCK, mockDelay, getMockWorkers, getMockWorkerById } from '@/mock'

const BASE_URL = '/workers'

export const workerService = {
  /**
   * Get Worker list
   */
  async getWorkers(page = 1, pageSize = 10): Promise<WorkersResponse> {
    if (USE_MOCK) {
      await mockDelay()
      return getMockWorkers(page, pageSize)
    }
    const response = await apiClient.get<WorkersResponse>(
      `${BASE_URL}/?page=${page}&page_size=${pageSize}`
    )
    return response.data
  },

  /**
   * Get single Worker details
   */
  async getWorker(id: number): Promise<WorkerNode> {
    if (USE_MOCK) {
      await mockDelay()
      const worker = getMockWorkerById(id)
      if (!worker) throw new Error('Worker not found')
      return worker
    }
    const response = await apiClient.get<WorkerNode>(`${BASE_URL}/${id}/`)
    return response.data
  },

  /**
   * Create Worker node
   */
  async createWorker(data: CreateWorkerRequest): Promise<WorkerNode> {
    const response = await apiClient.post<WorkerNode>(`${BASE_URL}/`, {
      name: data.name,
      ip_address: data.ipAddress,
      ssh_port: data.sshPort ?? 22,
      username: data.username ?? 'root',
      password: data.password,
    })
    return response.data
  },

  /**
   * Update Worker node
   */
  async updateWorker(id: number, data: UpdateWorkerRequest): Promise<WorkerNode> {
    const response = await apiClient.patch<WorkerNode>(`${BASE_URL}/${id}/`, {
      name: data.name,
      ssh_port: data.sshPort,
      username: data.username,
      password: data.password,
    })
    return response.data
  },

  /**
   * Delete Worker node
   */
  async deleteWorker(id: number): Promise<void> {
    await apiClient.delete(`${BASE_URL}/${id}/`)
  },

  /**
   * Deploy Worker node (placeholder implementation, currently only used to eliminate frontend type errors)
   */
  async deployWorker(id: number): Promise<never> {
    return Promise.reject(new Error(`Worker deploy is not implemented for id=${id}`))
  },

  /**
   * Restart Worker
   */
  async restartWorker(id: number): Promise<{ message: string }> {
    const response = await apiClient.post<{ message: string }>(`${BASE_URL}/${id}/restart/`)
    return response.data
  },

  /**
   * Stop Worker
   */
  async stopWorker(id: number): Promise<{ message: string }> {
    const response = await apiClient.post<{ message: string }>(`${BASE_URL}/${id}/stop/`)
    return response.data
  },
}
