import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { LOCALE_COOKIE_NAME } from "@/i18n/config"

const mockRefresh = vi.fn()
const mockReplace = vi.fn()
const mockUpdateLocaleAsync = vi.fn()

vi.mock("next-intl", () => ({
  useLocale: () => "zh",
  useTranslations: () => (key: string) => key,
}))

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    refresh: mockRefresh,
    replace: mockReplace,
  }),
}))

vi.mock("@/hooks/use-notification-settings", () => ({
  useUpdateNotificationLocale: () => ({
    mutateAsync: mockUpdateLocaleAsync,
    isPending: false,
  }),
}))

vi.mock("@/components/shared/dropdown-menu-owners", () => ({
  HeaderIconActionMenu: ({
    children,
    ariaLabel,
  }: {
    children: React.ReactNode
    ariaLabel: string
  }) => (
    <div aria-label={ariaLabel}>
      {children}
    </div>
  ),
}))

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenuRadioGroup: ({
    children,
  }: {
    children: React.ReactNode
  }) => <div role="radiogroup">{children}</div>,
  DropdownMenuRadioItem: ({
    children,
    onClick,
  }: {
    children: React.ReactNode
    onClick?: () => void
  }) => (
    <button type="button" role="menuitemradio" aria-checked={false} onClick={onClick}>
      {children}
    </button>
  ),
}))

describe("LanguageSwitcher", () => {
  beforeEach(() => {
    mockRefresh.mockReset()
    mockReplace.mockReset()
    mockUpdateLocaleAsync.mockReset()
    mockUpdateLocaleAsync.mockResolvedValue({ locale: "en" })
    document.cookie = `${LOCALE_COOKIE_NAME}=; Path=/; Max-Age=0; SameSite=Lax`
  })

  it("切换语言时持久化通知 locale、写入 cookie 并刷新当前 canonical 路由", async () => {
    const { LanguageSwitcher } = await import("@/components/language-switcher")

    render(<LanguageSwitcher />)

    fireEvent.click(screen.getByRole("menuitemradio", { name: "English" }))

    await waitFor(() => expect(document.cookie).toContain(`${LOCALE_COOKIE_NAME}=en`))
    expect(mockUpdateLocaleAsync).toHaveBeenCalledWith("en")
    expect(mockRefresh).toHaveBeenCalledTimes(1)
    expect(mockReplace).not.toHaveBeenCalled()
  })

  it("通知 locale 持久化失败时不切换页面语言", async () => {
    mockUpdateLocaleAsync.mockRejectedValueOnce(new Error("offline"))
    const { LanguageSwitcher } = await import("@/components/language-switcher")

    render(<LanguageSwitcher />)
    fireEvent.click(screen.getByRole("menuitemradio", { name: "English" }))

    await waitFor(() => expect(mockUpdateLocaleAsync).toHaveBeenCalledWith("en"))
    expect(document.cookie).not.toContain(`${LOCALE_COOKIE_NAME}=en`)
    expect(mockRefresh).not.toHaveBeenCalled()
  })
})
