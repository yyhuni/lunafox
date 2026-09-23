#!/usr/bin/env node

/**
 * Verify the byte-level release evidence closure used to grant component reuse.
 *
 * The manifest binds a composition's canonical core digest, while release
 * provenance binds the published JSON asset bytes. Keeping both checks here
 * prevents an otherwise valid composition from being substituted after its
 * manifest was generated.
 */

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

import { verify as verifyComponentEvidence } from "./verify-component-evidence-bundle.mjs";
import { verify as verifyComposition } from "./verify-release-component-composition.mjs";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_POLICY = path.join(SCRIPT_DIR, "public-release-policy.json");
const DIGEST_RE = /^sha256:[a-f0-9]{64}$/;
const RELEASE_TAG_RE = /^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/;
const COMMIT_RE = /^[0-9a-f]{40}$/;
const PUBLIC_WORKFLOW = "https://github.com/yyhuni/lunafox/.github/workflows/public-validate.yml@refs/heads/main";

function fail(message) { throw new Error(message); }
function assert(condition, message) { if (!condition) fail(message); }
function isPlainObject(value) { return value !== null && typeof value === "object" && !Array.isArray(value); }

function parseArgs(argv) {
  const options = {
    manifest: "", composition: "", bundle: "", provenance: "", policy: DEFAULT_POLICY, tag: "", releaseProfile: "", json: false,
  };
  const flags = new Set(["--manifest", "--composition", "--bundle", "--provenance", "--policy", "--tag", "--release-profile"]);
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--json") { options.json = true; continue; }
    if (arg === "--help" || arg === "-h") {
      process.stdout.write("Usage: verify-public-release-evidence.mjs --manifest FILE --composition FILE --bundle FILE --provenance FILE --tag TAG [--policy FILE] [--release-profile <modern|alpha164-bridge>] [--json]\n");
      process.exit(0);
    }
    if (!flags.has(arg)) fail(`unknown argument: ${arg}`);
    const value = argv[++index];
    if (!value || value.startsWith("--")) fail(`${arg} requires a value`);
    if (arg === "--manifest") options.manifest = path.resolve(value);
    else if (arg === "--composition") options.composition = path.resolve(value);
    else if (arg === "--bundle") options.bundle = path.resolve(value);
    else if (arg === "--provenance") options.provenance = path.resolve(value);
    else if (arg === "--policy") options.policy = path.resolve(value);
    else if (arg === "--tag") options.tag = value;
    else options.releaseProfile = value;
  }
  for (const key of ["manifest", "composition", "bundle", "provenance", "tag"]) assert(options[key], `--${key} is required`);
  assert(RELEASE_TAG_RE.test(options.tag), "--tag is invalid");
  return options;
}

function readRegular(file, label) {
  let info;
  try { info = fs.lstatSync(file); }
  catch (error) { fail(`cannot read ${label}: ${error.message}`); }
  assert(info.isFile() && !info.isSymbolicLink(), `${label} must be a regular file`);
  try { return fs.readFileSync(file); }
  catch (error) { fail(`cannot read ${label}: ${error.message}`); }
}

function sha256(bytes) {
  return `sha256:${crypto.createHash("sha256").update(bytes).digest("hex")}`;
}

function readJson(file, label) {
  try { return JSON.parse(readRegular(file, label).toString("utf8")); }
  catch (error) { fail(`cannot parse ${label}: ${error.message}`); }
}

function assertAllowedKeys(value, expected, label) {
  assert(isPlainObject(value), `${label} must be an object`);
  const keys = Object.keys(value).sort();
  const want = [...expected].sort();
  assert(JSON.stringify(keys) === JSON.stringify(want), `${label} has an unexpected field set`);
}

function validateReleaseProvenance(value, facts) {
  assertAllowedKeys(value, new Set([
    "schemaVersion", "status", "releaseTag", "artifactDigest", "runtimeComposition", "componentEvidence",
    "privateSourceRevision", "publicSourceCommit", "deploymentSnapshotCommit", "workflowIdentity", "builderRun",
    "exportManifest", "publicProvenance", "sbom", "scanEvidence",
  ]), "public release provenance");
  assert(value.schemaVersion === 1 && value.status === "published", "public release provenance schema is invalid");
  assert(value.releaseTag === facts.tag, "public release provenance tag does not match the release");
  assert(value.artifactDigest === facts.manifestSHA256, "public release provenance manifest digest does not match bytes");
  assert(DIGEST_RE.test(value.privateSourceRevision), "public release provenance private source revision is invalid");
  assert(COMMIT_RE.test(value.publicSourceCommit), "public release provenance public source commit is invalid");
  assert(COMMIT_RE.test(value.deploymentSnapshotCommit), "public release provenance deployment snapshot commit is invalid");
  assert(value.workflowIdentity === PUBLIC_WORKFLOW, "public release provenance workflow identity is invalid");
  assert(typeof value.builderRun === "string" && /^https:\/\/github\.com\/yyhuni\/lunafox\/actions\/runs\/\d+$/.test(value.builderRun), "public release provenance builder run is invalid");
  assert(DIGEST_RE.test(value.exportManifest) && DIGEST_RE.test(value.publicProvenance), "public release provenance source bindings are invalid");
  assert(value.sbom === true, "public release provenance must retain SBOM evidence");
  assertAllowedKeys(value.scanEvidence, new Set(["runtimeImages", "engineHandoff", "signatures"]), "public release provenance scanEvidence");
  assert(value.scanEvidence.runtimeImages === true && value.scanEvidence.engineHandoff === true && value.scanEvidence.signatures === true,
    "public release provenance scan evidence is incomplete");
  assertAllowedKeys(value.runtimeComposition, new Set(["asset", "digest", "assetSha256"]), "public release provenance runtimeComposition");
  assert(value.runtimeComposition.asset === "runtime-composition.json", "public release provenance composition asset is invalid");
  assert(value.runtimeComposition.digest === facts.compositionDigest, "public release provenance composition digest does not match canonical composition");
  assert(value.runtimeComposition.assetSha256 === facts.compositionSHA256, "public release provenance composition asset digest does not match bytes");
  assertAllowedKeys(value.componentEvidence, new Set(["asset", "assetSha256"]), "public release provenance componentEvidence");
  assert(value.componentEvidence.asset === "component-evidence.json", "public release provenance component evidence asset is invalid");
  assert(value.componentEvidence.assetSha256 === facts.bundleSHA256, "public release provenance component evidence digest does not match bytes");
}

function verify(options) {
  const composition = readJson(options.composition, "runtime composition");
  const bundle = readJson(options.bundle, "component evidence bundle");
  const provenance = readJson(options.provenance, "public release provenance");
  const manifestBytes = readRegular(options.manifest, "release manifest");
  const compositionBytes = readRegular(options.composition, "runtime composition");
  const bundleBytes = readRegular(options.bundle, "component evidence bundle");
  const compositionResult = verifyComposition({ composition: options.composition, manifest: options.manifest, policy: options.policy, releaseProfile: options.releaseProfile });
  const evidenceResult = verifyComponentEvidence({ bundle: options.bundle, composition: options.composition, manifest: options.manifest, policy: options.policy, releaseProfile: options.releaseProfile });
  assert(composition.releaseTag === options.tag && bundle.releaseTag === options.tag, "release evidence tag does not match the release");
  validateReleaseProvenance(provenance, {
    tag: options.tag,
    manifestSHA256: sha256(manifestBytes),
    compositionDigest: composition.compositionDigest,
    compositionSHA256: sha256(compositionBytes),
    bundleSHA256: sha256(bundleBytes),
  });
  return {
    schemaVersion: 1,
    passed: true,
    releaseTag: options.tag,
    manifestDigest: sha256(manifestBytes),
    compositionDigest: compositionResult.compositionDigest,
    componentCount: evidenceResult.componentCount,
  };
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const result = verify(options);
  process.stdout.write(`${options.json ? JSON.stringify(result, null, 2) : `public release evidence verified: ${result.releaseTag}`}\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
  try { main(); }
  catch (error) { process.stderr.write(`public release evidence verification failed: ${error.message}\n`); process.exitCode = 1; }
}

export { parseArgs, validateReleaseProvenance, verify };
