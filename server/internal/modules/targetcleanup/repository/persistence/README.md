# targetcleanup/repository/persistence

This directory contains the GORM mapping for the durable internal
`target_cleanup_job` table. The mapping must contain only Job identity, status,
retry diagnostics, timestamps, and completion time. Do not add a phase,
deletion cursor, worker owner, lease expiry, heartbeat, or claim field without
an approved multi-Worker topology change.
