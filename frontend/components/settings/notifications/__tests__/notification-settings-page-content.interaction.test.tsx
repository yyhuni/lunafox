import { fireEvent, render, screen, waitFor, within } from "@testing-library/react"
import { QueryClientProvider } from "@tanstack/react-query"
import { beforeEach, describe, expect, it, vi } from "vitest"

import NotificationSettingsPageContent from "../notification-settings-page-content"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const destinationQuery = vi.hoisted(() => ({
  data: {
    results: [
      {
        provider: "discord" as const,
        credential: "https://discord.com/api/webhooks/discord-id/discord-token",
        enabled: true,
        subscriptions: ["scan-failed" as const],
        requiresWebhookUpdate: false,
      },
      {
        provider: "wecom" as const,
        credential: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=wecom-key",
        enabled: true,
        subscriptions: ["scan-failed" as const],
        requiresWebhookUpdate: false,
      },
      {
        provider: "feishu" as const,
        credential: "",
        enabled: false,
        subscriptions: [],
        requiresWebhookUpdate: false,
      },
    ],
    supportedKinds: ["scan-succeeded", "scan-failed", "vulnerability-observed", "agent-offline"] as const,
  },
  isError: false,
  isLoading: false,
  refetch: vi.fn(),
}))

const destinationMutation = vi.hoisted(() => ({
  isPending: false,
  mutateAsync: vi.fn(),
}))

const testMutation = vi.hoisted(() => ({
  isPending: false,
  mutateAsync: vi.fn(),
}))

const toastMocks = vi.hoisted(() => ({
  dismiss: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  success: vi.fn(),
  warning: vi.fn(),
}))

vi.mock("@/hooks/use-notification-settings", () => ({
  notificationSettingsKeys: {
    destinations: () => ["notification-settings", "destinations"],
  },
  useNotificationDestinations: () => destinationQuery,
  useTestNotificationDestination: () => testMutation,
  useUpdateNotificationDestination: () => destinationMutation,
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

describe("NotificationSettingsPageContent channels", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    destinationQuery.data.results[0].requiresWebhookUpdate = false
    destinationQuery.data.results[1].credential = "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=wecom-key"
    destinationQuery.data.results[1].enabled = true
    destinationQuery.data.results[1].subscriptions = ["scan-failed"]
    destinationQuery.data.results[2].credential = ""
    destinationQuery.data.results[2].enabled = false
    destinationQuery.data.results[2].subscriptions = []
    destinationQuery.data.results[2].requiresWebhookUpdate = false
    destinationMutation.mutateAsync.mockImplementation(async ({ provider, data }) => ({
      provider,
      ...data,
      requiresWebhookUpdate: false,
    }))
    testMutation.mutateAsync.mockResolvedValue({ result: "unavailable" })
  })

  function renderPage() {
    const queryClient = createTestQueryClient()
    return render(
      <QueryClientProvider client={queryClient}>
        <NotificationSettingsPageContent
          pageTitle="notifications"
          pageDescription="settings"
          deferInitialSkeleton
        />
      </QueryClientProvider>,
    )
  }

  it("shows the three fixed configurable external destinations in order and one shared save action", () => {
    renderPage()

    expect(screen.getByTestId("notification-channel-editor-discord")).toBeInTheDocument()
    expect(screen.getByTestId("notification-channel-editor-wecom")).toBeInTheDocument()
    expect(screen.getByTestId("notification-channel-editor-feishu")).toBeInTheDocument()
    expect(screen.getByTestId("notification-channel-editors").children).toHaveLength(4)
    expect(screen.getByRole("button", { name: "destination.saveAll" })).toBeDisabled()
    expect(screen.queryByRole("radio")).not.toBeInTheDocument()
    expect(screen.queryByText("inbox.enabledLabel")).not.toBeInTheDocument()
  })

  it("keeps all provider drafts independently editable while credential visibility stays local", () => {
    renderPage()

    const discordEditor = within(screen.getByTestId("notification-channel-editor-discord"))
    const wecomEditor = within(screen.getByTestId("notification-channel-editor-wecom"))
    const feishuEditor = within(screen.getByTestId("notification-channel-editor-feishu"))
    const discordCredential = discordEditor.getByLabelText("destination.credentialLabel")

    expect(screen.getAllByLabelText("destination.credentialLabel")).toHaveLength(2)
    expect(discordCredential).toHaveAttribute("type", "password")
    expect(wecomEditor.getByLabelText("destination.credentialLabel")).toHaveAttribute("type", "password")

    fireEvent.click(discordEditor.getByRole("button", { name: "destination.showCredential" }))
    fireEvent.change(wecomEditor.getByLabelText("destination.credentialLabel"), { target: { value: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=draft" } })

    expect(discordCredential).toHaveAttribute("type", "text")
    expect(discordCredential).toHaveValue("https://discord.com/api/webhooks/discord-id/discord-token")
    expect(wecomEditor.getByLabelText("destination.credentialLabel")).toHaveValue("https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=draft")
    expect(screen.getByRole("button", { name: "destination.saveAll" })).toBeEnabled()
    expect(feishuEditor.queryByLabelText("destination.credentialLabel")).not.toBeInTheDocument()
  })

  it("collapses disabled channel details without discarding the draft", () => {
    renderPage()

    const wecomEditor = within(screen.getByTestId("notification-channel-editor-wecom"))
    const credential = wecomEditor.getByLabelText("destination.credentialLabel")

    fireEvent.change(credential, { target: { value: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=kept-draft" } })
    fireEvent.click(wecomEditor.getByRole("switch"))

    expect(wecomEditor.getByRole("switch")).toHaveAttribute("aria-checked", "false")
    expect(wecomEditor.queryByLabelText("destination.credentialLabel")).not.toBeInTheDocument()
    expect(wecomEditor.getByText("wecom.title")).toBeInTheDocument()

    fireEvent.click(wecomEditor.getByRole("switch"))

    expect(wecomEditor.getByRole("switch")).toHaveAttribute("aria-checked", "true")
    expect(wecomEditor.getByLabelText("destination.credentialLabel")).toHaveValue("https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=kept-draft")
  })

  it("expands the disabled Feishu editor without mutating another provider draft", () => {
    renderPage()

    const discordEditor = within(screen.getByTestId("notification-channel-editor-discord"))
    const feishuEditor = within(screen.getByTestId("notification-channel-editor-feishu"))

    expect(feishuEditor.queryByLabelText("destination.credentialLabel")).not.toBeInTheDocument()
    fireEvent.click(feishuEditor.getByRole("switch"))
    fireEvent.change(feishuEditor.getByLabelText("destination.credentialLabel"), {
      target: { value: "https://open.feishu.cn/open-apis/bot/v2/hook/draft-token" },
    })

    expect(feishuEditor.getByLabelText("destination.credentialLabel")).toHaveValue("https://open.feishu.cn/open-apis/bot/v2/hook/draft-token")
    expect(discordEditor.getByLabelText("destination.credentialLabel")).toHaveValue("https://discord.com/api/webhooks/discord-id/discord-token")
  })

  it("submits only dirty providers, retains failed drafts, and does not resubmit a successful provider", async () => {
    destinationMutation.mutateAsync
      .mockResolvedValueOnce({
        provider: "discord",
        credential: "https://discord.com/api/webhooks/discord-id/updated-token",
        enabled: true,
        subscriptions: ["scan-failed"],
        requiresWebhookUpdate: false,
      })
      .mockRejectedValueOnce(new Error("provider failure"))
      .mockResolvedValueOnce({
        provider: "wecom",
        credential: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=draft",
        enabled: true,
        subscriptions: ["scan-failed"],
        requiresWebhookUpdate: false,
      })
    renderPage()

    const discordEditor = within(screen.getByTestId("notification-channel-editor-discord"))
    const wecomEditor = within(screen.getByTestId("notification-channel-editor-wecom"))
    fireEvent.change(discordEditor.getByLabelText("destination.credentialLabel"), { target: { value: "https://discord.com/api/webhooks/discord-id/updated-token" } })
    fireEvent.change(wecomEditor.getByLabelText("destination.credentialLabel"), { target: { value: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=draft" } })
    fireEvent.click(screen.getByRole("button", { name: "destination.saveAll" }))

    await waitFor(() => expect(destinationMutation.mutateAsync).toHaveBeenCalledTimes(2))
    expect(destinationMutation.mutateAsync).toHaveBeenNthCalledWith(1, {
      provider: "discord",
      data: {
        credential: "https://discord.com/api/webhooks/discord-id/updated-token",
        enabled: true,
        subscriptions: ["scan-failed"],
      },
    })
    expect(wecomEditor.getByLabelText("destination.credentialLabel")).toHaveValue("https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=draft")
    await waitFor(() => expect(wecomEditor.getByRole("alert")).toHaveTextContent("destination.saveFailed"))

    fireEvent.click(screen.getByRole("button", { name: "destination.saveAll" }))
    await waitFor(() => expect(destinationMutation.mutateAsync).toHaveBeenCalledTimes(3))
    expect(destinationMutation.mutateAsync).toHaveBeenLastCalledWith({
        provider: "wecom",
        data: {
          credential: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=draft",
          enabled: true,
          subscriptions: ["scan-failed"],
      },
    })
  })

  it("tests the current unsaved draft without persisting it and exposes mock unavailability", async () => {
    renderPage()

    const wecomEditor = within(screen.getByTestId("notification-channel-editor-wecom"))
    const credential = "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=unsaved"
    fireEvent.change(wecomEditor.getByLabelText("destination.credentialLabel"), { target: { value: credential } })
    fireEvent.click(wecomEditor.getByRole("button", { name: "test.action" }))

    await waitFor(() => expect(testMutation.mutateAsync).toHaveBeenCalledWith({
      provider: "wecom",
      data: { credential },
    }))
    expect(destinationMutation.mutateAsync).not.toHaveBeenCalled()
    expect(wecomEditor.getByLabelText("destination.credentialLabel")).toHaveValue(credential)
    await waitFor(() => expect(wecomEditor.getByRole("alert")).toHaveTextContent("test.result.unavailable"))
  })

  it("tests the current unsaved Feishu draft without normalizing it or persisting it", async () => {
    renderPage()

    const feishuEditor = within(screen.getByTestId("notification-channel-editor-feishu"))
    const credential = "https://open.feishu.cn/open-apis/bot/v2/hook/unsaved-token"
    fireEvent.click(feishuEditor.getByRole("switch"))
    fireEvent.change(feishuEditor.getByLabelText("destination.credentialLabel"), { target: { value: credential } })
    fireEvent.click(feishuEditor.getByRole("button", { name: "test.action" }))

    await waitFor(() => expect(testMutation.mutateAsync).toHaveBeenCalledWith({
      provider: "feishu",
      data: { credential },
    }))
    expect(destinationMutation.mutateAsync).not.toHaveBeenCalled()
    expect(feishuEditor.getByLabelText("destination.credentialLabel")).toHaveValue(credential)
  })

  it("locks only the tested provider and preserves independent draft editing", async () => {
    let completeTest: ((value: { result: "unavailable" }) => void) | undefined
    testMutation.mutateAsync.mockImplementation(() => new Promise((resolve) => {
      completeTest = resolve
    }))
    renderPage()

    const discordEditor = within(screen.getByTestId("notification-channel-editor-discord"))
    const wecomEditor = within(screen.getByTestId("notification-channel-editor-wecom"))
    const feishuEditor = within(screen.getByTestId("notification-channel-editor-feishu"))
    fireEvent.click(wecomEditor.getByRole("button", { name: "test.action" }))

    await waitFor(() => expect(wecomEditor.getByLabelText("destination.credentialLabel")).toBeDisabled())
    expect(wecomEditor.getByRole("button", { name: /test\.action$/ })).toBeDisabled()
    expect(discordEditor.getByLabelText("destination.credentialLabel")).toBeEnabled()
    expect(discordEditor.getByRole("button", { name: /test\.action$/ })).toBeEnabled()
    expect(feishuEditor.getByRole("switch")).toBeEnabled()
    expect(screen.getByRole("button", { name: /destination\.saveAll$/ })).toBeDisabled()

    completeTest?.({ result: "unavailable" })
    await waitFor(() => expect(wecomEditor.getByLabelText("destination.credentialLabel")).toBeEnabled())
  })

  it("locks all channel controls during a shared save", async () => {
    let completeSave: ((value: { provider: "discord"; credential: string; enabled: boolean; subscriptions: ("scan-failed")[]; requiresWebhookUpdate: false }) => void) | undefined
    destinationMutation.mutateAsync.mockImplementation(() => new Promise((resolve) => {
      completeSave = resolve
    }))
    renderPage()

    const discordEditor = within(screen.getByTestId("notification-channel-editor-discord"))
    const wecomEditor = within(screen.getByTestId("notification-channel-editor-wecom"))
    const feishuEditor = within(screen.getByTestId("notification-channel-editor-feishu"))
    const updatedCredential = "https://discord.com/api/webhooks/discord-id/updated-token"
    fireEvent.change(discordEditor.getByLabelText("destination.credentialLabel"), { target: { value: updatedCredential } })
    fireEvent.click(screen.getByRole("button", { name: "destination.saveAll" }))

    await waitFor(() => expect(discordEditor.getByLabelText("destination.credentialLabel")).toBeDisabled())
    expect(wecomEditor.getByLabelText("destination.credentialLabel")).toBeDisabled()
    expect(discordEditor.getByRole("button", { name: /test\.action$/ })).toBeDisabled()
    expect(wecomEditor.getByRole("button", { name: /test\.action$/ })).toBeDisabled()
    expect(feishuEditor.getByRole("switch")).toBeDisabled()
    expect(screen.getByRole("button", { name: /destination\.saveAll$/ })).toBeDisabled()

    completeSave?.({
      provider: "discord",
      credential: updatedCredential,
      enabled: true,
      subscriptions: ["scan-failed"],
      requiresWebhookUpdate: false,
    })
    await waitFor(() => expect(discordEditor.getByLabelText("destination.credentialLabel")).toBeEnabled())
  })

  it("adds memory-only unload protection while dirty and removes it after revert or unmount", async () => {
    const addEventListener = vi.spyOn(window, "addEventListener")
    const removeEventListener = vi.spyOn(window, "removeEventListener")
    const beforeUnloadCalls = (spy: typeof addEventListener) => spy.mock.calls.filter(([event]) => event === "beforeunload").length
    const page = renderPage()
    const wecomEditor = within(screen.getByTestId("notification-channel-editor-wecom"))
    const credential = wecomEditor.getByLabelText("destination.credentialLabel")

    fireEvent.change(credential, { target: { value: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=draft" } })
    await waitFor(() => expect(beforeUnloadCalls(addEventListener)).toBe(1))

    fireEvent.change(credential, { target: { value: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=wecom-key" } })
    await waitFor(() => expect(beforeUnloadCalls(removeEventListener)).toBe(1))

    fireEvent.change(credential, { target: { value: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=draft-again" } })
    await waitFor(() => expect(beforeUnloadCalls(addEventListener)).toBe(2))
    page.unmount()
    expect(beforeUnloadCalls(removeEventListener)).toBe(2)
  })

  it("blocks invalid enablement state at the shared save boundary", async () => {
    destinationQuery.data.results[1].credential = ""
    destinationQuery.data.results[1].enabled = false
    destinationQuery.data.results[1].subscriptions = []
    renderPage()

    const wecomEditor = within(screen.getByTestId("notification-channel-editor-wecom"))
    fireEvent.click(wecomEditor.getByRole("switch"))
    fireEvent.change(wecomEditor.getByLabelText("destination.credentialLabel"), { target: { value: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=draft" } })
    fireEvent.click(screen.getByRole("button", { name: "destination.saveAll" }))

    expect(wecomEditor.getByRole("alert")).toHaveTextContent("validation.subscriptionRequired")
    expect(destinationMutation.mutateAsync).not.toHaveBeenCalled()
  })

  it("shows only the server-derived webhook remediation warning", () => {
    destinationQuery.data.results[0].requiresWebhookUpdate = true
    renderPage()

    expect(within(screen.getByTestId("notification-channel-editor-discord")).getByRole("alert")).toHaveTextContent("remediation.required")
    expect(within(screen.getByTestId("notification-channel-editor-wecom")).queryByText("remediation.required")).not.toBeInTheDocument()
  })
})
