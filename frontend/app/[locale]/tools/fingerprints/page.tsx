"use client"

import { redirect } from "next/navigation"

/**
 * Fingerprint management homepage - Redirect to EHole
 */
export default function FingerprintsPage() {
  redirect("/tools/fingerprints/ehole/")
}
