"use client"

import React, { useState, useEffect } from 'react'
import { IconEye, IconEyeOff, IconWorldSearch, IconRadar2 } from "@/components/icons"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Separator } from '@/components/ui/separator'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { useApiKeySettings, useUpdateApiKeySettings } from '@/hooks/use-api-key-settings'
import { PageHeader } from '@/components/common/page-header'
import type {
  ApiKeySettings,
  ProviderKey,
  FofaProviderConfig,
  CensysProviderConfig,
  SingleFieldProviderConfig,
} from '@/types/api-key-settings.types'

// Password input box component (with show/hide switch)
function PasswordInput({ value, onChange, placeholder, disabled, name }: {
  value: string
  onChange: (value: string) => void
  placeholder?: string
  disabled?: boolean
  name?: string
}) {
  const [show, setShow] = useState(false)
  return (
    <div className="relative">
      <Input
        type={show ? 'text' : 'password'}
        name={name}
        autoComplete={show ? "off" : "current-password"}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        disabled={disabled}
        className="pr-10"
      />
      <button
        type="button"
        onClick={() => setShow(!show)}
        className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
      >
        {show ? <IconEyeOff className="h-4 w-4" /> : <IconEye className="h-4 w-4" />}
      </button>
    </div>
  )
}

type ProviderField = {
  name: ProviderFieldName
  label: string
  type: "text" | "password"
  placeholder?: string
}

type ProviderFieldName =
  | keyof FofaProviderConfig
  | keyof CensysProviderConfig
  | keyof SingleFieldProviderConfig

type ProviderDefinition = {
  key: ProviderKey
  name: string
  description: string
  icon: React.ComponentType<{ className?: string }>
  color: string
  bgColor: string
  fields: ProviderField[]
  docUrl: string
}

// Provider configuration definition
const PROVIDERS: ProviderDefinition[] = [
  {
    key: 'fofa',
    name: 'FOFA',
    description: '网络空间测绘平台，提供全球互联网资产搜索',
    icon: IconWorldSearch,
    color: 'text-blue-500',
    bgColor: 'bg-blue-500/10',
    fields: [
      { name: 'email', label: '邮箱', type: 'text', placeholder: 'your@email.com' },
      { name: 'apiKey', label: 'API Key', type: 'password', placeholder: '输入 FOFA API Key' },
    ],
    docUrl: 'https://fofa.info/api',
  },
  {
    key: 'hunter',
    name: 'Hunter (鹰图)',
    description: '奇安信威胁情报平台，提供网络空间资产测绘',
    icon: IconRadar2,
    color: 'text-orange-500',
    bgColor: 'bg-orange-500/10',
    fields: [
      { name: 'apiKey', label: 'API Key', type: 'password', placeholder: '输入 Hunter API Key' },
    ],
    docUrl: 'https://hunter.qianxin.com/',
  },
  {
    key: 'shodan',
    name: 'Shodan',
    description: '全球最大的互联网设备搜索引擎',
    icon: IconWorldSearch,
    color: 'text-red-500',
    bgColor: 'bg-red-500/10',
    fields: [
      { name: 'apiKey', label: 'API Key', type: 'password', placeholder: '输入 Shodan API Key' },
    ],
    docUrl: 'https://developer.shodan.io/',
  },
  {
    key: 'censys',
    name: 'Censys',
    description: '互联网资产搜索和监控平台',
    icon: IconWorldSearch,
    color: 'text-purple-500',
    bgColor: 'bg-purple-500/10',
    fields: [
      { name: 'apiId', label: 'API ID', type: 'text', placeholder: '输入 Censys API ID' },
      { name: 'apiSecret', label: 'API Secret', type: 'password', placeholder: '输入 Censys API Secret' },
    ],
    docUrl: 'https://search.censys.io/api',
  },
  {
    key: 'zoomeye',
    name: 'ZoomEye (钟馗之眼)',
    description: '知道创宇网络空间搜索引擎',
    icon: IconWorldSearch,
    color: 'text-green-500',
    bgColor: 'bg-green-500/10',
    fields: [
      { name: 'apiKey', label: 'API Key', type: 'password', placeholder: '输入 ZoomEye API Key' },
    ],
    docUrl: 'https://www.zoomeye.org/doc',
  },
  {
    key: 'securitytrails',
    name: 'SecurityTrails',
    description: 'DNS 历史记录和子域名数据平台',
    icon: IconWorldSearch,
    color: 'text-cyan-500',
    bgColor: 'bg-cyan-500/10',
    fields: [
      { name: 'apiKey', label: 'API Key', type: 'password', placeholder: '输入 SecurityTrails API Key' },
    ],
    docUrl: 'https://securitytrails.com/corp/api',
  },
  {
    key: 'threatbook',
    name: 'ThreatBook (微步在线)',
    description: '威胁情报平台，提供域名和 IP 情报查询',
    icon: IconWorldSearch,
    color: 'text-indigo-500',
    bgColor: 'bg-indigo-500/10',
    fields: [
      { name: 'apiKey', label: 'API Key', type: 'password', placeholder: '输入 ThreatBook API Key' },
    ],
    docUrl: 'https://x.threatbook.com/api',
  },
  {
    key: 'quake',
    name: 'Quake (360)',
    description: '360 网络空间测绘系统',
    icon: IconWorldSearch,
    color: 'text-teal-500',
    bgColor: 'bg-teal-500/10',
    fields: [
      { name: 'apiKey', label: 'API Key', type: 'password', placeholder: '输入 Quake API Key' },
    ],
    docUrl: 'https://quake.360.net/quake/#/help',
  },
]

// Default configuration
const DEFAULT_SETTINGS: ApiKeySettings = {
  fofa: { enabled: false, email: '', apiKey: '' },
  hunter: { enabled: false, apiKey: '' },
  shodan: { enabled: false, apiKey: '' },
  censys: { enabled: false, apiId: '', apiSecret: '' },
  zoomeye: { enabled: false, apiKey: '' },
  securitytrails: { enabled: false, apiKey: '' },
  threatbook: { enabled: false, apiKey: '' },
  quake: { enabled: false, apiKey: '' },
}

export default function ApiKeysSettingsPage() {
  const pageTitle = 'API 密钥配置'
  const pageDescription =
    '配置第三方数据源的 API 密钥，用于增强子域名发现能力。启用后将在 subfinder 扫描时自动使用。'
  const { data: settings, isLoading } = useApiKeySettings()
  const updateMutation = useUpdateApiKeySettings()
  
  const [formData, setFormData] = useState<ApiKeySettings>(DEFAULT_SETTINGS)
  const [hasChanges, setHasChanges] = useState(false)

  // When the data is loaded, update the form data
  useEffect(() => {
    if (settings) {
      setFormData({ ...DEFAULT_SETTINGS, ...settings })
      setHasChanges(false)
    }
  }, [settings])

  const updateProvider = (
    providerKey: ProviderKey,
    field: ProviderFieldName,
    value: string | boolean
  ) => {
    setFormData((prev) => {
      const current = prev[providerKey]
      const updated = {
        ...current,
        [field]: value,
      } as typeof current
      return {
        ...prev,
        [providerKey]: updated,
      }
    })
    setHasChanges(true)
  }

  const handleSave = async () => {
    updateMutation.mutate(formData)
    setHasChanges(false)
  }


  if (isLoading) {
    return (
      <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
        <PageHeader
          code="API-01"
          title={pageTitle}
          description={pageDescription}
        />
        <div className="px-4 lg:px-6 space-y-6">
          <div>
            <Skeleton className="h-8 w-48" />
            <Skeleton className="h-4 w-96 mt-2" />
          </div>
          <div className="grid gap-4">
            {[1, 2, 3].map((i) => (
              <Skeleton key={i} className="h-24 w-full" />
            ))}
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <PageHeader
        code="API-01"
        title={pageTitle}
        description={pageDescription}
      />

      <div className="px-4 lg:px-6 space-y-6">
        {/* Provider cards */}
        <div className="grid gap-4">
          {PROVIDERS.map((provider) => {
            const data = formData[provider.key]
            const isEnabled = data.enabled

            return (
              <Card key={provider.key}>
                <CardHeader className="pb-4">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${provider.bgColor}`}>
                        <provider.icon className={`h-5 w-5 ${provider.color}`} />
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <CardTitle className="text-base">{provider.name}</CardTitle>
                          {isEnabled && <Badge variant="outline" className="text-xs text-green-600">已启用</Badge>}
                        </div>
                        <CardDescription>{provider.description}</CardDescription>
                      </div>
                    </div>
                    <Switch
                      checked={isEnabled}
                      onCheckedChange={(checked) => updateProvider(provider.key, 'enabled', checked)}
                    />
                  </div>
                </CardHeader>

                {/* Expanded configuration form */}
                {isEnabled && (
                  <CardContent className="pt-0">
                    <Separator className="mb-4" />
                    <div className="space-y-4">
                      {provider.fields.map((field) => {
                        const rawValue = (data as Record<ProviderFieldName, string | boolean>)[field.name]
                        const fieldValue = typeof rawValue === "string" ? rawValue : ""
                        return (
                        <div key={field.name} className="space-y-2">
                          <label className="text-sm font-medium">{field.label}</label>
                          {field.type === 'password' ? (
                            <PasswordInput
                              name={`${provider.key}-${String(field.name)}`}
                              value={fieldValue}
                              onChange={(value) => updateProvider(provider.key, field.name, value)}
                              placeholder={field.placeholder}
                            />
                          ) : (
                            <Input
                              type="text"
                              name={`${provider.key}-${String(field.name)}`}
                              autoComplete="off"
                              value={fieldValue}
                              onChange={(e) => updateProvider(provider.key, field.name, e.target.value)}
                              placeholder={field.placeholder}
                            />
                          )}
                        </div>
                      )})}
                      <p className="text-xs text-muted-foreground">
                        获取 API Key：
                        <a 
                          href={provider.docUrl} 
                          target="_blank" 
                          rel="noopener noreferrer"
                          className="text-primary hover:underline ml-1"
                        >
                          {provider.docUrl}
                        </a>
                      </p>
                    </div>
                  </CardContent>
                )}
              </Card>
            )
          })}
        </div>

        {/* Save button */}
        <div className="flex justify-end">
          <Button 
            onClick={handleSave} 
            disabled={updateMutation.isPending || !hasChanges}
          >
            {updateMutation.isPending ? '保存中…' : '保存配置'}
          </Button>
        </div>
      </div>
    </div>
  )
}
