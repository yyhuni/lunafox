export const normalizeOrigin = (value: string): string => value.replace(/\/+$/, "")

export const buildInstallCommand = (token: string, scriptBaseURL: string): string => {
  if (!token) return ""
  const trimmedUrl = scriptBaseURL.trim()
  if (!trimmedUrl) return ""

  const scriptBaseUrl = normalizeOrigin(trimmedUrl)
  return `curl -kfsSL "${scriptBaseUrl}/api/agent/install-script?token=${encodeURIComponent(token)}&mode=remote" | bash`
}
