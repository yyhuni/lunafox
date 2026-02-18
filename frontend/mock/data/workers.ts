import type { WorkerNode, WorkersResponse } from '@/types/worker.types'

export const mockWorkers: WorkerNode[] = [
  {
    id: 1,
    name: 'local-worker',
    ipAddress: '127.0.0.1',
    sshPort: 22,
    username: 'root',
    status: 'online',
    isLocal: true,
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-12-29T10:00:00Z',
    info: {
      cpuPercent: 23.5,
      memoryPercent: 45.2,
    },
  },
  {
    id: 2,
    name: 'worker-01',
    ipAddress: '192.168.1.101',
    sshPort: 22,
    username: 'scanner',
    status: 'online',
    isLocal: false,
    createdAt: '2024-06-15T08:00:00Z',
    updatedAt: '2024-12-29T09:30:00Z',
    info: {
      cpuPercent: 56.8,
      memoryPercent: 72.1,
    },
  },
  {
    id: 3,
    name: 'worker-02',
    ipAddress: '192.168.1.102',
    sshPort: 22,
    username: 'scanner',
    status: 'online',
    isLocal: false,
    createdAt: '2024-07-20T10:00:00Z',
    updatedAt: '2024-12-29T09:45:00Z',
    info: {
      cpuPercent: 34.2,
      memoryPercent: 58.9,
    },
  },
  {
    id: 4,
    name: 'worker-03',
    ipAddress: '192.168.1.103',
    sshPort: 22,
    username: 'scanner',
    status: 'offline',
    isLocal: false,
    createdAt: '2024-08-10T14:00:00Z',
    updatedAt: '2024-12-28T16:00:00Z',
  },
]

export function getMockWorkers(page = 1, pageSize = 10): WorkersResponse {
  const total = mockWorkers.length
  const start = (page - 1) * pageSize
  const results = mockWorkers.slice(start, start + pageSize)

  return {
    results,
    total,
    page,
    pageSize,
  }
}

export function getMockWorkerById(id: number): WorkerNode | undefined {
  return mockWorkers.find(w => w.id === id)
}
