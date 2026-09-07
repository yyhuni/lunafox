/**
 * Returns the first Unicode code point from a required organization name.
 *
 * Organization names are validated as required input upstream. Throwing here
 * keeps malformed records visible instead of silently inventing an identity.
 */
export function getOrganizationInitial(name: string): string {
  const trimmedName = name.trim()
  const initial = Array.from(trimmedName)[0]

  if (!initial) {
    throw new Error("Organization name is required to render its identity marker.")
  }

  return initial
}
