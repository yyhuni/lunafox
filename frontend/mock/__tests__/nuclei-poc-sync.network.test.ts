import { afterEach, describe, expect, it } from "vitest"

import { resetMockNucleiPocs } from "@/mock/data/nuclei-pocs"
import { resetMockScenario } from "@/mock/scenarios"

const API_BASE = "http://localhost/v1"

describe("Nuclei POC sync cancellation mock boundary", () => {
  afterEach(() => {
    resetMockNucleiPocs()
    resetMockScenario()
  })

  it("serves the :cancel custom method and converges to CANCELLED", async () => {
    const create = await fetch(`${API_BASE}/nucleiPocSources:sync`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        requestId: "00000000-0000-4000-8000-000000000106",
        sourceType: "git",
        repoUrl: "https://github.com/projectdiscovery/nuclei-templates.git",
      }),
    })
    expect(create.status).toBe(201)
    const task = await create.json() as { name: string }

    const cancel = await fetch(`${API_BASE}/${task.name}:cancel`, { method: "POST" })
    expect(cancel.status).toBe(200)
    expect((await cancel.json()).state).toBe("CANCELLING")

    const terminal = await fetch(`${API_BASE}/${task.name}`)
    expect(terminal.status).toBe(200)
    expect((await terminal.json()).state).toBe("CANCELLED")
  })
})
