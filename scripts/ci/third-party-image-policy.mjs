#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
export const THIRD_PARTY_POLICY_PATH = path.join(SCRIPT_DIR, "third-party-image-policy.json");
export const THIRD_PARTY_PROVENANCE = "lunafox-reviewed-content";

const POLICY_KEYS = Object.freeze(["entries", "provenanceClaim", "schemaVersion"]);
const ENTRY_KEYS = Object.freeze([
  "contentAudit",
  "digest",
  "evidence",
  "profiles",
  "publisherSignatureVerification",
  "registry",
  "repository",
  "service",
]);
const EXPECTED_ENTRIES = Object.freeze([
  Object.freeze({
    service: "postgres",
    profiles: Object.freeze(["embedded"]),
    registry: "docker.io",
    repository: "library/postgres",
  }),
  Object.freeze({
    service: "redis",
    profiles: Object.freeze(["embedded", "external"]),
    registry: "docker.io",
    repository: "library/redis",
  }),
  Object.freeze({
    service: "loki",
    profiles: Object.freeze(["embedded", "external"]),
    registry: "docker.io",
    repository: "grafana/loki",
  }),
  Object.freeze({
    service: "alloy",
    profiles: Object.freeze(["embedded", "external"]),
    registry: "docker.io",
    repository: "grafana/alloy",
  }),
]);
const EXPECTED_BY_SERVICE = new Map(EXPECTED_ENTRIES.map((entry) => [entry.service, entry]));
const BOUNDED_EVIDENCE = /^release-frozen [A-Za-z][A-Za-z0-9 -]* digest reviewed by LunaFox$/;
const UNBOUNDED_EVIDENCE = /\b(?:authenticated|authentication|cosign|publisher|signature|signed|signer|verified|verification)\b/i;

function fail(message) {
  throw new Error(message);
}

function hasExactKeys(value, expected, label) {
  const actual = Object.keys(value).sort();
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    fail(`${label} has unknown or missing fields`);
  }
}

function requiredString(entry, index, key) {
  const value = entry[key];
  if (typeof value !== "string" || value.trim() !== value || value.length === 0) {
    fail(`third-party policy entry ${index}.${key} is required`);
  }
  return value;
}

function validateProfiles(profiles, expected, service) {
  if (!Array.isArray(profiles) || JSON.stringify(profiles) !== JSON.stringify(expected)) {
    fail(`third-party policy profiles do not match the ${service} service closure`);
  }
}

export function validateThirdPartyPolicy(policy) {
  if (!policy || typeof policy !== "object" || Array.isArray(policy)) {
    fail("third-party image policy must be an object");
  }
  hasExactKeys(policy, POLICY_KEYS, "third-party image policy");
  if (policy.schemaVersion !== 1) fail("third-party image policy schemaVersion must be 1");
  if (policy.provenanceClaim !== THIRD_PARTY_PROVENANCE) {
    fail("third-party image policy provenance claim is invalid");
  }
  if (!Array.isArray(policy.entries) || policy.entries.length !== EXPECTED_ENTRIES.length) {
    fail("third-party image policy must contain the complete fixed service closure");
  }

  const normalizedEntries = [];
  const seen = new Set();
  for (const [index, entry] of policy.entries.entries()) {
    if (!entry || typeof entry !== "object" || Array.isArray(entry)) {
      fail(`third-party policy entry ${index} must be an object`);
    }
    hasExactKeys(entry, ENTRY_KEYS, `third-party policy entry ${index}`);

    const service = requiredString(entry, index, "service");
    const expected = EXPECTED_BY_SERVICE.get(service);
    if (!expected || seen.has(service)) {
      fail(`third-party policy service is duplicated or unsupported: ${service}`);
    }
    if (service !== EXPECTED_ENTRIES[index].service) {
      fail("third-party policy entries must use the canonical service order");
    }
    seen.add(service);

    validateProfiles(entry.profiles, expected.profiles, service);
    const registry = requiredString(entry, index, "registry");
    const repository = requiredString(entry, index, "repository");
    const digest = requiredString(entry, index, "digest");
    const contentAudit = requiredString(entry, index, "contentAudit");
    const publisherSignatureVerification = requiredString(entry, index, "publisherSignatureVerification");
    const evidence = requiredString(entry, index, "evidence");

    if (registry !== expected.registry || repository !== expected.repository) {
      fail(`third-party policy upstream identity is invalid: ${service}`);
    }
    if (!/^sha256:[a-f0-9]{64}$/.test(digest)) {
      fail(`third-party policy digest is not immutable: ${service}`);
    }
    if (contentAudit !== "reviewed") {
      fail(`third-party policy content audit is not reviewed: ${service}`);
    }
    if (publisherSignatureVerification !== "not-approved") {
      fail(`third-party policy publisher claim is too strong: ${service}`);
    }
    if (!BOUNDED_EVIDENCE.test(evidence) || UNBOUNDED_EVIDENCE.test(evidence)) {
      fail(`third-party policy evidence is not a bounded LunaFox review claim: ${service}`);
    }

    normalizedEntries.push({
      service,
      profiles: [...entry.profiles],
      registry,
      repository,
      digest,
      contentAudit,
      publisherSignatureVerification,
      evidence,
    });
  }

  if (seen.size !== EXPECTED_ENTRIES.length) {
    fail("third-party image policy is missing a required service");
  }
  return {
    schemaVersion: 1,
    provenanceClaim: THIRD_PARTY_PROVENANCE,
    entries: normalizedEntries,
  };
}

export function canonicalThirdPartyPolicyBytes(policy) {
  return Buffer.from(`${JSON.stringify(validateThirdPartyPolicy(policy), null, 2)}\n`);
}

export function readThirdPartyPolicy(filePath = THIRD_PARTY_POLICY_PATH) {
  let bytes;
  try {
    const stat = fs.lstatSync(filePath);
    if (!stat.isFile() || stat.isSymbolicLink()) {
      fail("third-party image policy must be a regular file");
    }
    bytes = fs.readFileSync(filePath);
  } catch (error) {
    if (error instanceof Error && error.message.startsWith("third-party image policy")) throw error;
    fail(`third-party image policy cannot be read: ${error.message}`);
  }

  let normalized;
  try {
    normalized = validateThirdPartyPolicy(JSON.parse(bytes.toString("utf8")));
  } catch (error) {
    if (error instanceof Error && error.message.startsWith("third-party")) throw error;
    fail(`third-party image policy is not valid JSON: ${error.message}`);
  }
  if (!bytes.equals(canonicalThirdPartyPolicyBytes(normalized))) {
    fail("third-party image policy must use canonical JSON bytes");
  }
  return normalized;
}

function serviceBlock(compose, service) {
  return new RegExp(`(^|\\n)  ${service}:([\\s\\S]*?)(?=\\n  [a-zA-Z0-9_-]+:|$)`).exec(compose)?.[2] ?? "";
}

export function validatePolicyAgainstComposeTemplate(policy, compose) {
  const normalized = validateThirdPartyPolicy(policy);
  if (typeof compose !== "string" || compose.length === 0) {
    fail("Compose content is required for third-party policy validation");
  }
  for (const entry of normalized.entries) {
    const image = `${entry.registry}/${entry.repository}@${entry.digest}`;
    if (!serviceBlock(compose, entry.service).includes(`image: ${image}`)) {
      fail(`Compose template digest does not match third-party policy for ${entry.service}`);
    }
  }
  for (const profile of ["embedded", "external"]) policyForProfile(normalized, profile);
  return normalized;
}

export function policyForProfile(policy, profile) {
  const normalized = validateThirdPartyPolicy(policy);
  if (profile !== "embedded" && profile !== "external") {
    fail(`unsupported database profile: ${profile}`);
  }
  return normalized.entries.filter((entry) => entry.profiles.includes(profile));
}

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
  try {
    const policy = readThirdPartyPolicy(process.argv[2] ? path.resolve(process.argv[2]) : THIRD_PARTY_POLICY_PATH);
    process.stdout.write(canonicalThirdPartyPolicyBytes(policy));
  } catch (error) {
    process.stderr.write(`Third-party image policy validation failed: ${error.message}\n`);
    process.exitCode = 1;
  }
}
