import { describe, expect, it } from 'vitest'
import {
  mergeScanWorkflowConfigurations,
  parseScanWorkflowCapabilities,
  parseScanWorkflowConfiguration,
  serializeScanWorkflowConfiguration,
} from '@/lib/scan-workflow-config'

describe('scan-workflow-config helpers', () => {
  it('从 YAML 文本解析 scan workflow 配置对象', () => {
    expect(
      parseScanWorkflowConfiguration('steps:\n  subdomain_discovery:\n    enabled: true\n    engineConfig:\n      recon:\n        enabled: true')
    ).toEqual({
      steps: {
        subdomain_discovery: {
          enabled: true,
          engineConfig: {
            recon: {
              enabled: true,
            },
          },
        },
      },
    })
  })

  it('从对象配置提取 capability', () => {
    expect(parseScanWorkflowCapabilities({
      steps: {
        subdomain_discovery: { enabled: true, engineConfig: {} },
        vuln_scan: { enabled: false },
      },
    })).toEqual([
      'subdomain_discovery',
      'vuln_scan',
    ])
  })

  it('合并多个 scan workflow 配置对象', () => {
    expect(
      mergeScanWorkflowConfigurations([
        { steps: { subdomain_discovery: { enabled: true, engineConfig: { recon: { enabled: true } } } } },
        { steps: { vuln_scan: { enabled: false } } },
      ])
    ).toEqual({
      steps: {
        subdomain_discovery: { enabled: true, engineConfig: { recon: { enabled: true } } },
        vuln_scan: { enabled: false },
      },
    })
  })

  it('把对象配置序列化回 YAML', () => {
    const payload = serializeScanWorkflowConfiguration({
      steps: {
        subdomain_discovery: {
          enabled: true,
          engineConfig: {
            recon: { enabled: true },
          },
        },
      },
    })
    expect(payload).toContain('steps:')
    expect(payload).toContain('engineConfig:')
    expect(payload).toContain('enabled: true')
  })
})
