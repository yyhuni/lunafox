#!/usr/bin/env node

/**
 * Verify the production Cloudflare Registry Worker after deployment.
 *
 * This gate is intentionally anonymous and digest-only.  It compares the
 * canonical third-party policy with the generated deployment snapshot and the
 * reviewed Worker source before making any network request.  The network
 * evidence is limited to the Registry V2 probe, the current manifest digests,
 * and one deliberately unlisted digest that must be rejected locally.
 */

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

import { canonicalThirdPartyPolicyBytes, validateThirdPartyPolicy } from "./third-party-image-policy.mjs";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_ROOT = path.resolve(SCRIPT_DIR, "../..");
const DEFAULT_POLICY = path.join(DEFAULT_ROOT, "scripts/ci/third-party-image-policy.json");
const DEFAULT_SNAPSHOT = path.join(DEFAULT_ROOT, "dist/final/deployment-snapshot/third-party-image-policy.json");
const DEFAULT_WORKER_SOURCE = path.join(DEFAULT_ROOT, "tools/lunafox-ghcr-registry/src/registry.ts");
const DIGEST_RE = /^sha256:[a-f0-9]{64}$/;
const COMMIT_RE = /^[0-9a-f]{40}$/;
const RELEASE_TAG_RE = /^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/;
const DEFAULT_TIMEOUT_MS = 15_000;
const MAX_TIMEOUT_MS = 120_000;
const USER_AGENT = "lunafox-cloudflare-worker-release-verifier/1";

class VerificationError extends Error {}

function fail(message) {
  throw new VerificationError(message);
}

function parseArgs(argv) {
  const options = {
    policy: DEFAULT_POLICY,
    snapshotPolicy: DEFAULT_SNAPSHOT,
    workerSource: DEFAULT_WORKER_SOURCE,
    registryHost: "docker.lunafox.cc.cd",
    releaseTag: "",
    privateSourceRevision: "",
    publicSourceCommit: "",
    workerVersionId: "",
    deploymentTag: "",
    output: "",
    timeoutMs: DEFAULT_TIMEOUT_MS,
    json: false,
  };
  const valueFlags = new Set([
    "--policy", "--snapshot-policy", "--worker-source", "--registry-host", "--release-tag",
    "--private-source-revision", "--public-source-commit", "--worker-version-id",
    "--deployment-tag", "--output", "--timeout-ms",
  ]);
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--json") { options.json = true; continue; }
    if (arg === "--help" || arg === "-h") {
      process.stdout.write(
        "Usage: verify-cloudflare-worker-release.mjs --release-tag TAG --private-source-revision DIGEST --public-source-commit SHA --worker-version-id ID --deployment-tag TAG [--policy FILE] [--snapshot-policy FILE] [--worker-source FILE] [--registry-host HOST] [--output FILE] [--timeout-ms N] [--json]\n",
      );
      process.exit(0);
    }
    if (!valueFlags.has(arg)) fail(`unknown argument: ${arg}`);
    const value = argv[++index];
    if (!value || value.startsWith("--")) fail(`${arg} requires a value`);
    switch (arg) {
      case "--policy": options.policy = path.resolve(value); break;
      case "--snapshot-policy": options.snapshotPolicy = path.resolve(value); break;
      case "--worker-source": options.workerSource = path.resolve(value); break;
      case "--registry-host": options.registryHost = value; break;
      case "--release-tag": options.releaseTag = value; break;
      case "--private-source-revision": options.privateSourceRevision = value; break;
      case "--public-source-commit": options.publicSourceCommit = value; break;
      case "--worker-version-id": options.workerVersionId = value; break;
      case "--deployment-tag": options.deploymentTag = value; break;
      case "--output": options.output = path.resolve(value); break;
      case "--timeout-ms": options.timeoutMs = Number(value); break;
      default: fail(`unknown argument: ${arg}`);
    }
  }
  for (const [name, value] of [
    ["--release-tag", options.releaseTag],
    ["--private-source-revision", options.privateSourceRevision],
    ["--public-source-commit", options.publicSourceCommit],
    ["--worker-version-id", options.workerVersionId],
    ["--deployment-tag", options.deploymentTag],
  ]) {
    if (!value) fail(`${name} is required`);
  }
  if (!RELEASE_TAG_RE.test(options.releaseTag)) fail("--release-tag is invalid");
  if (!DIGEST_RE.test(options.privateSourceRevision)) fail("--private-source-revision must be a sha256 digest");
  if (!COMMIT_RE.test(options.publicSourceCommit)) fail("--public-source-commit must be a 40-character commit SHA");
  if (!options.workerVersionId || /[\r\n]/.test(options.workerVersionId)) fail("--worker-version-id is invalid");
  if (!/^lunafox-[a-z0-9][a-z0-9.-]*$/.test(options.deploymentTag)) fail("--deployment-tag is invalid");
  if (!Number.isInteger(options.timeoutMs) || options.timeoutMs < 1 || options.timeoutMs > MAX_TIMEOUT_MS) {
    fail(`--timeout-ms must be an integer from 1 to ${MAX_TIMEOUT_MS}`);
  }
  options.registryHost = normalizeRegistryHost(options.registryHost);
  return options;
}

function normalizeRegistryHost(value) {
  if (typeof value !== "string" || value.trim() !== value || value.length === 0) fail("--registry-host is required");
  const candidate = value.includes("://") ? value : `https://${value}`;
  let url;
  try { url = new URL(candidate); } catch { fail("--registry-host must be a hostname or HTTPS URL"); }
  if (url.protocol !== "https:" || url.username || url.password || url.pathname !== "/" || url.search || url.hash) {
    fail("--registry-host must identify an HTTPS host without credentials or a path");
  }
  return url.hostname + (url.port ? `:${url.port}` : "");
}

function readRegular(file, label) {
  let stat;
  try { stat = fs.lstatSync(file); } catch (error) { fail(`cannot read ${label}: ${error.message}`); }
  if (!stat.isFile() || stat.isSymbolicLink()) fail(`${label} must be a regular file`);
  try { return fs.readFileSync(file); } catch (error) { fail(`cannot read ${label}: ${error.message}`); }
}

function readJson(file, label) {
  try { return JSON.parse(readRegular(file, label).toString("utf8")); }
  catch (error) {
    if (error instanceof VerificationError) throw error;
    fail(`cannot parse ${label}: ${error.message}`);
  }
}

function sha256(bytes) {
  return `sha256:${crypto.createHash("sha256").update(bytes).digest("hex")}`;
}

function policyEntries(policy, label) {
  try { return validateThirdPartyPolicy(policy); }
  catch (error) { fail(`${label} is invalid: ${error.message}`); }
}

function canonicalPolicy(file, label) {
  const bytes = readRegular(file, label);
  const policy = readJson(file, label);
  const normalized = policyEntries(policy, label);
  const canonical = canonicalThirdPartyPolicyBytes(normalized);
  if (!bytes.equals(canonical)) fail(`${label} must use canonical third-party policy JSON bytes`);
  return { policy: normalized, bytes, digest: sha256(bytes) };
}

function extractWorkerPolicy(source) {
  // Keep this check deliberately narrow: only the source's explicit
  // repository/digest literals define the third-party closure.  A caller must
  // never be able to add an upstream mapping through a request parameter.
  const entries = [];
  const pattern = /repository:\s*["']([^"']+)["'][\s\S]*?digest:\s*["'](sha256:[a-f0-9]{64})["']/g;
  for (const match of source.matchAll(pattern)) entries.push({ repository: match[1], digest: match[2] });
  if (entries.length === 0) fail("Worker source does not contain an explicit third-party digest closure");
  return entries;
}

function comparePolicyEntries(expected, actual, label) {
  const expectedIdentity = expected.map(({ repository, digest }) => ({ repository, digest }));
  const actualIdentity = actual.map(({ repository, digest }) => ({ repository, digest }));
  if (JSON.stringify(expectedIdentity) !== JSON.stringify(actualIdentity)) {
    fail(`${label} does not match the canonical current third-party digest set`);
  }
}

function staleDigest(entries) {
  const used = new Set(entries.map((entry) => entry.digest));
  const source = entries[0]?.digest ?? "sha256:" + "0".repeat(64);
  const prefix = source.slice(0, -1);
  for (const candidate of ["0", "1", "f", "e"]) {
    const value = prefix + (source.endsWith(candidate) ? "d" : candidate);
    if (!used.has(value)) return value;
  }
  for (let code = 0; code < 16; code += 1) {
    const value = prefix + code.toString(16);
    if (!used.has(value)) return value;
  }
  fail("unable to construct an unlisted stale digest");
}

async function requestWithTimeout(fetchImpl, url, init, timeoutMs) {
  const controller = new AbortController();
  let timedOut = false;
  let timer;
  const timeout = new Promise((_, reject) => {
    timer = setTimeout(() => {
      timedOut = true;
      controller.abort();
      reject(new Error("request timeout"));
    }, timeoutMs);
  });
  try {
    return await Promise.race([
      fetchImpl(url, { ...init, signal: controller.signal }),
      timeout,
    ]);
  } catch (error) {
    if (timedOut) fail(`request timed out after ${timeoutMs}ms: ${url}`);
    fail(`request failed for ${url}: ${error instanceof Error ? error.message : String(error)}`);
  } finally {
    clearTimeout(timer);
  }
}

async function probe(fetchImpl, baseURL, pathName, { expectedStatus, expectedDigest = "", label, timeoutMs }) {
  const url = `${baseURL}${pathName}`;
  const startedAt = Date.now();
  const response = await requestWithTimeout(fetchImpl, url, {
    method: "GET",
    headers: { Accept: "application/vnd.oci.image.manifest.v1+json", "User-Agent": USER_AGENT },
  }, timeoutMs);
  // Consume the body so CI does not leave keep-alive responses pending.
  await response.arrayBuffer();
  const elapsedMs = Math.max(0, Date.now() - startedAt);
  const observedDigest = response.headers.get("Docker-Content-Digest") ?? "";
  const result = { label, method: "GET", path: pathName, status: response.status, dockerContentDigest: observedDigest, elapsedMs };
  if (response.status !== expectedStatus) fail(`${label} expected HTTP ${expectedStatus}, got ${response.status}`);
  if (expectedDigest && observedDigest !== expectedDigest) {
    fail(`${label} returned Docker-Content-Digest ${observedDigest || "<missing>"}, expected ${expectedDigest}`);
  }
  return result;
}

function writeEvidence(file, evidence) {
  if (!file) return;
  fs.mkdirSync(path.dirname(file), { recursive: true });
  const temporary = `${file}.tmp-${process.pid}`;
  fs.writeFileSync(temporary, `${JSON.stringify(evidence, null, 2)}\n`, { mode: 0o644 });
  fs.renameSync(temporary, file);
}

async function verify(options, fetchImpl = globalThis.fetch) {
  if (typeof fetchImpl !== "function") fail("a fetch implementation is required");
  const canonical = canonicalPolicy(options.policy, "canonical third-party policy");
  const snapshot = canonicalPolicy(options.snapshotPolicy, "deployment snapshot third-party policy");
  comparePolicyEntries(canonical.policy.entries, snapshot.policy.entries, "deployment snapshot policy");
  const workerSourceBytes = readRegular(options.workerSource, "Worker source");
  comparePolicyEntries(canonical.policy.entries, extractWorkerPolicy(workerSourceBytes.toString("utf8")), "Worker source policy");

  const baseURL = `https://${options.registryHost}`;
  const current = [];
  current.push(await probe(fetchImpl, baseURL, "/v2/", {
    expectedStatus: 200,
    label: "Registry V2 probe",
    timeoutMs: options.timeoutMs,
  }));
  for (const entry of canonical.policy.entries) {
    current.push(await probe(fetchImpl, baseURL, `/v2/${entry.repository}/manifests/${entry.digest}`, {
      expectedStatus: 200,
      expectedDigest: entry.digest,
      label: `current digest ${entry.repository}`,
      timeoutMs: options.timeoutMs,
    }));
  }
  const stale = staleDigest(canonical.policy.entries);
  const staleProbe = await probe(fetchImpl, baseURL, `/v2/${canonical.policy.entries[0].repository}/manifests/${stale}`, {
    expectedStatus: 403,
    label: "unlisted stale digest",
    timeoutMs: options.timeoutMs,
  });
  const evidence = {
    schemaVersion: 1,
    kind: "lunafox.cloudflare-worker-release-evidence.v1",
    passed: true,
    releaseTag: options.releaseTag,
    privateSourceRevision: options.privateSourceRevision,
    publicSourceCommit: options.publicSourceCommit,
    deploymentTag: options.deploymentTag,
    workerVersionId: options.workerVersionId,
    workerSourceSha256: sha256(workerSourceBytes),
    policySha256: canonical.digest,
    snapshotPolicySha256: snapshot.digest,
    registryHost: options.registryHost,
    staleDigest: { repository: canonical.policy.entries[0].repository, digest: stale, result: staleProbe },
    smoke: { registryV2: current[0], currentDigests: current.slice(1) },
    generatedAt: new Date().toISOString(),
  };
  writeEvidence(options.output, evidence);
  return evidence;
}

async function main() {
  const options = parseArgs(process.argv.slice(2));
  const result = await verify(options);
  process.stdout.write(`${options.json ? JSON.stringify(result, null, 2) : `Cloudflare Worker release verified: ${result.workerVersionId}`}\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
  main().catch((error) => {
    process.stderr.write(`Cloudflare Worker release verification failed: ${error.message}\n`);
    process.exitCode = 1;
  });
}

export {
  comparePolicyEntries,
  extractWorkerPolicy,
  normalizeRegistryHost,
  parseArgs,
  staleDigest,
  verify,
};
