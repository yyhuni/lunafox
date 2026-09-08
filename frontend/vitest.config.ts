import path from "node:path"
import { fileURLToPath } from "node:url"
import { defineConfig } from "vitest/config"

const __dirname = path.dirname(fileURLToPath(import.meta.url))

export default defineConfig({
  oxc: {
    jsx: {
      runtime: "automatic",
      importSource: "react",
    },
  },
  resolve: {
    alias: {
      "@": __dirname,
    },
  },
  test: {
    environment: "jsdom",
    // @ts-expect-error Vitest InlineConfig type definition omits environmentMatchGlobs
    environmentMatchGlobs: [
      ["**/*.contract.test.*", "node"],
      ["**/*.types.test.*", "node"],
    ],
    globals: true,
    setupFiles: ["./vitest.setup.ts"],
    include: ["**/*.{test,spec}.{ts,tsx}"],
    exclude: ["node_modules", ".next", "dist", "coverage"],
    coverage: {
      provider: "v8",
      reporter: ["text", "lcov"],
      reportsDirectory: "./coverage",
    },
  },
})
