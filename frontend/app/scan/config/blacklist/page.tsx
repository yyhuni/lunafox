import { ScanConfigurationWorkspace } from "@/components/scan/scan-configuration-workspace"
import { BlacklistSettingsWorkspace } from "@/components/settings/blacklist/blacklist-settings-workspace"

export default function ScanConfigurationBlacklistPage() {
  return (
    <ScanConfigurationWorkspace activeTab="blacklist">
      <BlacklistSettingsWorkspace />
    </ScanConfigurationWorkspace>
  )
}
