import type { Meta, StoryObj } from "@storybook/nextjs-vite"
import { expect, userEvent, waitFor, within } from "storybook/test"

import { ResponseEvidencePanel } from "./response-evidence-panel"

const longResponse = Array.from({ length: 80 }, (_, index) => `Response line ${index}`).join("\n")

const meta = {
  title: "Shared/Response Evidence Panel",
  component: ResponseEvidencePanel,
  parameters: { layout: "padded" },
  args: {
    responseBody: longResponse,
    responseHeaders: longResponse,
    location: longResponse,
    labels: {
      body: "Body",
      headers: "Headers",
      location: "Location",
      emptyLabel: "No response",
    },
    showMetadata: true,
    metadata: {
      contentType: "text/html",
      contentLength: longResponse.length,
      labels: { contentType: "Content type", responseSize: "Response size" },
    },
  },
  render: (args) => (
    <div className="h-80 w-full max-w-3xl">
      <ResponseEvidencePanel {...args} />
    </div>
  ),
} satisfies Meta<typeof ResponseEvidencePanel>

export default meta
type Story = StoryObj<typeof meta>

export const LongResponses: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    for (const name of ["Body", "Headers", "Location"]) {
      await userEvent.click(canvas.getByRole("tab", { name }))
      const panel = canvas.getByRole("tabpanel", { name })
      const viewport = panel.querySelector<HTMLElement>('[data-id$="-viewport"]')
      if (!viewport) throw new Error("Response scroll viewport is missing")

      // A clipped Root can look scrollable while the viewport has no scroll range.
      await waitFor(() => {
        expect(viewport.clientHeight).toBeGreaterThan(0)
        expect(viewport.scrollHeight).toBeGreaterThan(viewport.clientHeight)
        expect(viewport.getBoundingClientRect().height).toBeLessThanOrEqual(panel.getBoundingClientRect().height)
      })
      viewport.scrollTo({ top: viewport.scrollHeight })
      await waitFor(() => {
        expect(viewport.scrollTop).toBeGreaterThan(0)
        expect(viewport.scrollTop + viewport.clientHeight).toBeGreaterThanOrEqual(viewport.scrollHeight - 1)
      })
    }
  },
}

export const ShortResponse: Story = {
  args: { responseBody: "OK", responseHeaders: "server: nginx", location: "" },
}
