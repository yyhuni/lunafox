import { describe, expect, it } from "vitest"

const API_BASE = "http://localhost/v1"

describe("Screenshot image mock network boundaries", () => {
  it("serves target and scan screenshot blobs as SVG image responses", async () => {
    const [targetResponse, scanResponse] = await Promise.all([
      fetch(`${API_BASE}/screenshots/1/blob`),
      fetch(`${API_BASE}/scans/7/screenshotSnapshots/2/blob`),
    ])

    expect(targetResponse.status).toBe(200)
    expect(scanResponse.status).toBe(200)
    expect(targetResponse.headers.get("content-type")).toContain("image/svg+xml")
    expect(scanResponse.headers.get("content-type")).toContain("image/svg+xml")

    const [targetImage, scanImage] = await Promise.all([
      targetResponse.text(),
      scanResponse.text(),
    ])

    expect(targetImage).toContain("<svg")
    expect(targetImage).toContain("Lunafox mock screenshot 1")
    expect(scanImage).toContain("<svg")
    expect(scanImage).toContain("Lunafox mock screenshot 2")
  })

  it("does not retain legacy screenshot image routes", async () => {
    const [targetResponse, scanResponse] = await Promise.all([
      fetch(`${API_BASE}/assets/screenshots/1/image`),
      fetch(`${API_BASE}/scans/7/screenshots/2/image`),
    ])

    expect(targetResponse.status).toBe(501)
    expect(scanResponse.status).toBe(501)
  })
})
