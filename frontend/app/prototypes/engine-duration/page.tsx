import { Suspense } from "react"

import { EngineDurationInputPrototype } from "@/components/scan/engine-duration-input-prototype"

export default function EngineDurationPrototypePage() {
  return (
    <Suspense fallback={null}>
      <EngineDurationInputPrototype />
    </Suspense>
  )
}
