# Geolocation

This package is the fixed, process-local FreeIPAPI adapter and resource
coordinator shared by Agent target-IP lookups and the current single Server's
no-target self lookup.

## Provider Boundary

- Production uses only `https://free.freeipapi.com/api/json` and
  `https://free.freeipapi.com/api/json/{ip_address}` with
  `providerKey=freeipapi`. There are no credentials, configurable provider
  schemas, MaxMind/MMDB fallback, or administrator override.
- One logical lookup makes at most one HTTPS attempt with a 3-second end-to-end
  context, a 64 KiB body limit, strict JSON decoding, and HTTPS-only redirects.
- Only normalized public IP, finite in-range WGS84 latitude/longitude, nullable
  non-negative `accuracyRadiusKm`, provider key, and UTC `resolvedAt` leave the
  adapter. Raw responses and administrative/network labels are not persisted.

## Coordinator Bounds

- Waiting work is a true 128-entry FIFO excluding active requests; admission is
  drop-new and non-blocking.
- At most 4 provider requests run concurrently, with no more than 60 actual
  attempts in any rolling minute.
- The normalized-IP cache holds at most 1024 combined success and provider
  negative entries. Success expires absolutely at `resolvedAt + 7 days`;
  provider negatives expire after 5 minutes. Expired entries are removed before
  valid LRU entries, and reads never extend either TTL.
- Target-IP work is single-flight per normalized IP. A provider `429` also
  applies the later `Retry-After` or a 60-second default process-wide cooldown.
- Queue, rate, cooldown, cache, and flight state are memory-only. Shutdown closes
  admission, discards queued work, cancels active requests, and preserves all
  database snapshots.

These limits coordinate one Server process only. Multi-Server quota, cache,
flight, or singleton ownership is unsupported and requires a separate design.
Lookup submission and provider failure are best-effort enhancement failures:
they never fail Agent connection, heartbeat, claim, or Scan paths. Persisted
last-success snapshots remain readable when expired or when refresh fails;
without a prior success the location remains unknown. Management APIs expose
only current/expired/unknown success state, not last-attempt diagnostics.
