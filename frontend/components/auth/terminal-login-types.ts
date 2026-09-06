export type LoginStep = "username" | "password" | "authenticating" | "success" | "error"

export interface TerminalLoginTranslations {
  title: string
  subtitle: string
  usernamePrompt: string
  passwordPrompt: string
  authenticating: string
  processing: string
  accessGranted: string
  welcomeMessage: string
  authFailed: string
  invalidCredentials: string
  shortcuts: string
  submit: string
  cancel: string
  clear: string
  startEnd: string
}

export interface TerminalLine {
  text: string
  type: "prompt" | "input" | "info" | "success" | "error" | "warning"
}
