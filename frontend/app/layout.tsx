import type React from "react"

/**
 * Root layout component
 * This is the outermost layout, actual content is handled by [locale]/layout.tsx
 */
export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return children
}
