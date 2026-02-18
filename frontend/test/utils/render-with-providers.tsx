import * as React from "react"
import { QueryClientProvider, type QueryClient } from "@tanstack/react-query"
import { render, renderHook, type RenderHookOptions, type RenderOptions } from "@testing-library/react"
import { createTestQueryClient } from "./test-query-client"

type WrapperProps = { children: React.ReactNode }

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: WrapperProps) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

export function renderWithProviders(
  ui: React.ReactElement,
  options?: Omit<RenderOptions, "wrapper"> & { queryClient?: QueryClient }
) {
  const queryClient = options?.queryClient ?? createTestQueryClient()
  return {
    queryClient,
    ...render(ui, {
      ...options,
      wrapper: createWrapper(queryClient),
    }),
  }
}

export function renderHookWithProviders<Result, Props>(
  renderCallback: (initialProps: Props) => Result,
  options?: Omit<RenderHookOptions<Props>, "wrapper"> & { queryClient?: QueryClient }
) {
  const queryClient = options?.queryClient ?? createTestQueryClient()
  return {
    queryClient,
    ...renderHook(renderCallback, {
      ...options,
      wrapper: createWrapper(queryClient),
    }),
  }
}
