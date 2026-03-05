# Subdomain Discovery Configuration Reference

<!-- Source: worker/internal/workflow/subdomain_discovery/contract_definition.go -->

- Workflow: `subdomain_discovery`
- Target Types: `domain`

## Stages

### `recon` - Reconnaissance

Collect subdomains from multiple data sources without directly probing the target

- Required Stage: `true`
- Parallel Execution: `true`

#### Tool `subfinder`

Collect subdomains using multiple data sources (Shodan, Censys, VirusTotal, etc.)

| Parameter | Type | Required (when enabled=true) | Description |
| --- | --- | --- | --- |
| `enabled` | `boolean` | `yes` | Whether to enable this tool |
| `timeout-runtime` | `integer` | `yes` | Scan timeout in seconds |
| `threads-cli` | `integer` | `yes` | Number of concurrent threads |

### `bruteforce` - Dictionary Bruteforce

Bruteforce domains using dictionaries to discover unpublished subdomains

- Required Stage: `false`
- Parallel Execution: `false`

#### Tool `subdomain-bruteforce`

DNS bruteforce domains using dictionaries

| Parameter | Type | Required (when enabled=true) | Description |
| --- | --- | --- | --- |
| `enabled` | `boolean` | `yes` | Whether to enable this tool |
| `timeout-runtime` | `integer` | `yes` | Scan timeout in seconds |
| `subdomain-wordlist-name-runtime` | `string` | `yes` | Subdomain wordlist name (stored on server) |
| `threads-cli` | `integer` | `yes` | Number of concurrent threads |
| `rate-limit-cli` | `integer` | `yes` | Rate limit per second |
| `wildcard-tests-cli` | `integer` | `yes` | Number of wildcard detection tests |
| `wildcard-batch-cli` | `integer` | `yes` | Wildcard batch processing size |

### `permutation` - Permutation

Generate new possible subdomains by permuting discovered subdomains

- Required Stage: `false`
- Parallel Execution: `false`
- Notes:
  - Permutation performs a wildcard sampling check before execution and may skip if wildcard is detected.

#### Tool `subdomain-permutation-resolve`

Generate new possible subdomains by permuting discovered subdomains and resolve them

| Parameter | Type | Required (when enabled=true) | Description |
| --- | --- | --- | --- |
| `enabled` | `boolean` | `yes` | Whether to enable this tool |
| `timeout-runtime` | `integer` | `yes` | Scan timeout in seconds |
| `wildcard-sample-timeout-runtime` | `integer` | `yes` | Wildcard sample timeout in seconds |
| `wildcard-sample-multiplier-runtime` | `integer` | `yes` | Wildcard sample size multiplier |
| `wildcard-expansion-threshold-runtime` | `integer` | `yes` | Wildcard expansion ratio threshold |
| `threads-cli` | `integer` | `yes` | Number of concurrent threads |
| `rate-limit-cli` | `integer` | `yes` | Rate limit per second |
| `wildcard-tests-cli` | `integer` | `yes` | Number of wildcard detection tests |
| `wildcard-batch-cli` | `integer` | `yes` | Wildcard batch processing size |

### `resolve` - Resolution

Resolve all discovered subdomains to IP addresses and verify they are alive

- Required Stage: `false`
- Parallel Execution: `false`

#### Tool `subdomain-resolve`

Resolve subdomain list to verify their validity

| Parameter | Type | Required (when enabled=true) | Description |
| --- | --- | --- | --- |
| `enabled` | `boolean` | `yes` | Whether to enable this tool |
| `timeout-runtime` | `integer` | `yes` | Scan timeout in seconds |
| `threads-cli` | `integer` | `yes` | Number of concurrent threads |
| `rate-limit-cli` | `integer` | `yes` | Rate limit per second |

