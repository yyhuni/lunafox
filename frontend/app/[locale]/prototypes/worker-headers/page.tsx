"use client"

import { WorkerHeaderVariants } from "@/components/prototypes/worker-header-variants"

export default function WorkerHeaderVariantsPage() {
    return (
        <div className="flex flex-1 flex-col gap-4 py-4 md:gap-8 md:py-8 max-w-7xl mx-auto px-4 lg:px-6 w-full">
            <div className="flex flex-col gap-2">
                <h1 className="text-3xl font-bold tracking-tight">Worker Top Bar Designs</h1>
                <p className="text-muted-foreground text-lg">Compare 5 different top layout designs for the Worker settings page.</p>
            </div>

            <WorkerHeaderVariants />
        </div>
    )
}
