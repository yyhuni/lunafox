import { ScanConfigurationWorkspace } from "@/components/scan/scan-configuration-workspace"
import { EngineInstallationPage } from "@/components/tools/engines/engine-installation-page"

export default function ScanConfigurationEnginesPage() {
  return (
    <ScanConfigurationWorkspace activeTab="engines">
      <EngineInstallationPage embedded />
    </ScanConfigurationWorkspace>
  )
}
