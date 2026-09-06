#!/usr/bin/env node

/** Validate the immutable public Engine handoff consumed by private release. */

import fs from "node:fs";
import path from "node:path";
import { pathToFileURL } from "node:url";

const PUBLIC_REPOSITORY = "yyhuni/lunafox";
const PUBLIC_SIGNER_RE = /^https:\/\/github\.com\/yyhuni\/lunafox\/.github\/workflows\/public-validate\.yml@refs\/heads\/main$/;
const DIGEST_RE = /^sha256:[a-f0-9]{64}$/;

function fail(message) { throw new Error(message); }

function parseArgs(argv) {
  const options = { manifest: "", runtime: "", packages: "", packageDigests: "", runtimeEvidence: "", packageEvidence: "", tag: "", merge: "", source: "", exportManifest: "", provenance: "", workflowIdentity: "", json: false };
  const valueFlags = new Set([
    "--manifest", "--runtime-build-results", "--package-build-results", "--package-digests-dir",
    "--runtime-evidence", "--package-evidence", "--tag", "--public-merge-commit",
    "--source-revision-digest", "--export-manifest-sha256", "--public-provenance-sha256",
    "--workflow-identity",
  ]);
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--json") { options.json = true; continue; }
    if (arg === "--help" || arg === "-h") { process.stdout.write("Usage: verify-public-engine-release.mjs --manifest FILE --runtime-build-results FILE --package-build-results FILE --package-digests-dir DIR --runtime-evidence FILE --package-evidence FILE --tag TAG --public-merge-commit SHA --source-revision-digest DIGEST --export-manifest-sha256 DIGEST --public-provenance-sha256 DIGEST --workflow-identity ID\n"); process.exit(0); }
    if (!valueFlags.has(arg)) fail(`unknown argument: ${arg}`);
    const value = argv[++index];
    if (!value || value.startsWith("--")) fail(`${arg} requires a value`);
    const key = {
      "--manifest": "manifest", "--runtime-build-results": "runtime", "--package-build-results": "packages",
      "--package-digests-dir": "packageDigests", "--runtime-evidence": "runtimeEvidence", "--package-evidence": "packageEvidence",
      "--tag": "tag", "--public-merge-commit": "merge", "--source-revision-digest": "source",
      "--export-manifest-sha256": "exportManifest", "--public-provenance-sha256": "provenance", "--workflow-identity": "workflowIdentity",
    }[arg];
    options[key] = value;
  }
  return options;
}

function readJson(file, label) {
  try { return JSON.parse(fs.readFileSync(file, "utf8")); }
  catch (error) { fail(`cannot read ${label} ${file}: ${error.message}`); }
}

function requireFile(file, label) {
  if (!file || !fs.statSync(file, { throwIfNoEntry: false })?.isFile()) fail(`${label} is required: ${file}`);
}

function requireDigest(value, label) {
  if (!DIGEST_RE.test(value ?? "")) fail(`${label} must be an immutable sha256 digest`);
}

function requireDigestRef(value, registry, label) {
  if (typeof value !== "string" || !value.startsWith(`${registry}/`) || !/@sha256:[a-f0-9]{64}$/.test(value)) {
    fail(`${label} must be a ${registry} digest reference`);
  }
}

function sortedUnique(values, label) {
  if (!Array.isArray(values) || values.length === 0 || new Set(values).size !== values.length) fail(`${label} must be a non-empty unique array`);
  const sorted = [...values].sort();
  if (JSON.stringify(values) !== JSON.stringify(sorted)) fail(`${label} must be canonically sorted`);
  return new Set(values);
}

function validateReceipt(receipt, schema, key, label) {
  if (receipt?.schemaVersion !== schema || receipt?.mode !== "production") fail(`${label} has an invalid schema or mode`);
  const items = receipt[key];
  if (!Array.isArray(items) || items.length === 0) fail(`${label} must contain ${key}`);
  const ids = items.map((item) => item.engineId);
  sortedUnique(ids, `${label}.${key}.engineId`);
  return items;
}

function validateEvidence(file, expectedSchema, signer, label, binding) {
  const evidence = readJson(file, label);
  if (evidence.schemaVersion !== expectedSchema || evidence.passed !== true) fail(`${label} did not pass`);
  if (evidence.signerIdentity !== signer) fail(`${label} signer identity mismatch`);
  for (const [field, expected] of Object.entries(binding)) {
    if (evidence[field] !== expected) fail(`${label} ${field} binding mismatch`);
  }
  return evidence;
}

function readPackageDigestFiles(directory) {
  if (!directory || !fs.statSync(directory, { throwIfNoEntry: false })?.isDirectory()) fail(`package digest directory is required: ${directory}`);
  const files = fs.readdirSync(directory).filter((name) => name.endsWith(".env")).sort();
  if (files.length === 0) fail("public Engine package digest directory is empty");
  const result = new Map();
  for (const name of files) {
    const payload = fs.readFileSync(path.join(directory, name), "utf8");
    const values = Object.fromEntries(payload.split(/\r?\n/).filter(Boolean).map((line) => {
      const separator = line.indexOf("=");
      return separator > 0 ? [line.slice(0, separator), line.slice(separator + 1)] : [line, ""];
    }));
    if (!values.ENGINE_ID || result.has(values.ENGINE_ID)) fail(`invalid or duplicate Engine package receipt: ${name}`);
    const refs = (values.ENGINE_REFS ?? "").split(",");
    if (refs.length !== 2) fail(`Engine package receipt must contain two refs: ${name}`);
    requireDigestRef(refs[0], "docker.io", `${name} Docker Hub ref`);
    requireDigestRef(refs[1], "ghcr.io", `${name} GHCR ref`);
    if (refs[0].split("@")[1] !== refs[1].split("@")[1]) fail(`Engine package registry digest drift: ${name}`);
    result.set(values.ENGINE_ID, refs);
  }
  return result;
}

function validate(options) {
  for (const [file, label] of [[options.manifest, "Engine manifest"], [options.runtime, "Runtime receipt"], [options.packages, "Package receipt"], [options.runtimeEvidence, "Runtime evidence"], [options.packageEvidence, "Package evidence"]]) requireFile(file, label);
  const manifest = readJson(options.manifest, "Engine manifest");
  const runtime = readJson(options.runtime, "Runtime receipt");
  const packages = readJson(options.packages, "Package receipt");
  const runtimeItems = validateReceipt(runtime, "lunafox.engine-runtime-image-build-results.v1", "engines", "Runtime receipt");
  const packageItems = validateReceipt(packages, "lunafox.engine-package-build-results.v1", "packages", "Package receipt");
  const packageRefs = readPackageDigestFiles(options.packageDigests);
  const signer = options.workflowIdentity || manifest.signerIdentity;
  if (!PUBLIC_SIGNER_RE.test(signer)) fail("Engine signer must be the protected public-validate.yml identity");
  if (manifest.schemaVersion !== "lunafox.engine-release-manifest.v1") fail("unsupported public Engine manifest schema");
  if (manifest.repository !== PUBLIC_REPOSITORY || manifest.releaseTag !== options.tag || manifest.publicMergeCommit !== options.merge || manifest.sourceRevisionDigest !== options.source || manifest.publicExportManifestSha256 !== options.exportManifest || manifest.publicProvenanceSha256 !== options.provenance || manifest.signerIdentity !== signer) fail("public Engine manifest provenance binding mismatch");
  if (!/^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(options.tag) || !/^[0-9a-f]{40}$/.test(options.merge)) fail("release tag or public merge commit is invalid");
  for (const [value, label] of [[options.source, "source revision"], [options.exportManifest, "export manifest"], [options.provenance, "public provenance"]]) requireDigest(value, label);
  const evidenceBinding = {
    publicMergeCommit: options.merge,
    sourceRevisionDigest: options.source,
    releaseTag: options.tag,
    publicExportManifestSha256: options.exportManifest,
    publicProvenanceSha256: options.provenance,
  };
  const runtimeEvidence = validateEvidence(options.runtimeEvidence, "lunafox.engine-runtime-image-publication-evidence.v1", signer, "Runtime evidence", evidenceBinding);
  const packageEvidence = validateEvidence(options.packageEvidence, "lunafox.engine-package-v2-publication-evidence.v1", signer, "Package evidence", evidenceBinding);
  if (!Array.isArray(manifest.engines) || manifest.engines.length !== runtimeItems.length || manifest.engines.length !== packageItems.length) fail("public Engine manifest engine count does not match receipts");
  const runtimeById = new Map(runtimeItems.map((item) => [item.engineId, item]));
  const packageById = new Map(packageItems.map((item) => [item.engineId, item]));
  const manifestIds = manifest.engines.map((item) => item.engineId);
  sortedUnique(manifestIds, "public Engine manifest engineId");
  for (const item of manifest.engines) {
    const runtimeItem = runtimeById.get(item.engineId);
    const packageItem = packageById.get(item.engineId);
    const refs = packageRefs.get(item.engineId);
    if (!runtimeItem || !packageItem || !refs) fail(`public Engine manifest is missing receipt ${item.engineId}`);
    requireDigest(item.runtimeImageDigest, `${item.engineId} Runtime Image digest`);
    requireDigest(item.packageDigest, `${item.engineId} Package digest`);
    requireDigest(item.packageArtifactManifestDigest, `${item.engineId} Package OCI manifest digest`);
    if (item.runtimeImageDigest !== runtimeItem.indexDigest || item.packageRuntimeImageDigest !== runtimeItem.indexDigest || item.packageDigest !== packageItem.packageDigest) fail(`public Engine digest binding mismatch: ${item.engineId}`);
    if (JSON.stringify(item.runtimeImageRefs) !== JSON.stringify(runtimeItem.refs) || JSON.stringify(item.packageRefs) !== JSON.stringify(refs)) fail(`public Engine registry refs mismatch: ${item.engineId}`);
    requireDigestRef(item.runtimeImageRefs[0], "docker.io", `${item.engineId} Runtime Docker Hub ref`);
    requireDigestRef(item.runtimeImageRefs[1], "ghcr.io", `${item.engineId} Runtime GHCR ref`);
    requireDigestRef(item.packageRefs[0], "docker.io", `${item.engineId} Package Docker Hub ref`);
    requireDigestRef(item.packageRefs[1], "ghcr.io", `${item.engineId} Package GHCR ref`);
    if (item.packageRefs[0].split("@")[1] !== item.packageArtifactManifestDigest || item.packageRefs[1].split("@")[1] !== item.packageArtifactManifestDigest) {
      fail(`public Engine Package OCI manifest digest mismatch: ${item.engineId}`);
    }
    const packageRecord = packageEvidence.packages?.find((record) => record.engineId === item.engineId);
    if (!packageRecord || packageRecord.packageDigest !== item.packageDigest || packageRecord.artifactManifestDigest !== item.packageArtifactManifestDigest) {
      fail(`public Engine package evidence binding mismatch: ${item.engineId}`);
    }
    const runtimeRecord = runtimeEvidence.engines?.find((record) => record.engineId === item.engineId);
    if (!runtimeRecord || runtimeRecord.indexDigest !== item.runtimeImageDigest || JSON.stringify(runtimeRecord.refs) !== JSON.stringify(item.runtimeImageRefs)) {
      fail(`public Engine Runtime evidence binding mismatch: ${item.engineId}`);
    }
  }
  return { schemaVersion: 1, passed: true, repository: PUBLIC_REPOSITORY, signerIdentity: signer, releaseTag: options.tag, engineCount: manifest.engines.length };
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const result = validate(options);
  process.stdout.write(`${options.json ? JSON.stringify(result, null, 2) : "public Engine release verified"}\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try { main(); } catch (error) { process.stderr.write(`public Engine release verification failed: ${error.message}\n`); process.exitCode = 1; }
}

export { validate, parseArgs };
