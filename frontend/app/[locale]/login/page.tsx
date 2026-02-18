"use client"

import { lazyPage } from "@/components/common/lazy-page"

const LoginPageContent = lazyPage(
  () => import("./content"), "min-h-svh"
)

export default function LoginPage() {
  return <LoginPageContent />
}
