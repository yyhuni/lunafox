export const normalizeOrigin = (value: string): string => value.replace(/\/+$/, "")

export const buildInstallCommand = (token: string, scriptBaseURL: string): string => {
  if (!token) return ""
  const trimmedUrl = scriptBaseURL.trim()
  if (!trimmedUrl) return ""

  const scriptBaseUrl = normalizeOrigin(trimmedUrl)
  return `curl -kfsSL "${scriptBaseUrl}/v1/agents:downloadInstallScript?registrationToken=${encodeURIComponent(token)}&profile=external" | bash`
}
