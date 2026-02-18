"use client"

import {
  IPAddressesViewContent,
  IPAddressesViewErrorState,
  IPAddressesViewLoadingState,
} from "./ip-addresses-view-sections"
import { useIPAddressesViewState } from "./ip-addresses-view-state"

export function IPAddressesView({
  targetId,
  scanId,
}: {
  targetId?: number
  scanId?: number
}) {
  const state = useIPAddressesViewState({ targetId, scanId })

  if (state.error) {
    return <IPAddressesViewErrorState state={state} />
  }

  if (state.isLoading && !state.data) {
    return <IPAddressesViewLoadingState />
  }

  return <IPAddressesViewContent state={state} />
}
