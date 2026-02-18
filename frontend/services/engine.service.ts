import apiClient from '@/lib/api-client'
import type { ScanEngine, PresetEngine } from '@/types/engine.types'
import { USE_MOCK, mockDelay, getMockEngines, getMockEngineById, getMockPresetEngines, getMockPresetEngineById } from '@/mock'

/**
 * Engine API service
 */

/**
 * Get preset engine list (system-defined, read-only)
 */
export async function getPresetEngines(): Promise<PresetEngine[]> {
  if (USE_MOCK) {
    await mockDelay()
    return getMockPresetEngines()
  }
  const response = await apiClient.get('/engines/presets/')
  return response.data
}

/**
 * Get preset engine by ID
 */
export async function getPresetEngine(id: string): Promise<PresetEngine> {
  if (USE_MOCK) {
    await mockDelay()
    const preset = getMockPresetEngineById(id)
    if (!preset) throw new Error('Preset engine not found')
    return preset
  }
  const response = await apiClient.get(`/engines/presets/${id}/`)
  return response.data
}

/**
 * Get user engine list (stored in database, editable)
 */
export async function getEngines(): Promise<ScanEngine[]> {
  if (USE_MOCK) {
    await mockDelay()
    return getMockEngines()
  }
  // Engines are usually not many, get all
  const response = await apiClient.get('/engines/', {
    params: { pageSize: 1000 }
  })
  // Backend returns paginated data: { results: [...], total, page, pageSize, totalPages }
  return response.data.results || response.data
}

/**
 * Get user engine details
 */
export async function getEngine(id: number): Promise<ScanEngine> {
  if (USE_MOCK) {
    await mockDelay()
    const engine = getMockEngineById(id)
    if (!engine) throw new Error('Engine not found')
    return engine
  }
  const response = await apiClient.get(`/engines/${id}/`)
  return response.data
}

/**
 * Create user engine
 */
export async function createEngine(data: {
  name: string
  configuration: string
}): Promise<ScanEngine> {
  if (USE_MOCK) {
    await mockDelay()
    // Mock create - return a new engine with generated ID
    const newEngine: ScanEngine = {
      id: Date.now(),
      name: data.name,
      configuration: data.configuration,
      isValid: true,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    }
    return newEngine
  }
  const response = await apiClient.post('/engines/', data)
  return response.data
}

/**
 * Update user engine
 */
export async function updateEngine(
  id: number,
  data: Partial<{
    name: string
    configuration: string
  }>
): Promise<ScanEngine> {
  const response = await apiClient.patch(`/engines/${id}/`, data)
  return response.data
}

/**
 * Delete user engine
 */
export async function deleteEngine(id: number): Promise<void> {
  await apiClient.delete(`/engines/${id}/`)
}

