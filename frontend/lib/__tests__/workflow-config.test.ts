import { describe, expect, it } from 'vitest'
import {
  mergeWorkflowConfigurations,
  parseWorkflowCapabilities,
  parseWorkflowConfiguration,
  serializeWorkflowConfiguration,
} from '@/lib/workflow-config'

describe('workflow-config helpers', () => {
  it('从 YAML 文本解析 workflow 配置对象', () => {
    expect(parseWorkflowConfiguration('subdomain_discovery:\n  recon:\n    enabled: true')).toEqual({
      subdomain_discovery: {
        recon: {
          enabled: true,
        },
      },
    })
  })

  it('从对象配置提取 capability', () => {
    expect(parseWorkflowCapabilities({ subdomain_discovery: {}, vuln_scan: {} })).toEqual([
      'subdomain_discovery',
      'vuln_scan',
    ])
  })

  it('合并多个 workflow 配置对象', () => {
    expect(
      mergeWorkflowConfigurations([
        { subdomain_discovery: { recon: { enabled: true } } },
        { vuln_scan: { enabled: true } },
      ])
    ).toEqual({
      subdomain_discovery: { recon: { enabled: true } },
      vuln_scan: { enabled: true },
    })
  })

  it('把对象配置序列化回 YAML', () => {
    const payload = serializeWorkflowConfiguration({ subdomain_discovery: { recon: { enabled: true } } })
    expect(payload).toContain('subdomain_discovery:')
    expect(payload).toContain('enabled: true')
  })
})
