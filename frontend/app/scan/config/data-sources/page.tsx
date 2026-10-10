import { ScanConfigurationWorkspace } from "@/components/scan/scan-configuration-workspace"

import { ApiKeysSettingsWorkspace } from "./api-keys-settings-workspace"

export default function ScanConfigurationDataSourcesPage() {
  return (
    <ScanConfigurationWorkspace activeTab="dataSources">
      <ApiKeysSettingsWorkspace />
    </ScanConfigurationWorkspace>
  )
}
