"use client"

import * as React from "react"

import { readStorageValue, writeStorageValue } from "@/lib/browser-storage"

export const LOGIN_VISUAL_UNLOCK_STORAGE_KEY = "lunafox.login-visual-unlocked"
export const LOGIN_VISUAL_UNLOCK_EVENT = "lunafox:login-visual-unlocked"

export function isLoginVisualUnlocked() {
  return readStorageValue(LOGIN_VISUAL_UNLOCK_STORAGE_KEY) === "true"
}

export function unlockLoginVisual() {
  if (isLoginVisualUnlocked()) return false
  if (!writeStorageValue(LOGIN_VISUAL_UNLOCK_STORAGE_KEY, "true")) return false

  window.dispatchEvent(new Event(LOGIN_VISUAL_UNLOCK_EVENT))
  return true
}

function subscribeToLoginVisualUnlock(onStoreChange: () => void) {
  const handleStorage = (event: StorageEvent) => {
    if (event.key === LOGIN_VISUAL_UNLOCK_STORAGE_KEY) onStoreChange()
  }

  window.addEventListener("storage", handleStorage)
  window.addEventListener(LOGIN_VISUAL_UNLOCK_EVENT, onStoreChange)
  return () => {
    window.removeEventListener("storage", handleStorage)
    window.removeEventListener(LOGIN_VISUAL_UNLOCK_EVENT, onStoreChange)
  }
}

export function useLoginVisualUnlocked() {
  return React.useSyncExternalStore(subscribeToLoginVisualUnlock, isLoginVisualUnlocked, () => false)
}
